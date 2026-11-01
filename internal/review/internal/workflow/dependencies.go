package workflow

import (
	"archive/zip"
	"io"
	"io/fs"
	"path"
	"slices"
	"unicode/utf8"

	"github.com/alexfalkowski/go-service/v2/bytes"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

type dependencyRequest struct {
	Manifest string `json:"manifest"`
	Module   string `json:"module"`
	Path     string `json:"path"`
}

type dependencyRef struct {
	module   module.Version
	checksum string
}

func (s *sourceSession) dependency(ctx context.Context, args dependencyRequest) (string, error) {
	if path.Base(args.Manifest) != "go.mod" || !fs.ValidPath(args.Manifest) || !fs.ValidPath(args.Path) || strings.Contains(args.Path, "\\") {
		return strings.Empty, errors.New("review: invalid dependency path")
	}

	ref, err := s.resolveDependency(ctx, args)
	if err != nil {
		return strings.Empty, err
	}

	key := ref.module.String()
	archive := s.archives[key]
	if archive == nil {
		archive, err = s.sources.archive(ctx, ref.module.Path, ref.module.Version, ref.checksum)
		if err != nil {
			return strings.Empty, errors.New("review: public dependency source unavailable or checksum mismatch")
		}

		s.archives[key] = archive
	}
	return s.dependencySource(archive, key, args.Path), nil
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

	sums := s.repositoryFile(ctx, path.Join(path.Dir(args.Manifest), "go.sum"), false)
	checksum := moduleChecksum(sums, selected)
	if strings.IsEmpty(checksum) {
		return nil, errors.New("review: dependency has no reviewed go.sum checksum")
	}
	return &dependencyRef{module: selected, checksum: checksum}, nil
}

func (s *sourceSession) moduleManifest(ctx context.Context, filename string) (*modfile.File, error) {
	paths, err := s.treeFiles(ctx, s.headSHA)
	if err != nil {
		return nil, errors.New("review: repository tree")
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

	manifest, err := modfile.Parse(filename, []byte(s.repositoryFile(ctx, filename, false)), nil)
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
	for line := range strings.SplitSeq(sums, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 3 && fields[0] == selected.Path && fields[1] == selected.Version && strings.HasPrefix(fields[2], "h1:") {
			return fields[2]
		}
	}
	return strings.Empty
}

func (s *sourceSession) dependencySource(archive *zip.Reader, key, requested string) string {
	prefix := key + "/"
	var source strings.Builder
	var names []string
	for _, file := range archive.File {
		filename := strings.TrimPrefix(file.Name, prefix)
		if filename == requested {
			return s.dependencyFile(file)
		}

		if packageSourceFile(filename, requested) {
			source.WriteString(s.dependencyFile(file))
			source.WriteString("\n")
		}

		if requested == "." || strings.HasPrefix(filename, requested+"/") {
			names = append(names, filename)
		}
	}

	if source.Len() > 0 {
		return source.String()
	}

	if len(names) == 0 {
		return s.unavailable(key + "/" + requested)
	}

	slices.Sort(names)

	return strings.Join("\n", names...)
}

func packageSourceFile(filename, directory string) bool {
	return path.Dir(filename) == directory && strings.HasSuffix(filename, ".go") && !strings.HasSuffix(filename, "_test.go")
}

func (s *sourceSession) dependencyFile(file *zip.File) string {
	label := file.Name
	reader, err := file.Open()
	if err != nil {
		return s.unavailable(label)
	}
	defer reader.Close()
	content, err := io.ReadAll(reader)
	if err != nil || !utf8.Valid(content) || bytes.ContainsRune(content, 0) {
		return s.unavailable(label)
	}

	if !slices.Contains(s.inspected, label) {
		s.inspected = append(s.inspected, label)
	}
	return "Source: " + label + " (reviewed go.sum verified)\n" + string(content)
}
