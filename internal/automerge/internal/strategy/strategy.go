package strategy

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
)

// Strategy checks snapshot eligibility locally, then refreshes state before
// acting. Callers select the first match for a sweep, including no-op and error
// outcomes, so an ambiguous mutation cannot fall through to another strategy.
type Strategy interface {
	// Matches evaluates eligibility from the supplied snapshots without remote calls.
	Matches(repository *github.Repository, pull *github.PullRequest) bool

	// Assess evaluates the selected policy without mutating GitHub. Call only
	// after Matches and metadata validation. Evidence failures return an error
	// with a decision identifying the unavailable evidence.
	Assess(ctx context.Context, repository *github.Repository, pull *github.PullRequest) (Decision, error)

	// Reconcile refreshes remote state before attempting at most one mutation.
	// Pull requests requiring no action return nil; failures are not retried.
	Reconcile(ctx context.Context, repository *github.Repository, pull *github.PullRequest) error
}

func strategies(service github.Service) []Strategy {
	return []Strategy{NewDependabot(service), NewLabel(service), NewRebase(service)}
}
