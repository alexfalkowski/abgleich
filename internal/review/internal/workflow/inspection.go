package workflow

import (
	"fmt"
	"path"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/io"
	"github.com/alexfalkowski/go-service/v2/slices"
	"github.com/alexfalkowski/go-service/v2/strings"
)

func (s *sourceSession) list(ctx context.Context, filter string) ([]string, error) {
	files, err := s.treeEntries(ctx, s.headSHA)
	if err != nil {
		return nil, err
	}

	var paths []string
	for _, file := range files {
		if github.IsRegularSourceFile(file) && strings.Contains(file.GetPath(), filter) {
			var size string
			if file.Size != nil {
				size = fmt.Sprintf("%d bytes", file.GetSize())
			} else {
				size = "size unavailable"
			}

			paths = append(paths, file.GetPath()+"\t"+size)
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
	if filename == "." {
		return s.unavailable(filename)
	}

	for _, entry := range s.trees[s.headSHA] {
		name := entry.GetPath()
		if name == filename || strings.HasPrefix(name, filename+"/") ||
			(entry.GetType() != github.TreeDirectory && !github.IsRegularSourceFile(entry) && strings.HasPrefix(filename, name+"/")) {
			return s.unavailable(filename)
		}
	}

	return strings.Concat("Source not found: ", filename)
}
