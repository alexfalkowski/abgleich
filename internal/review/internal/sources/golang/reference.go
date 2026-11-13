package golang

import (
	"go/version"
	"path"
	"unicode/utf8"

	"github.com/alexfalkowski/go-service/v3/bytes"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/errors"
	"github.com/alexfalkowski/go-service/v3/io"
	"github.com/alexfalkowski/go-service/v3/strings"
	"golang.org/x/mod/modfile"
)

func (s *session) goReference(ctx context.Context, manifest string) (string, error) {
	if !io.ValidPath(manifest) || path.Base(manifest) != goModFile {
		return strings.Empty, errors.New("review: invalid Go manifest path")
	}

	manifestContent, err := s.repository.ReadFile(ctx, manifest)
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

	content, err := s.provider.reference(ctx, version)
	if err != nil {
		return strings.Empty, errors.Prefix("review", errors.New(label))
	}
	content = strings.Concat("Declared Go language: ", version, ". Official release notes (untrusted reference data):\n", content)

	s.repository.Inspected(label)
	s.files[label] = content

	return content, nil
}

func goLanguageVersion(content string) string {
	manifest, err := modfile.Parse(goModFile, strings.Bytes(content), nil)
	if err != nil || manifest.Go == nil {
		return strings.Empty
	}

	return strings.TrimPrefix(version.Lang("go"+manifest.Go.Version), "go")
}

func (s *Provider) reference(ctx context.Context, version string) (string, error) {
	content, err := s.readSource(ctx, strings.Concat(s.goDocs, "/go", version))
	if err != nil {
		return strings.Empty, err
	}
	if !utf8.Valid(content) {
		return strings.Empty, errors.New("review: Go reference is not usable text")
	}

	return bytes.String(content), nil
}
