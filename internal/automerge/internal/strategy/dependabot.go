package strategy

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
)

var _ Strategy = (*Dependabot)(nil)

// NewDependabot constructs the policy for same-repository Dependabot pull requests.
func NewDependabot(service github.Service) *Dependabot {
	return &Dependabot{github: service}
}

// Dependabot enables squash auto-merge or merges ready Dependabot pull requests.
type Dependabot struct {
	github github.Service
}

// Matches identifies eligible Dependabot pull requests that need a merge action.
func (s *Dependabot) Matches(repository *github.Repository, pull *github.PullRequest) bool {
	return Eligible(pull) && s.github.IsSameRepository(repository, pull) && isDependabot(pull) &&
		(readyToMerge(pull) || pull.GetAutoMerge() == nil)
}

// Assess selects the Dependabot merge action using the same policy as reconciliation.
func (s *Dependabot) Assess(_ context.Context, repository *github.Repository, pull *github.PullRequest) (Decision, error) {
	decision, err := decideMerge(repository, pull)
	decision.Policy = "Dependabot"
	return decision, err
}

// Reconcile refreshes the pull request before merging or enabling auto-merge.
func (s *Dependabot) Reconcile(ctx context.Context, repository *github.Repository, expected *github.PullRequest) error {
	current, err := s.github.GetPullRequest(ctx, repository, expected.GetNumber())
	if err != nil {
		return err
	}

	if !s.Matches(repository, current) {
		return nil
	}

	if !s.github.IsSamePullRequest(current, expected) {
		return errors.New("pull request changed before merge; retry on the next run")
	}

	decision, err := s.Assess(ctx, repository, current)
	if err != nil {
		return err
	}

	if decision.IsMerge() {
		return s.github.Merge(ctx, repository, current)
	}

	if decision.IsEnable() && current.GetAutoMerge() == nil {
		return s.github.EnableAutoMerge(ctx, current)
	}

	return nil
}

// isDependabot identifies the bot by its GitHub login rather than branch naming.
func isDependabot(pull *github.PullRequest) bool {
	return pull.GetUser().GetLogin() == "dependabot[bot]"
}
