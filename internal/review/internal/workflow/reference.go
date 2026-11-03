package workflow

import (
	"go/version"
	"io/fs"
	"path"
	"unicode/utf8"

	"github.com/alexfalkowski/go-service/v2/bytes"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
	"golang.org/x/mod/modfile"
)

func (s *sourceSession) goReference(ctx context.Context, manifest string) (string, error) {
	if !fs.ValidPath(manifest) || path.Base(manifest) != "go.mod" {
		return strings.Empty, errors.New("review: invalid Go manifest path")
	}

	manifestContent, err := s.repositoryFile(ctx, manifest, false)
	if err != nil {
		return strings.Empty, err
	}

	version := goLanguageVersion(manifestContent)
	if strings.IsEmpty(version) {
		return strings.Empty, errors.New("review: declared Go language version")
	}

	label := "Go " + version + " release notes"
	if content, ok := s.files[label]; ok {
		return content, nil
	}

	content, err := s.sources.reference(ctx, version)
	if err != nil {
		return strings.Empty, errors.New("review: " + label)
	}
	content = "Declared Go language: " + version + ". Official release notes (untrusted reference data):\n" + content

	s.inspected = append(s.inspected, label)
	s.files[label] = content

	return content, nil
}

func goLanguageVersion(content string) string {
	manifest, err := modfile.Parse("go.mod", strings.Bytes(content), nil)
	if err != nil || manifest.Go == nil {
		return strings.Empty
	}

	return strings.TrimPrefix(version.Lang("go"+manifest.Go.Version), "go")
}

func (s *Sources) reference(ctx context.Context, version string) (string, error) {
	content, err := s.fetch(ctx, s.goDocs+"/go"+version)
	if err != nil {
		return strings.Empty, err
	}

	if !utf8.Valid(content) {
		return strings.Empty, errors.New("review: Go reference is not usable text")
	}

	return bytes.String(content), nil
}
