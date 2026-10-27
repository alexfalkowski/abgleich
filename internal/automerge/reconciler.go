package automerge

import (
	"fmt"

	"github.com/alexfalkowski/abgleich/internal/automerge/internal/strategy"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
)

// NewReconciler constructs owner-wide reconciliation using strategies in the
// supplied order. Each invocation discovers fresh repositories and pull requests
// without retaining progress between runs.
func NewReconciler(service github.Service, cfg *Config, strategies []strategy.Strategy) *Reconciler {
	return &Reconciler{github: service, config: cfg, strategies: strategies}
}

// Reconciler automates Dependabot and labeled PRs in opted-in repositories and
// keeps same-repository auto-merge branches current. Failures are isolated.
type Reconciler struct {
	github     github.Service
	config     *Config
	strategies []strategy.Strategy
}

// Reconcile discovers PRs, merges or enables auto-merge for eligible PRs, and
// rebases stale eligible branches. It joins failures and respects
// cancellation. Mutations are attempted once per PR; a later invocation discovers
// remote state after partial success or an ambiguous response.
func (r *Reconciler) Reconcile(ctx context.Context) error {
	repositories, err := r.github.ListRepositories(ctx)
	if err != nil {
		return err
	}

	var failures error

	for _, repository := range repositories {
		if ctx.Err() != nil {
			return errors.Join(failures, context.Cause(ctx))
		}

		if err := r.reconcileRepository(ctx, repository, false); err != nil {
			failures = errors.Join(failures, fmt.Errorf("automerge: repository %s: %w", repository.GetFullName(), err))

			if github.IsRateLimited(err) || ctx.Err() != nil {
				return failures
			}
		}
	}

	return failures
}

// ReconcileRepository completes fresh installation discovery before checking
// one repository. Missing or no longer accessible repositories need no action;
// event payloads never supply repository settings or pull request state.
// Rebasing requires the configured completion signal on the current head.
func (r *Reconciler) ReconcileRepository(ctx context.Context, name string) error {
	repository, err := r.github.FindRepository(ctx, name)
	if err != nil || repository == nil {
		return err
	}

	return r.reconcileRepository(ctx, repository, true)
}

func (r *Reconciler) reconcileRepository(ctx context.Context, repository *github.Repository, waitForCI bool) error {
	repository, err := r.github.GetRepository(ctx, repository)
	if err != nil {
		return err
	}

	if !eligibleRepository(repository) {
		return nil
	}

	pulls, err := r.github.ListOpenPullRequests(ctx, repository)
	if err != nil {
		return err
	}

	var failures error
	for _, pull := range pulls {
		if ctx.Err() != nil {
			return errors.Join(failures, context.Cause(ctx))
		}

		if !r.eligibleForAutomation(repository, pull) {
			continue
		}

		if err := r.reconcilePullRequest(ctx, repository, pull.GetNumber(), waitForCI); err != nil {
			failures = errors.Join(failures, fmt.Errorf("automerge: pull request #%d: %w", pull.GetNumber(), err))

			if github.IsRateLimited(err) || ctx.Err() != nil {
				return failures
			}
		}
	}

	return failures
}

func (r *Reconciler) reconcilePullRequest(ctx context.Context, repository *github.Repository, number int, waitForCI bool) error {
	pull, err := r.github.GetPullRequest(ctx, repository, number)
	if err != nil {
		return err
	}

	if !r.eligibleForAutomation(repository, pull) {
		return nil
	}

	if err := github.ValidatePullRequestMetadata(pull); err != nil {
		return err
	}

	for _, s := range r.strategies {
		if s.Matches(repository, pull) {
			if rebase, ok := s.(*strategy.Rebase); ok && waitForCI {
				return rebase.ReconcileAfterChecks(ctx, repository, pull, r.config.GetCompletionCheck())
			}

			return s.Reconcile(ctx, repository, pull)
		}
	}

	return nil
}

func (r *Reconciler) eligibleForAutomation(repository *github.Repository, pull *github.PullRequest) bool {
	for _, s := range r.strategies {
		if s.Matches(repository, pull) {
			return true
		}
	}

	return false
}

func eligibleRepository(repository *github.Repository) bool {
	return !repository.GetArchived() && !repository.GetDisabled() && repository.GetAllowAutoMerge()
}
