package github

import (
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/errors"
)

// HasLabel reports exact, case-sensitive membership. Nil or empty label lists
// contain no matching label.
func HasLabel(labels []*Label, name string) bool {
	for _, label := range labels {
		if label.GetName() == name {
			return true
		}
	}

	return false
}

// AddLabel accepts success only when the response contains the requested label.
func (s *service) AddLabel(ctx context.Context, repo *Repository, number int, label string) error {
	labels, _, err := s.client.installation.Issues.AddLabelsToIssue(ctx, repo.GetOwner().GetLogin(), repo.GetName(), number, []string{label})
	if err != nil {
		return err
	}

	if HasLabel(labels, label) {
		return nil
	}

	return errors.New("github: label response does not contain the requested label")
}
