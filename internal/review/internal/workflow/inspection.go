package workflow

import (
	"fmt"
	"path"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/io"
	"github.com/alexfalkowski/go-service/v3/slices"
	"github.com/alexfalkowski/go-service/v3/strings"
)

func (s *sourceSession) list(ctx context.Context, filter string) ([]string, error) {
	files, err := s.treeEntries(ctx, s.headSHA)
	if err != nil {
		return nil, err
	}

	var paths []string
	for _, file := range files {
		readableSource := github.IsSubmodule(file) || github.IsRegularSourceFile(file)
		matchesFilter := strings.Contains(file.GetPath(), filter)

		if readableSource && matchesFilter {
			paths = append(paths, sourceFileDescription(file))
		}
	}

	return paths, nil
}

func (s *sourceSession) read(ctx context.Context, filenames []string) (string, error) {
	paths, err := s.treeFiles(ctx, s.headSHA)
	if err != nil {
		return strings.Empty, err
	}

	var selected []string
	for _, filename := range paths {
		if slices.Contains(filenames, filename) || slices.Contains(filenames, path.Dir(filename)) {
			selected = append(selected, filename)
		}
	}

	contents, err := s.repositoryFiles(ctx, s.headSHA, selected)
	if err != nil {
		return strings.Empty, err
	}

	var source strings.Builder
	for _, filename := range filenames {
		if err := ctx.Err(); err != nil {
			return strings.Empty, err
		}

		if request, found := s.submodulePath(filename); found {
			content, err := s.readSubmodule(ctx, request)
			if err != nil {
				return strings.Empty, err
			}

			source.WriteString(strings.Concat(content, "\n"))
			continue
		}

		source.WriteString(s.readPath(ctx, filename, paths, contents))
		source.WriteString("\n")
	}

	return source.String(), nil
}

func (s *sourceSession) readPath(ctx context.Context, filename string, paths []string, contents map[string]string) string {
	if !io.ValidPath(filename) || strings.Contains(filename, "\\") {
		return s.unavailable("review: invalid repository path")
	}

	if content, ok := contents[filename]; ok {
		return strings.Concat("Source: ", filename, " at ", s.headSHA, "\n", content)
	}

	var source strings.Builder
	for _, child := range paths {
		if path.Dir(child) != filename {
			continue
		}

		if err := ctx.Err(); err != nil {
			break
		}

		fmt.Fprintf(&source, "\nSource: %s at %s\n", child, s.headSHA)
		source.WriteString(contents[child])
		source.WriteString("\n")
	}

	if source.Len() == 0 {
		return s.missingPath(filename)
	}

	return source.String()
}

// missingPath distinguishes absent paths from excluded source using the complete tree.
func (s *sourceSession) missingPath(filename string) string {
	if sourcePathUnavailable(s.trees[s.headSHA], filename) {
		return s.unavailable(filename)
	}

	return strings.Concat("Source not found: ", filename)
}

func sourcePathUnavailable(entries []*github.TreeEntry, filename string) bool {
	if filename == "." {
		return true
	}

	for _, entry := range entries {
		name := entry.GetPath()
		if name == filename || strings.HasPrefix(name, filename+"/") ||
			(entry.GetType() != github.TreeDirectory && !github.IsRegularSourceFile(entry) && strings.HasPrefix(filename, name+"/")) {
			return true
		}
	}

	return false
}

func sourceFileDescription(file *github.TreeEntry) string {
	switch {
	case github.IsSubmodule(file):
		return strings.Concat(file.GetPath(), "\tsubmodule")
	case file.GetType() == github.TreeDirectory:
		return strings.Concat(file.GetPath(), "\tdirectory")
	case !github.IsRegularSourceFile(file):
		return strings.Concat(file.GetPath(), "\texcluded source")
	case file.Size != nil:
		return fmt.Sprintf("%s\t%d bytes", file.GetPath(), file.GetSize())
	default:
		return strings.Concat(file.GetPath(), "\tsize unavailable")
	}
}
