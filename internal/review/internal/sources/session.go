package sources

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/errors"
	"github.com/alexfalkowski/go-service/v3/slices"
)

var errRepositorySource = errors.New("review: could not retrieve complete repository source")

// NewSession creates per-review storage pinned to both reviewed revisions.
func NewSession(service github.Service, repository *github.Repository, headSHA, baseSHA string) *Session {
	return &Session{
		github: service, repository: repository, headSHA: headSHA, baseSHA: baseSHA,
		files: make(map[string]string), trees: make(map[string][]*github.TreeEntry),
		fingerprints: make(map[string]string), unavailableTrees: make(map[string]string),
	}
}

// Session reads pinned source and records the evidence supplied during one review.
// A session belongs to a single review conversation.
type Session struct {
	github           github.Service
	repository       *github.Repository
	files            map[string]string
	fingerprints     map[string]string
	trees            map[string][]*github.TreeEntry
	unavailableTrees map[string]string
	headSHA          string
	baseSHA          string
	missing          []string
	external         []string
	inspected        []string
	binary           []string
}

func (s *Session) recordFingerprint(revision string, entry *github.TreeEntry) {
	var side string
	if revision == s.headSHA {
		side = "head:"
	} else {
		side = "base:"
	}
	s.fingerprints[side+entry.GetPath()] = sourceFingerprint(entry)
}

func (s *Session) unavailable(label string) string {
	message := "Source unavailable: " + label
	if !slices.Contains(s.missing, message) {
		s.missing = append(s.missing, message)
	}

	return message
}

// ExternalUnavailable records a best-effort dependency/reference omission.
func (s *Session) ExternalUnavailable(label string) string {
	message := "External source unavailable: " + label
	if !slices.Contains(s.external, message) {
		s.external = append(s.external, message)
	}

	return message
}

// Files exposes the reviewed head's regular paths to source providers.
func (s *Session) Files(ctx context.Context) ([]string, error) {
	return s.treeFiles(ctx, s.headSHA)
}

// ReadFile keeps provider repository reads pinned to the reviewed head.
func (s *Session) ReadFile(ctx context.Context, filename string) (string, error) {
	return s.repositoryFile(ctx, s.headSHA, filename)
}

// Inspected records only the dependency text actually supplied to the model.
func (s *Session) Inspected(label string) {
	if !slices.Contains(s.inspected, label) {
		s.inspected = append(s.inspected, label)
	}
}

// Report describes the source actually supplied and any scope limitations.
type Report struct {
	// Fingerprints tracks inspected paths and verified unchanged paths from the prior review.
	Fingerprints map[string]string
	// Inspected lists text read from the repository or external sources.
	Inspected []string

	// Missing lists required repository source or patches that were unavailable.
	Missing []string

	// External lists unavailable best-effort dependency and reference text.
	External []string

	// Binary lists confirmed binary files excluded from text review.
	Binary []string
}

// Report returns the session's evidence after collection and tool use.
// The returned slices and map are read-only.
func (s *Session) Report() Report {
	return Report{
		Inspected: s.inspected, Missing: s.missing, External: s.external,
		Binary: s.binary, Fingerprints: s.fingerprints,
	}
}
