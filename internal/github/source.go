package github

import (
	"crypto/sha1" // #nosec G505 -- Git blob identifiers require SHA-1; HTTPS authenticates the source.
	"fmt"
	"io/fs"
	"unicode/utf8"

	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/google/go-github/v93/github"
)

func (s *service) ReadSourceFile(ctx context.Context, repo *Repository, revision, filename string) (string, error) {
	if !fs.ValidPath(filename) || strings.Contains(filename, "\\") || strings.IsEmpty(revision) {
		return strings.Empty, errors.New("github: invalid source path or revision")
	}

	file, _, _, err := s.client.installation.Repositories.GetContents(ctx, repo.GetOwner().GetLogin(), repo.GetName(), filename,
		&github.RepositoryContentGetOptions{Ref: revision})
	if err != nil {
		return strings.Empty, errors.New("github: source unavailable")
	}

	if !regularSourceFile(file, filename) {
		return strings.Empty, errors.New("github: source returned no matching regular file")
	}

	return decodeSourceFile(file)
}

func (s *service) ListSourceFiles(ctx context.Context, repo *Repository, revision string) ([]string, error) {
	if strings.IsEmpty(revision) {
		return nil, errors.New("github: source revision missing")
	}

	tree, _, err := s.client.installation.Git.GetTree(ctx, repo.GetOwner().GetLogin(), repo.GetName(), revision, true)
	if err != nil {
		return nil, errors.New("github: source tree unavailable")
	}

	return sourcePaths(tree)
}

func regularSourceFile(file *github.RepositoryContent, filename string) bool {
	return file != nil && file.GetType() == "file" && file.GetPath() == filename &&
		file.GetEncoding() == "base64" && strings.IsEmpty(file.GetTarget()) && strings.IsEmpty(file.GetSubmoduleGitURL())
}

func decodeSourceFile(file *github.RepositoryContent) (string, error) {
	content, err := file.GetContent()
	if err != nil || !utf8.ValidString(content) || strings.Contains(content, "\x00") || file.Size == nil || file.GetSize() != len(content) {
		return strings.Empty, errors.New("github: source content is incomplete or not text")
	}

	blob := fmt.Sprintf("blob %d%c%s", len(content), 0, content)
	if fmt.Sprintf("%x", sha1.Sum([]byte(blob))) != file.GetSHA() { // #nosec G401 -- Match Git's native blob identifier.
		return strings.Empty, errors.New("github: source blob identity mismatch")
	}

	return content, nil
}

func sourcePaths(tree *github.Tree) ([]string, error) {
	if !completeSourceTree(tree) {
		return nil, errors.New("github: incomplete source tree")
	}

	var files []string
	seen := make(map[string]bool)

	for _, entry := range tree.Entries {
		name := entry.GetPath()
		if !fs.ValidPath(name) || strings.Contains(name, "\\") || seen[name] {
			return nil, errors.New("github: invalid source tree path")
		}

		seen[name] = true
		if regularTreeEntry(entry) {
			files = append(files, name)
		}
	}

	return files, nil
}

func completeSourceTree(tree *github.Tree) bool {
	return tree != nil && tree.Truncated != nil && !tree.GetTruncated() && !strings.IsEmpty(tree.GetSHA()) && tree.Entries != nil
}

func regularTreeEntry(entry *github.TreeEntry) bool {
	return entry.GetType() == "blob" && (entry.GetMode() == "100644" || entry.GetMode() == "100755")
}
