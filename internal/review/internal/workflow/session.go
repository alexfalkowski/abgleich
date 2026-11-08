package workflow

import (
	"archive/zip"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/slices"
)

var errRepositorySource = errors.New("review: could not retrieve complete repository source")

type sourceSession struct {
	github     github.Service
	sources    *Sources
	repository *github.Repository
	files      map[string]string
	trees      map[string][]*github.TreeEntry
	archives   map[string]*zip.Reader
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

func (s *sourceSession) externalUnavailable(label string) string {
	message := "External source unavailable: " + label
	if !slices.Contains(s.external, message) {
		s.external = append(s.external, message)
	}

	return message
}
