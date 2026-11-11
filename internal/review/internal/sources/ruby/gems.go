package ruby

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"path"
	"unicode/utf8"

	"github.com/alexfalkowski/abgleich/internal/review/internal/sources/provider"
	"github.com/alexfalkowski/go-service/v2/bytes"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/encoding/codec"
	"github.com/alexfalkowski/go-service/v2/encoding/json"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/io"
	"github.com/alexfalkowski/go-service/v2/slices"
	"github.com/alexfalkowski/go-service/v2/strings"
)

func (s *session) rubyDependency(ctx context.Context, args rubyDependencyRequest) (string, error) {
	if err := args.validate(); err != nil {
		return strings.Empty, err
	}

	lock, err := s.repository.ReadFile(ctx, args.Manifest)
	if err != nil {
		return strings.Empty, err
	}

	ref, err := readLockedGem(lock, args.Gem, args.Platform)
	if err != nil {
		return strings.Empty, err
	}

	key := strings.Concat(ref.name, "@", ref.version, "-", ref.platform)
	cacheKey := key + ":" + ref.checksum
	archive := s.gems[cacheKey]
	if archive == nil {
		archive, err = s.provider.gem(ctx, ref, &s.bytesRead)
		if err != nil {
			return strings.Empty, err
		}
	}

	result, err := readGem(ctx, archive, args, &s.bytesRead)
	if err != nil {
		return strings.Empty, err
	}
	if s.gems == nil {
		s.gems = make(map[string][]byte)
	}
	s.gems[cacheKey] = archive

	return s.gemSource(result, key, args, ref.checksum), nil
}

func (r rubyDependencyRequest) validate() error {
	validLock := path.Base(r.Manifest) == "Gemfile.lock"
	validManifest := io.ValidPath(r.Manifest) && !strings.Contains(r.Manifest, "\\")
	validGem := gemName.MatchString(r.Gem)
	validPlatform := gemName.MatchString(r.Platform)
	validPath := io.ValidPath(r.Path) && !strings.Contains(r.Path, "\\")

	if !validLock || !validManifest || !validGem || !validPlatform || !validPath {
		return errors.New("review: invalid Ruby dependency path")
	}

	return r.Validate()
}

func (s *session) gemSource(result *gemLookupResult, key string, args rubyDependencyRequest, checksum string) string {
	if _, found := result.fileSizes[args.Path]; found {
		label := key + "/" + args.Path
		if !utf8.Valid(result.fileData) || bytes.ContainsRune(result.fileData, 0) {
			return s.repository.ExternalUnavailable(label + " (binary file)")
		}

		var verification string
		if strings.IsEmpty(checksum) {
			verification = "RubyGems registry checksum verified"
		} else {
			verification = "reviewed lockfile checksum verified"
		}

		return strings.Concat("Source: ", label, " (", verification, ")\n", provider.Text(s.repository, label, bytes.String(result.fileData), args.Range))
	}

	var names []string
	for name, size := range result.fileSizes {
		if args.Path == "." || strings.HasPrefix(name, args.Path+"/") {
			names = append(names, fmt.Sprintf("%s\t%d bytes", name, size))
		}
	}

	if len(names) == 0 {
		return strings.Concat("Source not found: ", key, "/", args.Path)
	}

	slices.Sort(names)

	return strings.Join("\n", names...)
}

func (s *Provider) gem(ctx context.Context, ref *lockedGem, used *int64) ([]byte, error) {
	endpoint := strings.Concat(s.gemRegistry, "/api/v2/rubygems/", ref.name, "/versions/", ref.version, ".json?platform=", ref.platform)
	content, err := s.readSource(ctx, endpoint, used)
	if err != nil {
		return nil, err
	}

	var metadata gemMetadata
	err = json.NewEncoder().Decode(bytes.NewReader(content), &metadata, codec.WithDiscardUnknown())
	validName := metadata.Name == ref.name
	validVersion := metadata.Version == ref.version
	validPlatform := metadata.Platform == ref.platform
	validChecksum := gemChecksum.MatchString(metadata.SHA)
	if err != nil || !validName || !validVersion || !validPlatform || !validChecksum {
		return nil, errors.New("review: inconsistent RubyGems metadata")
	}

	artifact := strings.Concat(ref.name, "-", ref.version)
	if ref.platform != "ruby" {
		artifact += "-" + ref.platform
	}

	content, err = s.readSource(ctx, strings.Concat(s.gemRegistry, "/gems/", artifact, ".gem"), used)
	if err != nil {
		return nil, err
	}

	if err := verifyGemChecksum(content, metadata.SHA, ref.checksum); err != nil {
		return nil, err
	}

	return content, ctx.Err()
}

func verifyGemChecksum(content []byte, registry, locked string) error {
	digest := fmt.Sprintf("%x", sha256.Sum256(content))
	if digest != strings.ToLower(registry) || (!strings.IsEmpty(locked) && digest != locked) {
		return errors.New("review: Ruby gem checksum mismatch")
	}

	return nil
}

func readGem(ctx context.Context, content []byte, args rubyDependencyRequest, used *int64) (*gemLookupResult, error) {
	archive := tar.NewReader(bytes.NewReader(content))
	var source *gemLookupResult

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		file, err := archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, errors.New("review: invalid gem archive")
		}

		if file.Name != "data.tar.gz" {
			continue
		}

		if source != nil || file.Typeflag != tar.TypeReg {
			return nil, errors.New("review: invalid gem payload")
		}

		source, err = readGemPayload(ctx, archive, args, used)
		if err != nil {
			return nil, err
		}
	}

	if source == nil {
		return nil, errors.New("review: missing gem payload")
	}

	return source, ctx.Err()
}

func readGemPayload(ctx context.Context, archive io.Reader, args rubyDependencyRequest, used *int64) (*gemLookupResult, error) {
	compressed := &gemReader{ctx: ctx, reader: archive, used: used}
	payload, err := gzip.NewReader(compressed)
	if err != nil {
		return nil, errors.Prefix("review: invalid gem payload", err)
	}
	defer payload.Close()

	reader := &gemReader{ctx: ctx, reader: payload, used: used}
	return readGemFiles(ctx, reader, args, used)
}

func readGemFiles(ctx context.Context, payload io.Reader, args rubyDependencyRequest, used *int64) (*gemLookupResult, error) {
	archive := tar.NewReader(payload)
	result := &gemLookupResult{fileSizes: make(map[string]int64)}
	seen := make(map[string]bool)

	for {
		file, err := archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, errors.Prefix("review: invalid gem source archive", err)
		}

		name, err := gemEntryName(file, seen)
		if err != nil {
			return nil, err
		}

		seen[name] = true
		if file.FileInfo().IsDir() {
			continue
		}

		result.fileSizes[name] = file.Size
		if name != args.Path {
			continue
		}
		if file.Size > gemByteLimit-*used {
			return nil, errGemByteLimit
		}
		// Count logical file reads too: sparse tar entries can produce bytes
		// without consuming the same number of bytes from the gzip stream.
		reader := &gemReader{ctx: ctx, reader: archive, used: used}
		result.fileData, _, err = io.ReadAll(reader)
		if err != nil {
			return nil, err
		}
	}

	// Consume the gzip trailer as well as tar entries so CRC failures remain errors.
	if _, err := io.Copy(io.Discard, payload); err != nil {
		return nil, errors.Prefix("review: invalid gem compression checksum", err)
	}

	return result, nil
}

func gemEntryName(file *tar.Header, seen map[string]bool) (string, error) {
	name := strings.TrimSuffix(strings.TrimPrefix(file.Name, "./"), "/")
	validPath := io.ValidPath(name) && !strings.Contains(name, "\\")
	validType := file.Typeflag == tar.TypeReg || file.Typeflag == tar.TypeDir
	if !validPath || seen[name] || !validType {
		return strings.Empty, errors.New("review: invalid gem source path")
	}

	return name, nil
}
