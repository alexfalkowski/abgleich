package ruby

import "github.com/alexfalkowski/abgleich/internal/review/internal/sources/provider"

type gemMetadata struct {
	// Name identifies the registry gem and must match the reviewed lockfile.
	Name string `json:"name"`

	// Version is the exact locked version, excluding any platform suffix.
	Version string `json:"version"`

	// Platform identifies the locked artifact; ruby denotes a generic gem.
	Platform string `json:"platform"`

	// SHA is the registry's hexadecimal SHA-256 digest of the complete .gem archive.
	SHA string `json:"sha"`
}

type gemLookupResult struct {
	fileSizes map[string]int64
	fileData  []byte
}

type rubyDependencyRequest struct {
	// Manifest selects the reviewed repository's Gemfile.lock.
	Manifest string `json:"manifest"`

	// Gem selects a dependency declared by the reviewed lockfile.
	Gem string `json:"gem"`

	// Platform selects the exact locked artifact; ruby denotes a generic gem.
	Platform string `json:"platform"`

	// Path selects a file or directory within the gem; . lists the archive.
	Path string `json:"path"`

	// Range selects an inclusive excerpt; zero values request the whole file.
	provider.Range
}
