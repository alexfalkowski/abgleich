package strategy

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
)

var _ Strategy = (*Dependabot)(nil)

// NewDependabot constructs the policy for same-repository Dependabot pull requests.
// Excluded repositories retain explicit label opt-in and existing auto-merge rebasing.
// Nil or empty excludedRepositories preserves author-based automation everywhere.
func NewDependabot(service github.Service, excludedRepositories []string) *Dependabot {
	return &Dependabot{github: service, exclude: excludedRepositories}
}

// Dependabot enables squash auto-merge or merges ready Dependabot pull requests.
type Dependabot struct {
	github  github.Service
	exclude []string
}

// Matches selects eligible Dependabot PRs. Excluded PRs without explicit opt-in
// match a no-op decision for status; labels and enabled auto-merge fall through
// to their respective strategies.
func (s *Dependabot) Matches(repository *github.Repository, pull *github.PullRequest) bool {
	if !Eligible(pull) || !s.github.IsSameRepository(repository, pull) || !isDependabot(pull) {
		return false
	}

	if s.excludesRepository(repository) {
		return pull.GetAutoMerge() == nil && !github.HasLabel(pull.GetLabels(), AutoMergeLabel)
	}

	return readyToMerge(pull) || pull.GetAutoMerge() == nil
}

// Assess selects the Dependabot merge action using the same policy as reconciliation.
func (s *Dependabot) Assess(_ context.Context, repository *github.Repository, pull *github.PullRequest) (Decision, error) {
	if s.excludesRepository(repository) {
		return Decision{Policy: "Dependabot", Reason: ReasonDependabotExcluded}, nil
	}

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
		return errors.New("strategy: pull request changed before merge; retry on the next run")
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

func (s *Dependabot) excludesRepository(repository *github.Repository) bool {
	for _, name := range s.exclude {
		if s.github.IsRepository(repository, name) {
			return true
		}
	}

	return false
}

// isDependabot identifies the bot by its GitHub login rather than branch naming.
func isDependabot(pull *github.PullRequest) bool {
	return pull.GetUser().GetLogin() == "dependabot[bot]"
}
