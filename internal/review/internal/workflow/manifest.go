package workflow

import (
	"path"

	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/slices"
	"github.com/alexfalkowski/go-service/v2/strings"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

const goModFile = "go.mod"

type dependencyRef struct {
	module   module.Version
	checksum string
}

func (s *sourceSession) resolveDependency(ctx context.Context, args dependencyRequest) (*dependencyRef, error) {
	manifest, err := s.moduleManifest(ctx, args.Manifest)
	if err != nil {
		return nil, err
	}

	selected := module.Version{Path: args.Module}

	for _, required := range manifest.Require {
		if required.Mod.Path == args.Module {
			selected = required.Mod
		}
	}

	if strings.IsEmpty(selected.Version) {
		return nil, errors.New("review: dependency is not required by the reviewed manifest")
	}

	selected, err = replacedModule(manifest, selected)
	if err != nil {
		return nil, err
	}

	if err := module.Check(selected.Path, selected.Version); err != nil {
		return nil, errors.New("review: invalid module identity")
	}

	sums, err := s.repositoryFile(ctx, s.headSHA, path.Join(path.Dir(args.Manifest), "go.sum"))
	if err != nil {
		return nil, err
	}

	checksum := moduleChecksum(sums, selected)
	if strings.IsEmpty(checksum) {
		return nil, errors.New("review: dependency has no reviewed go.sum checksum")
	}

	return &dependencyRef{module: selected, checksum: checksum}, nil
}

func (s *sourceSession) moduleManifest(ctx context.Context, filename string) (*modfile.File, error) {
	paths, err := s.treeFiles(ctx, s.headSHA)
	if err != nil {
		return nil, err
	}

	directory := path.Dir(filename)
	if slices.Contains(paths, path.Join(directory, "vendor/modules.txt")) {
		return nil, errors.New("review: workspace or vendored dependency; inspect repository source")
	}

	for {
		if slices.Contains(paths, path.Join(directory, "go.work")) {
			return nil, errors.New("review: workspace or vendored dependency; inspect repository source")
		}

		if directory == "." {
			break
		}

		directory = path.Dir(directory)
	}

	content, err := s.repositoryFile(ctx, s.headSHA, filename)
	if err != nil {
		return nil, err
	}

	manifest, err := modfile.Parse(filename, strings.Bytes(content), nil)
	if err != nil {
		return nil, errors.New("review: Go dependency manifest")
	}

	return manifest, nil
}

func replacedModule(manifest *modfile.File, selected module.Version) (module.Version, error) {
	replacement := selected

	for _, candidate := range manifest.Replace {
		if candidate.Old == selected {
			replacement = candidate.New
			break
		}

		if candidate.Old.Path == selected.Path && strings.IsEmpty(candidate.Old.Version) {
			replacement = candidate.New
		}
	}

	if strings.IsEmpty(replacement.Version) {
		return module.Version{}, errors.New("review: local dependency replacement; inspect repository source")
	}

	return replacement, nil
}

func moduleChecksum(sums string, selected module.Version) string {
	const checksumFieldCount = 3

	for line := range strings.SplitSeq(sums, "\n") {
		fields := strings.Fields(line)
		if len(fields) != checksumFieldCount {
			continue
		}

		sameModule := fields[0] == selected.Path
		sameVersion := fields[1] == selected.Version
		hasChecksum := strings.HasPrefix(fields[2], "h1:")
		if sameModule && sameVersion && hasChecksum {
			return fields[2]
		}
	}

	return strings.Empty
}
