package strategy

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
)

// AutoMergeLabel is the exact opt-in shared by labels and comment commands.
const AutoMergeLabel = "automerge"

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
	if !Eligible(pull) || !s.github.IsBaseRepository(repository, pull) || pull.GetAutoMerge() != nil {
		return false
	}

	return github.HasLabel(pull.GetLabels(), AutoMergeLabel)
}

// Assess reads complete label authorization before selecting a merge action.
func (s *Label) Assess(ctx context.Context, repository *github.Repository, pull *github.PullRequest) (Decision, error) {
	events, err := s.github.ListPullRequestAutomationEvents(ctx, pull)
	if err != nil {
		return Decision{Policy: "automerge label", Reason: ReasonHistoryUnavailable}, err
	}

	if !labelAuthorizesAutoMerge(events) {
		return Decision{Policy: "automerge label", Reason: ReasonLabelAuthorizationRevoked}, nil
	}

	decision, err := decideMerge(repository, pull)
	decision.Policy = "automerge label"
	return decision, err
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

	decision, err := decideMerge(repository, current)
	if err != nil {
		return err
	}

	if decision.IsMerge() {
		return s.github.Merge(ctx, repository, current)
	}

	if decision.IsEnable() {
		return s.github.EnableAutoMerge(ctx, current)
	}

	return nil
}

// labelAuthorizesAutoMerge requires a label application newer than any removal
// or auto-merge disable event, preserving revocation across worker restarts.
func labelAuthorizesAutoMerge(events []*github.IssueEvent) bool {
	labeledAt, removedAt, disabledAt := github.LatestAutomationEventTimes(events, AutoMergeLabel)

	// Equal timestamps are ambiguous; wait for a later label application.
	return labeledAt.After(removedAt) && labeledAt.After(disabledAt)
}
