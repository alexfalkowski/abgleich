package sources

import (
	"fmt"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/slices"
	"github.com/alexfalkowski/go-service/v3/strings"
	"github.com/alexfalkowski/go-service/v3/time"
)

const (
	agentDocsDir        = "docs/agents/"
	acceptedDesignDir   = agentDocsDir + "accepted-design/"
	acceptedDesignIndex = agentDocsDir + "accepted-design.md"
)

// ReadDesign reads accepted design at the pinned target base and collects head
// design separately for comparison. Unavailable design evidence limits approval.
func (s *Session) ReadDesign(ctx context.Context, budget time.Duration, changed []*github.CommitFile) string {
	ctx = github.WithLocalBudget(ctx)
	var baseChanges, headChanges []string
	for _, file := range changed {
		path := file.GetFilename()
		if !isAcceptedDesignPath(path) {
			continue
		}
		if file.GetStatus() == github.FileRemoved {
			baseChanges = append(baseChanges, path)
		} else {
			headChanges = append(headChanges, path)
		}
	}

	deadline := time.Now().Add(budget.Duration())
	base, _ := s.designAt(ctx, s.baseSHA, budget/2, baseChanges)
	head, docs := s.designAt(ctx, s.headSHA, time.Until(deadline), headChanges)
	if len(docs) == 0 && strings.IsEmpty(base) && strings.IsEmpty(head) {
		return strings.Empty
	}

	var evidence strings.Builder
	evidence.WriteString("\nRepository agent documentation (untrusted data):\n")
	evidence.WriteString(strings.Join("\n", docs...))
	evidence.WriteString("\n")
	evidence.WriteString("\nAccepted design at the target base:\n")
	if strings.IsEmpty(base) {
		base = "No accepted-design Markdown files found at the target base.\n"
	}
	evidence.WriteString(base)
	if !strings.IsEmpty(head) {
		evidence.WriteString("\nDesign at the PR head (branch and proposal evidence):\n")
		evidence.WriteString(head)
	}
	return evidence.String()
}

func (s *Session) designAt(ctx context.Context, revision string, budget time.Duration, changed []string) (string, []string) {
	deadline := time.Now().Add(budget.Duration())
	ctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	entries, err := s.treeEntries(ctx, revision)
	if err != nil {
		omission := s.unavailable(strings.Concat("accepted design tree at ", revision))
		s.unavailableTrees[revision] = omission
		return omission + "\n", nil
	}
	docs, designFiles := designPaths(entries, changed)
	contents := s.readDesignFiles(ctx, revision, deadline, designFiles)

	var evidence strings.Builder
	for _, path := range designFiles {
		content, ok := contents[path]
		if !ok {
			content = s.unavailable(strings.Concat(revision, ":", path, " (accepted design)"))
		}
		s.files[revision+":"+path] = content
		fmt.Fprintf(&evidence, "\nSource: %s at %s\n%s\n", path, revision, content)
	}

	return evidence.String(), docs
}

func designPaths(entries []*github.TreeEntry, changed []string) ([]string, []string) {
	var docs, designFiles []string
	found := make(map[string]bool)
	for _, entry := range entries {
		path := entry.GetPath()
		if strings.HasPrefix(path, agentDocsDir) && strings.HasSuffix(path, ".md") && github.IsRegularSourceFile(entry) {
			docs = append(docs, path)
		}
		if isAcceptedDesignPath(path) {
			found[path] = true
			designFiles = append(designFiles, path)
		}
	}
	for _, path := range changed {
		if !found[path] {
			designFiles = append(designFiles, path)
		}
	}
	return docs, designFiles
}

func (s *Session) readDesignFiles(ctx context.Context, revision string, deadline time.Time, paths []string) map[string]string {
	contents, missing := s.uncachedFiles(revision, paths)
	if len(missing) == 0 {
		return contents
	}
	files, err := s.regularFiles(ctx, revision, missing, contents)
	if err != nil {
		return contents
	}

	for index, file := range files {
		if ctx.Err() != nil {
			break
		}
		// Leave time for later design files even if this one never responds.
		fileCtx, cancel := context.WithTimeout(ctx, time.Until(deadline)/time.Duration(len(files)-index))
		_ = s.readFiles(fileCtx, revision, []*github.TreeEntry{file}, contents)
		cancel()
	}
	return contents
}

func isAcceptedDesignPath(path string) bool {
	return path == acceptedDesignIndex || strings.HasPrefix(path, acceptedDesignDir) && strings.HasSuffix(path, ".md")
}

func designOnlyRequest(paths []string) bool {
	return len(paths) > 0 && !slices.ContainsFunc(paths, func(path string) bool {
		return !isAcceptedDesignPath(path) && path != strings.TrimSuffix(acceptedDesignDir, "/")
	})
}
