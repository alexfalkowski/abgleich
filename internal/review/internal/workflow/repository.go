package workflow

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/io"
	"github.com/alexfalkowski/go-service/v2/strings"
)

func (s *sourceSession) repositoryFile(ctx context.Context, revision, filename string) (string, error) {
	contents, err := s.repositoryFiles(ctx, revision, []string{filename})
	return contents[filename], err
}

func (s *sourceSession) repositoryFiles(ctx context.Context, revision string, filenames []string) (map[string]string, error) {
	contents, missing := s.uncachedFiles(revision, filenames)
	if len(missing) == 0 {
		return contents, nil
	}

	files, err := s.regularFiles(ctx, revision, missing, contents)
	if err != nil || len(files) == 0 {
		return contents, err
	}

	current, err := s.github.ReadSourceFiles(ctx, s.repository, revision, files)
	if err != nil {
		return nil, errors.Join(errRepositorySource, err)
	}

	for _, file := range files {
		filename := file.GetPath()
		key := revision + ":" + filename
		if content, ok := current[filename]; ok {
			s.inspected = append(s.inspected, key)
			s.files[key] = content
			contents[filename] = content
			continue
		}

		content := "Source excluded: " + key + " (binary file)"
		s.binary = append(s.binary, key)
		s.files[key] = content
		contents[filename] = content
	}

	return contents, nil
}

func (s *sourceSession) uncachedFiles(revision string, filenames []string) (map[string]string, []string) {
	contents := make(map[string]string, len(filenames))
	seen := make(map[string]bool, len(filenames))

	var missing []string
	for _, filename := range filenames {
		if seen[filename] {
			continue
		}

		seen[filename] = true

		if !io.ValidPath(filename) || strings.Contains(filename, "\\") {
			contents[filename] = s.unavailable("invalid repository path")
			continue
		}

		if content, ok := s.files[revision+":"+filename]; ok {
			contents[filename] = content
			continue
		}

		missing = append(missing, filename)
	}

	return contents, missing
}

func (s *sourceSession) regularFiles(ctx context.Context, revision string, filenames []string, contents map[string]string) ([]*github.TreeEntry, error) {
	entries, err := s.treeEntries(ctx, revision)
	if err != nil {
		return nil, err
	}

	byPath := make(map[string]*github.TreeEntry, len(entries))
	for _, entry := range entries {
		byPath[entry.GetPath()] = entry
	}

	var files []*github.TreeEntry
	for _, filename := range filenames {
		if entry := byPath[filename]; github.IsRegularSourceFile(entry) {
			files = append(files, entry)
		} else {
			contents[filename] = s.unavailable(strings.Concat(revision, ":", filename, " (regular file unavailable)"))
		}
	}

	return files, nil
}

func (s *sourceSession) treeFiles(ctx context.Context, revision string) ([]string, error) {
	entries, err := s.treeEntries(ctx, revision)
	if err != nil {
		return nil, err
	}

	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if github.IsRegularSourceFile(entry) {
			paths = append(paths, entry.GetPath())
		}
	}

	return paths, nil
}

func (s *sourceSession) treeEntries(ctx context.Context, revision string) ([]*github.TreeEntry, error) {
	if entries, ok := s.trees[revision]; ok {
		return entries, nil
	}

	entries, err := s.github.ListSourceFiles(ctx, s.repository, revision)
	if err != nil {
		return nil, errors.Join(errRepositorySource, err)
	}

	s.trees[revision] = entries
	return entries, nil
}
