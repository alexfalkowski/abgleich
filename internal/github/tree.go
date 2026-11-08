package github

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/io"
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
)

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
