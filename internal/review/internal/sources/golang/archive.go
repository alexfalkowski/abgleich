package golang

import (
	"archive/zip"
	"net/url"

	"github.com/alexfalkowski/go-service/v3/bytes"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/errors"
	"github.com/alexfalkowski/go-service/v3/io"
	"github.com/alexfalkowski/go-service/v3/strings"
	"golang.org/x/mod/module"
	"golang.org/x/mod/sumdb/dirhash"
)

func (s *Provider) archive(ctx context.Context, name, version, checksum string) (*zip.Reader, error) {
	escapedName, err := module.EscapePath(name)
	if err != nil {
		return nil, errors.Prefix("review: escape module path", err)
	}

	escapedVersion, err := module.EscapeVersion(version)
	if err != nil {
		return nil, errors.Prefix("review: escape module version", err)
	}

	endpoint := strings.Concat(s.proxy, "/", escapedName, "/@v/", url.PathEscape(escapedVersion), ".zip")
	content, err := s.readSource(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	return verifiedArchive(content, strings.Concat(name, "@", version, "/"), checksum)
}

func verifiedArchive(content []byte, prefix, checksum string) (*zip.Reader, error) {
	archive, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, errors.Prefix("review: open module archive", err)
	}

	var names []string
	files := make(map[string]*zip.File)
	for _, file := range archive.File {
		valid := file.Mode().IsRegular() && io.ValidPath(file.Name) && strings.HasPrefix(file.Name, prefix)
		if !valid || files[file.Name] != nil {
			return nil, errors.New("review: invalid module archive")
		}

		names = append(names, file.Name)
		files[file.Name] = file
	}

	hash, err := dirhash.Hash1(names, func(name string) (io.ReadCloser, error) { return files[name].Open() })
	if err != nil || hash != checksum {
		return nil, errors.New("review: module checksum mismatch")
	}

	return archive, nil
}
