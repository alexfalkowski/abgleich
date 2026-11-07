package github

import (
	"encoding/hex"
	"regexp"

	"github.com/alexfalkowski/go-service/v2/bytes"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/encoding/base64"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/io"
	"github.com/alexfalkowski/go-service/v2/slices"
	"github.com/alexfalkowski/go-service/v2/strings"
)

var publicRepositoryName = regexp.MustCompile(`^[A-Za-z0-9_-]+/[A-Za-z0-9_.-]+$`)

// PublicSource records an upstream file or directory at a resolved commit.
type PublicSource struct {
	// Text contains the complete file text, including an empty string for an
	// empty file. It is nil for directory listings and absent paths.
	Text *string

	// Revision is the immutable commit SHA resolved from the requested ref.
	Revision string

	// Paths lists sorted, repository-relative regular file paths beneath a
	// directory. An empty, non-nil slice identifies an empty directory;
	// nil with nil Text means the requested path is absent.
	Paths []string
}

// ReadPublicSource resolves ref to a commit in a confirmed public owner/name
// repository without App credentials. It returns verified complete text or sorted
// regular-file paths beneath a directory. A confirmed absent path has nil Text
// and Paths; excluded paths, binary content, incomplete trees, and inconsistent
// responses return an error.
func (s *service) ReadPublicSource(ctx context.Context, repository, ref, path string) (*PublicSource, error) {
	if err := validatePublicSource(repository, ref, path); err != nil {
		return nil, err
	}

	owner, name, _ := strings.Cut(repository, "/")
	if err := s.checkPublicRepository(ctx, owner, name); err != nil {
		return nil, err
	}

	revision, entries, err := s.publicSourceTree(ctx, owner, name, ref)
	if err != nil {
		return nil, err
	}

	entry, paths, err := publicSourcePath(entries, path)
	if err != nil {
		return nil, err
	}

	source := &PublicSource{Revision: revision, Paths: paths}
	if entry != nil {
		text, err := s.publicText(ctx, owner, name, entry)
		if err != nil {
			return nil, err
		}
		source.Text = new(text)
	}

	return source, nil
}

func (s *service) checkPublicRepository(ctx context.Context, owner, name string) error {
	repo, _, err := s.client.public.Repositories.Get(ctx, owner, name)
	if err != nil || repo == nil {
		return errors.New("github: public source repository unavailable")
	}

	sameRepository := strings.ToLower(repo.GetFullName()) == strings.ToLower(owner+"/"+name)
	privacyKnown := repo.Private != nil
	publicRepository := !repo.GetPrivate()
	visibility := repo.GetVisibility()
	publicVisibility := strings.IsEmpty(visibility) || visibility == "public"
	if !sameRepository || !privacyKnown || !publicRepository || !publicVisibility {
		return errors.New("github: source repository is not confirmed public")
	}

	return nil
}

func (s *service) publicSourceTree(ctx context.Context, owner, name, ref string) (string, []*TreeEntry, error) {
	commit, _, err := s.client.public.Repositories.GetCommit(ctx, owner, name, ref, nil)
	if err != nil {
		return strings.Empty, nil, errors.New("github: public source revision unavailable or inconsistent")
	}

	revision := commit.GetSHA()
	treeSHA := commit.GetCommit().GetTree().GetSHA()
	validRevision := validSourceSHA(revision)
	validTree := validSourceSHA(treeSHA)
	pinnedRef := validSourceSHA(ref)
	matchesRef := !pinnedRef || strings.ToLower(revision) == strings.ToLower(ref)
	if !validRevision || !validTree || !matchesRef {
		return strings.Empty, nil, errors.New("github: public source revision unavailable or inconsistent")
	}

	tree, _, err := s.client.public.Git.GetTree(ctx, owner, name, revision, true)
	if err != nil {
		return strings.Empty, nil, errors.New("github: public source tree unavailable or inconsistent")
	}

	sameTree := tree.GetSHA() == treeSHA
	if !sameTree {
		return strings.Empty, nil, errors.New("github: public source tree unavailable or inconsistent")
	}

	entries, err := sourceFiles(tree)
	return revision, entries, err
}

func (s *service) publicText(ctx context.Context, owner, name string, entry *TreeEntry) (string, error) {
	blobSHA := entry.GetSHA()
	if !validSourceSHA(blobSHA) {
		return strings.Empty, errors.New("github: invalid public source blob identity")
	}

	blob, _, err := s.client.public.Git.GetBlob(ctx, owner, name, blobSHA)
	if err != nil {
		return strings.Empty, errors.New("github: public source blob unavailable")
	}
	if blob.GetEncoding() != "base64" {
		return strings.Empty, errors.New("github: public source blob unavailable")
	}

	content, err := base64.Decode(blob.Content)
	if err != nil {
		return strings.Empty, errors.New("github: invalid public source blob encoding")
	}

	source := &sourceBlob{
		Type:      new(blobType),
		Content:   new(bytes.String(content)),
		Size:      blob.Size,
		SHA:       blob.SHA,
		Binary:    new(false),
		Truncated: new(false),
	}
	return decodeSourceBlob(source, blobSHA)
}

func publicSourcePath(entries []*TreeEntry, path string) (*TreeEntry, []string, error) {
	directoryExists := path == "."
	directoryPrefix := path + "/"
	if directoryExists {
		directoryPrefix = strings.Empty
	}

	var paths []string
	for _, entry := range entries {
		entryPath := entry.GetPath()
		regularFile := IsRegularSourceFile(entry)
		if excludedPublicSourcePath(entry, path) {
			return nil, nil, errors.New("github: public source path is not a regular file or directory")
		}

		if entryPath == path {
			if regularFile {
				return entry, nil, nil
			}

			directoryExists = true
		}

		childFile := regularFile && strings.HasPrefix(entryPath, directoryPrefix)
		if childFile {
			paths = append(paths, entryPath)
		}
	}

	if paths == nil && directoryExists {
		paths = []string{}
	}

	slices.Sort(paths)

	return nil, paths, nil
}

func excludedPublicSourcePath(entry *TreeEntry, path string) bool {
	return !IsRegularSourceFile(entry) && entry.GetType() != TreeDirectory &&
		(entry.GetPath() == path || strings.HasPrefix(path, entry.GetPath()+"/"))
}

func validSourceSHA(sha string) bool {
	decoded, err := hex.DecodeString(sha)
	return err == nil && len(decoded) == 20
}

func validPublicRef(ref string) bool {
	validPath := !strings.IsEmpty(ref) && io.ValidPath(ref)
	hasBackslash := strings.Contains(ref, "\\")
	hasQuery := strings.Contains(ref, "?")
	hasFragment := strings.Contains(ref, "#")
	hasEscape := strings.Contains(ref, "%")

	return validPath && !hasBackslash && !hasQuery && !hasFragment && !hasEscape
}

func validatePublicSource(repository, ref, path string) error {
	validRepository := publicRepositoryName.MatchString(repository)
	validRef := validPublicRef(ref)
	validPath := io.ValidPath(path) && !strings.Contains(path, "\\")
	if !validRepository || !validRef || !validPath {
		return errors.New("github: invalid public source repository, ref, or path")
	}

	_, name, _ := strings.Cut(repository, "/")
	reservedName := name == "." || name == ".."
	if reservedName {
		return errors.New("github: invalid public source repository")
	}

	return nil
}
