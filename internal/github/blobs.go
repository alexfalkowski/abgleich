package github

import (
	"crypto/sha1" // #nosec G505 -- Git blob identifiers require SHA-1; HTTPS authenticates the source.
	_ "embed"
	"fmt"
	"maps"
	"strconv"
	"unicode/utf8"

	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/errors"
	"github.com/alexfalkowski/go-service/v3/io"
	"github.com/alexfalkowski/go-service/v3/net/http"
	"github.com/alexfalkowski/go-service/v3/net/http/media"
	"github.com/alexfalkowski/go-service/v3/slices"
	"github.com/alexfalkowski/go-service/v3/strings"
)

const (
	blobType = "Blob"

	// Large combined queries timed out on GitHub; this bounds each request, not the review scope.
	sourceBatchSize = 50
)

// sourceFilesQuery formats generated variable declarations and file selections.
//
//go:embed graphql/source_files.gql
var sourceFilesQuery string

// sourceFileQuery formats generated aliases; revision/path expressions stay in variables.
//
//go:embed graphql/source_file.gql
var sourceFileQuery string

// ReadSourceFiles retrieves regular-file text at revision in sequential GraphQL
// batches without automatic retries. Binary blobs are omitted; missing or inconsistent
// repository identities, text, byte sizes, or blob hashes reject the whole group.
func (s *service) ReadSourceFiles(ctx context.Context, repo *Repository, revision string, files []*TreeEntry) (map[string]string, error) {
	if err := validateSourceRequest(repo, revision, files); err != nil {
		return nil, err
	}

	contents := make(map[string]string, len(files))
	for batch := range slices.Chunk(files, sourceBatchSize) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		current, err := s.readSourceBatch(ctx, repo, revision, batch)
		if err != nil {
			return nil, err
		}

		maps.Copy(contents, current)
	}

	return contents, nil
}

func (s *service) readSourceBatch(ctx context.Context, repo *Repository, revision string, files []*TreeEntry) (map[string]string, error) {
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
		fields = append(fields, fmt.Sprintf(sourceFileQuery, alias, alias))
	}

	return &sourceFilesRequest{
		Variables: variables,
		Query:     fmt.Sprintf(sourceFilesQuery, strings.Join(", ", parameters...), strings.Join("\n", fields...)),
	}, nil
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
