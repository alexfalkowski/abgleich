package strategy

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/time"
)

const autoMergeLabel = "automerge"

var _ Strategy = (*Label)(nil)

// NewLabel constructs the policy for pull requests authorized by the automerge label.
func NewLabel(service github.Service) *Label {
	return &Label{github: service}
}

// Label automates labeled pull requests, including contributions from forks.
type Label struct {
	github github.Service
}

// Matches identifies eligible labeled pull requests without auto-merge enabled.
func (s *Label) Matches(repository *github.Repository, pull *github.PullRequest) bool {
	if !eligible(pull) || !s.github.IsBaseRepository(repository, pull) || pull.GetAutoMerge() != nil {
		return false
	}

	for _, label := range pull.Labels {
		if label.GetName() == autoMergeLabel {
			return true
		}
	}

	return false
}

// Reconcile verifies label authorization and refreshes the pull request before acting.
func (s *Label) Reconcile(ctx context.Context, repository *github.Repository, expected *github.PullRequest) error {
	events, err := s.github.ListPullRequestAutomationEvents(ctx, expected)
	if err != nil {
		return err
	}

	if !labelAuthorizesAutoMerge(events) {
		return nil
	}

	current, err := s.github.GetPullRequest(ctx, repository, expected.GetNumber())
	if err != nil {
		return err
	}

	if !s.Matches(repository, current) {
		return nil
	}

	if !s.github.IsSamePullRequest(current, expected) {
		return errors.New("pull request changed before labeled automation; retry on the next run")
	}

	if !repository.GetAllowSquashMerge() {
		return errors.New("repository does not allow squash merging")
	}

	if readyToMerge(current) {
		return s.github.Merge(ctx, repository, current)
	}

	return s.github.EnableAutoMerge(ctx, current)
}

// labelAuthorizesAutoMerge requires a label application newer than any removal
// or auto-merge disable event, preserving revocation across worker restarts.
func labelAuthorizesAutoMerge(events []*github.IssueEvent) bool {
	var labeledAt, revokedAt time.Time
	for _, event := range events {
		occurredAt := event.GetCreatedAt().Time
		switch event.GetEvent() {
		case "LabeledEvent":
			if event.GetLabel().GetName() == autoMergeLabel && occurredAt.After(labeledAt) {
				labeledAt = occurredAt
			}
		case "UnlabeledEvent":
			if event.GetLabel().GetName() == autoMergeLabel && occurredAt.After(revokedAt) {
				revokedAt = occurredAt
			}
		case "AutoMergeDisabledEvent":
			if occurredAt.After(revokedAt) {
				revokedAt = occurredAt
			}
		}
	}

	// Equal timestamps are ambiguous; wait for a later label application.
	return labeledAt.After(revokedAt)
}
