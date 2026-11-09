package workflow

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/io"
	"github.com/alexfalkowski/go-service/v2/slices"
	"github.com/alexfalkowski/go-service/v2/strings"
)

func (s *sourceSession) submodulePath(filename string) (submoduleRequest, bool) {
	if !io.ValidPath(filename) || strings.Contains(filename, "\\") {
		return submoduleRequest{}, false
	}

	for _, entry := range s.trees[s.headSHA] {
		if !github.IsSubmodule(entry) {
			continue
		}

		mount := entry.GetPath()
		if filename == mount {
			return submoduleRequest{Submodule: mount, Path: "."}, true
		}

		prefix := strings.Concat(mount, "/")
		if strings.HasPrefix(filename, prefix) {
			return submoduleRequest{Submodule: mount, Path: strings.TrimPrefix(filename, prefix)}, true
		}
	}

	return submoduleRequest{}, false
}

func (s *sourceSession) readSubmodule(ctx context.Context, args submoduleRequest) (string, error) {
	if err := args.validate(); err != nil {
		return strings.Empty, err
	}

	entries, err := s.treeEntries(ctx, s.headSHA)
	if err != nil {
		return strings.Empty, err
	}

	for _, entry := range entries {
		if entry.GetPath() != args.Submodule || !github.IsSubmodule(entry) {
			continue
		}

		repo, err := s.github.GetSubmoduleRepository(ctx, s.repository, s.headSHA, entry)
		if err != nil {
			err = errors.Prefix(strings.Concat(args.Submodule, "/", args.Path), err)
			return s.unavailable(err.Error()), nil
		}

		files, err := s.github.ListSourceFiles(ctx, repo, entry.GetSHA())
		if err != nil {
			return strings.Empty, errors.Join(errRepositorySource, err)
		}

		return s.submoduleSource(ctx, repo, entry.GetSHA(), args, files)
	}

	return strings.Concat("Source not found: ", args.Submodule), nil
}

func (s *sourceSession) submoduleSource(ctx context.Context, repo *github.Repository, revision string, args submoduleRequest, files []*github.TreeEntry) (string, error) {
	label := strings.Concat(args.Submodule, " (", repo.GetFullName(), "): ", revision)
	var paths []string

	for _, file := range files {
		if file.GetPath() == args.Path && github.IsRegularSourceFile(file) {
			return s.submoduleFile(ctx, repo, revision, file, args)
		}

		if args.Path == "." || strings.HasPrefix(file.GetPath(), strings.Concat(args.Path, "/")) {
			paths = append(paths, sourceFileDescription(file))
		}
	}

	if len(paths) == 0 {
		if sourcePathUnavailable(files, args.Path) {
			return s.unavailable(strings.Concat(label, ":", args.Path)), nil
		}

		return strings.Concat("Source not found: ", args.Submodule, "/", args.Path), nil
	}

	slices.Sort(paths)

	return strings.Concat("Submodule: ", label, "\n", strings.Join("\n", paths...)), nil
}

func (s *sourceSession) submoduleFile(ctx context.Context, repo *github.Repository, revision string, file *github.TreeEntry, args submoduleRequest) (string, error) {
	contents, err := s.github.ReadSourceFiles(ctx, repo, revision, []*github.TreeEntry{file})
	if err != nil {
		return strings.Empty, errors.Join(errRepositorySource, err)
	}

	label := strings.Concat(args.Submodule, " (", repo.GetFullName(), "): ", revision, ":", args.Path)
	if content, ok := contents[file.GetPath()]; ok {
		// A pinned text read resolves only this mount's initial regular-file omission.
		omission := strings.Concat("Source unavailable: ", s.headSHA, ":", args.Submodule, " (regular file unavailable)")
		s.missing = slices.DeleteFunc(s.missing, func(message string) bool { return message == omission })
		s.inspected = append(s.inspected, label)
		return strings.Concat("Source: ", label, "\n", content), nil
	}

	s.binary = append(s.binary, label)

	return strings.Concat("Source excluded: ", label, " (binary file)"), nil
}
