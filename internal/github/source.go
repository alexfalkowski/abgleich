package github

import (
	"crypto/sha1" // #nosec G505 -- Git blob identifiers require SHA-1; HTTPS authenticates the source.
	"fmt"
	"strconv"
	"unicode/utf8"

	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/io"
	"github.com/alexfalkowski/go-service/v2/net/http"
	"github.com/alexfalkowski/go-service/v2/net/http/media"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/google/go-github/v93/github"
)

// TreeBlob identifies a blob entry; its mode still determines whether it is a regular file.
const TreeBlob = "blob"

// TreeDirectory identifies a directory entry in a repository tree.
const TreeDirectory = "tree"

const (
	modeRegular    = "100644"
	modeExecutable = "100755"
	blobType       = "Blob"
)

// ReadSourceFiles retrieves regular-file text at revision in one GraphQL query
// without automatic retries. Binary blobs are omitted; missing or inconsistent
// repository identities, text, byte sizes, or blob hashes reject the whole group.
func (s *service) ReadSourceFiles(ctx context.Context, repo *Repository, revision string, files []*TreeEntry) (map[string]string, error) {
	payload, err := newSourceFilesRequest(repo, revision, files)
	if err != nil {
		return nil, err
	}

	request, err := s.client.installation.NewRequest(ctx, http.MethodPost, "graphql", payload)
	if err != nil {
		return nil, errors.New("github: source request unavailable")
	}

	request.Header.Set(http.AcceptKey, media.JSON)

	var response sourceFilesResponse
	if _, err := s.client.installation.Do(request, &response); err != nil {
		return nil, errors.New("github: source unavailable")
	}

	return response.files(repo, files)
}

// ListSourceFiles reads a complete recursive tree at revision, preserving
// nonregular entries as metadata. Truncated trees, invalid or duplicate paths,
// and missing regular-file blob identities return an error.
func (s *service) ListSourceFiles(ctx context.Context, repo *Repository, revision string) ([]*TreeEntry, error) {
	if strings.IsEmpty(revision) {
		return nil, errors.New("github: source revision missing")
	}

	tree, _, err := s.client.installation.Git.GetTree(ctx, repo.GetOwner().GetLogin(), repo.GetName(), revision, true)
	if err != nil {
		return nil, errors.New("github: source tree unavailable")
	}

	return sourceFiles(tree)
}

// IsRegularSourceFile reports whether an entry identifies a regular-file blob.
// Directories, symlinks, and submodules are not followed or read as source files.
func IsRegularSourceFile(entry *TreeEntry) bool {
	return entry.GetType() == TreeBlob && (entry.GetMode() == modeRegular || entry.GetMode() == modeExecutable)
}

func newSourceFilesRequest(repo *Repository, revision string, files []*TreeEntry) (*sourceFilesRequest, error) {
	if err := validateSourceRequest(repo, revision, files); err != nil {
		return nil, err
	}

	variables := map[string]string{"owner": repo.GetOwner().GetLogin(), "repository": repo.GetName()}
	parameters := []string{"$owner: String!", "$repository: String!"}

	var fields []string
	for index, file := range files {
		alias := "file" + strconv.Itoa(index)
		variables[alias] = revision + ":" + file.GetPath()
		parameters = append(parameters, "$"+alias+": String!")
		fields = append(fields, fmt.Sprintf(`%s: repository(owner: $owner, name: $repository) {
  node_id: id full_name: nameWithOwner
  object(expression: $%s) {
    type: __typename
    ... on Blob { content: text size: byteSize sha: oid binary: isBinary truncated: isTruncated }
  }
}`, alias, alias))
	}

	return &sourceFilesRequest{
		Variables: variables,
		Query:     strings.Concat("query SourceFiles(", strings.Join(", ", parameters...), ") {\n", strings.Join("\n", fields...), "\n}"),
	}, nil
}

// Source queries bind revision/path expressions as variables, never GraphQL text.
type sourceFilesRequest struct {
	Variables map[string]string `json:"variables"`
	Query     string            `json:"query"`
}

type sourceRepository struct {
	Repository

	Object *sourceBlob `json:"object"`
}

// GraphQL adds binary/truncation flags to the SDK's pointer-valued content fields.
type sourceBlob struct {
	RepositoryContent

	Binary    *bool `json:"binary"`
	Truncated *bool `json:"truncated"`
}

// Missing repositories or objects cannot establish available source.
type sourceFilesResponse struct {
	Data   map[string]*sourceRepository `json:"data"`
	Errors Errors                       `json:"errors"`
}

func (r *sourceFilesResponse) files(repo *Repository, files []*TreeEntry) (map[string]string, error) {
	if r.Errors.Validate() != nil || r.Data == nil {
		return nil, errors.New("github: source query returned missing or failed data")
	}

	contents := make(map[string]string, len(files))
	for index, file := range files {
		current := r.Data["file"+strconv.Itoa(index)]
		if current == nil {
			return nil, errors.New("github: source returned no repository")
		}

		if current.GetNodeID() != repo.GetNodeID() || strings.ToLower(current.GetFullName()) != strings.ToLower(repo.GetFullName()) {
			return nil, errors.New("github: source returned a different repository")
		}

		blob := current.Object
		if binarySourceBlob(blob, file.GetSHA()) {
			continue
		}

		content, err := decodeSourceBlob(blob, file.GetSHA())
		if err != nil {
			return nil, err
		}

		contents[file.GetPath()] = content
	}

	return contents, nil
}

// validateSourceRequest preserves source identity and regular-file checks even
// when a decorator can satisfy the entire request from cached data.
func validateSourceRequest(repo *Repository, revision string, files []*TreeEntry) error {
	if strings.IsEmpty(revision) || strings.IsEmpty(repo.GetNodeID()) {
		return errors.New("github: source revision or repository identity missing")
	}

	for _, file := range files {
		if !IsRegularSourceFile(file) || !io.ValidPath(file.GetPath()) || strings.Contains(file.GetPath(), "\\") || strings.IsEmpty(file.GetSHA()) {
			return errors.New("github: invalid source tree entry")
		}
	}
	return nil
}

func decodeSourceBlob(blob *sourceBlob, expectedSHA string) (string, error) {
	if !regularSourceBlob(blob) {
		return strings.Empty, errors.New("github: source content is incomplete or not text")
	}

	content := *blob.Content
	if !utf8.ValidString(content) || strings.Contains(content, "\x00") || *blob.Size != len(content) {
		return strings.Empty, errors.New("github: source content is incomplete or not text")
	}
	object := fmt.Sprintf("blob %d%c%s", len(content), 0, content)
	if blob.GetSHA() != expectedSHA || fmt.Sprintf("%x", sha1.Sum(strings.Bytes(object))) != expectedSHA { // #nosec G401 -- Match Git's native blob identifier.
		return strings.Empty, errors.New("github: source blob identity mismatch")
	}

	return content, nil
}

func regularSourceBlob(blob *sourceBlob) bool {
	return blob != nil && blob.GetType() == blobType && blob.Content != nil && blob.Size != nil &&
		blob.Binary != nil && !*blob.Binary && blob.Truncated != nil && !*blob.Truncated
}

func binarySourceBlob(blob *sourceBlob, expectedSHA string) bool {
	return blob != nil && blob.GetType() == blobType && blob.GetSHA() == expectedSHA && blob.Binary != nil && *blob.Binary
}

func sourceFiles(tree *github.Tree) ([]*TreeEntry, error) {
	if !completeSourceTree(tree) {
		return nil, errors.New("github: incomplete source tree")
	}

	seen := make(map[string]bool)

	for _, entry := range tree.Entries {
		name := entry.GetPath()
		if !io.ValidPath(name) || strings.Contains(name, "\\") || seen[name] {
			return nil, errors.New("github: invalid source tree path")
		}

		seen[name] = true
		if IsRegularSourceFile(entry) && strings.IsEmpty(entry.GetSHA()) {
			return nil, errors.New("github: source tree omitted a blob identity")
		}
	}

	return tree.Entries, nil
}

func completeSourceTree(tree *github.Tree) bool {
	return tree != nil && tree.Truncated != nil && !tree.GetTruncated() && !strings.IsEmpty(tree.GetSHA()) && tree.Entries != nil
}
