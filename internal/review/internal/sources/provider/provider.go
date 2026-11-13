package provider

import (
	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/errors"
	"github.com/alexfalkowski/go-signal/v2"
)

// ErrInvalidLookup identifies malformed tool input, which must stop inference
// rather than become a best-effort external source omission.
var ErrInvalidLookup = errors.New("invalid source lookup")

// Result contributes an optional strategy to the review source DI value group.
// An omitted review configuration contributes a nil strategy without opening transports.
type Result struct {
	// Out marks the result as an injected value group contribution.
	signal.Out

	// Strategy supplies source support; nil disables this contribution.
	Strategy Strategy `group:"review_sources"`
}

// Repository exposes revision-pinned source and per-review inspection records.
// Workflow checks current consent before every provider tool invocation.
type Repository interface {
	// Files lists regular paths at the reviewed head for manifest context.
	Files(ctx context.Context) ([]string, error)

	// ReadFile reads a complete regular file at the reviewed head.
	ReadFile(ctx context.Context, filename string) (string, error)

	// Inspected records a supplied file or excerpt, independently of model prose.
	Inspected(label string)

	// ExternalUnavailable records a best-effort external source omission.
	ExternalUnavailable(label string) string
}

// Strategy supplies language or framework guidance and optional source tools
// when the reviewed repository's paths match its selection rules.
type Strategy interface {
	// Matches selects support from reviewed head paths and changed filenames,
	// including deletions, without fetching external source.
	Matches(paths []string) bool

	// Guide returns worker-owned review instructions; repository and remote
	// content must never be loaded as instructions.
	Guide() string

	// Tools binds source lookups and fresh caches to one review. Implementations
	// must not retain the repository or its consent state across reviews.
	Tools(repository Repository) []ai.Tool
}
