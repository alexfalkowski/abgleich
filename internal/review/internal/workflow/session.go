package workflow

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/slices"
)

var errRepositorySource = errors.New("review: could not retrieve complete repository source")

type sourceSession struct {
	github     github.Service
	repository *github.Repository
	files      map[string]string
	trees      map[string][]*github.TreeEntry
	headSHA    string
	missing    []string
	external   []string
	inspected  []string
	binary     []string
}

func (s *sourceSession) unavailable(label string) string {
	message := "Source unavailable: " + label
	if !slices.Contains(s.missing, message) {
		s.missing = append(s.missing, message)
	}

	return message
}

// ExternalUnavailable records a best-effort dependency/reference omission.
func (s *sourceSession) ExternalUnavailable(label string) string {
	message := "External source unavailable: " + label
	if !slices.Contains(s.external, message) {
		s.external = append(s.external, message)
	}

	return message
}

// Files exposes the reviewed head's regular paths to source providers.
func (s *sourceSession) Files(ctx context.Context) ([]string, error) {
	return s.treeFiles(ctx, s.headSHA)
}

// ReadFile keeps provider repository reads pinned to the reviewed head.
func (s *sourceSession) ReadFile(ctx context.Context, filename string) (string, error) {
	return s.repositoryFile(ctx, s.headSHA, filename)
}

// Inspected records only the dependency text actually supplied to the model.
func (s *sourceSession) Inspected(label string) {
	if !slices.Contains(s.inspected, label) {
		s.inspected = append(s.inspected, label)
	}
}
