package workflow

import (
	"archive/zip"
	"fmt"
	"path"
	"unicode/utf8"

	"github.com/alexfalkowski/go-service/v2/bytes"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/io"
	"github.com/alexfalkowski/go-service/v2/slices"
	"github.com/alexfalkowski/go-service/v2/strings"
)

func (s *sourceSession) dependency(ctx context.Context, args dependencyRequest) (string, error) {
	validManifest := path.Base(args.Manifest) == goModFile && io.ValidPath(args.Manifest)
	validPath := io.ValidPath(args.Path) && !strings.Contains(args.Path, "\\")
	if !validManifest || !validPath {
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

	return s.dependencySource(archive, key, args.Path, args.sourceRange), nil
}

func (s *sourceSession) dependencySource(archive *zip.Reader, key, requested string, lines sourceRange) string {
	prefix := key + "/"
	var names []string

	for _, file := range archive.File {
		filename := strings.TrimPrefix(file.Name, prefix)
		if filename == requested {
			return s.dependencyFile(file, lines)
		}

		if requested == "." || strings.HasPrefix(filename, requested+"/") {
			names = append(names, fmt.Sprintf("%s\t%d bytes", filename, file.UncompressedSize64))
		}
	}

	if len(names) == 0 {
		// The verified archive proves absence; a guessed path is not missing source.
		return strings.Concat("Source not found: ", key, "/", requested)
	}

	slices.Sort(names)

	return strings.Join("\n", names...)
}

func (s *sourceSession) dependencyFile(file *zip.File, lines sourceRange) string {
	label := file.Name

	reader, err := file.Open()
	if err != nil {
		return s.externalUnavailable(label)
	}
	defer reader.Close()

	content, _, err := io.ReadAll(reader)
	if err != nil || !utf8.Valid(content) || bytes.ContainsRune(content, 0) {
		return s.externalUnavailable(label)
	}

	return strings.Concat(
		"Source: ", label, " (reviewed go.sum verified)\n",
		s.sourceText(label, bytes.String(content), lines),
	)
}
