package threads

import (
	"strconv"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/abgleich/internal/review/internal/history"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/errors"
	"github.com/alexfalkowski/go-service/v3/strings"
)

// Reconciler retires outdated App-owned threads for a single review feature.
type Reconciler struct {
	github  github.Service
	feature string
}

// NewReconciler binds maintenance to a feature's persisted result namespace.
func NewReconciler(service github.Service, feature string) Reconciler {
	return Reconciler{github: service, feature: feature}
}

// Reconcile discovers candidates and refreshes each thread before resolving it.
func (r Reconciler) Reconcile(ctx context.Context) error {
	repositories, err := r.github.ListRepositories(ctx)
	if err != nil {
		return err
	}

	var failures error
	for _, repository := range repositories {
		if err := ctx.Err(); err != nil {
			return errors.Join(failures, err)
		}

		if err := r.reconcileRepositoryThreads(ctx, repository); err != nil {
			failures = errors.Join(failures, errors.Prefix(strings.Concat("review: repository ", repository.GetFullName()), err))
			if github.IsRateLimited(err) || ctx.Err() != nil {
				return failures
			}
		}
	}
	return failures
}

func (r Reconciler) reconcileRepositoryThreads(ctx context.Context, repository *github.Repository) error {
	repository, err := r.github.GetRepository(ctx, repository)
	if err != nil || repository.GetArchived() || repository.GetDisabled() {
		return err
	}
	pulls, err := r.github.ListOpenPullRequests(ctx, repository)
	if err != nil {
		return err
	}

	var failures error
	for _, pull := range pulls {
		if err := ctx.Err(); err != nil {
			return errors.Join(failures, err)
		}

		if err := r.reconcilePullThreads(ctx, repository, pull); err != nil {
			failures = errors.Join(failures, errors.Prefix(strings.Concat("review: pull request #", strconv.Itoa(pull.GetNumber())), err))
			if github.IsRateLimited(err) || ctx.Err() != nil {
				return failures
			}
		}
	}

	return failures
}

func (r Reconciler) reconcilePullThreads(ctx context.Context, repository *github.Repository, pull *github.PullRequest) error {
	if !r.github.IsBaseRepository(repository, pull) {
		return errors.New("review: thread maintenance found a mismatched pull request")
	}
	if !r.eligibleForThreadMaintenance(repository, pull) {
		return nil
	}

	threads, err := r.github.ListReviewThreads(ctx, repository, pull)
	if err != nil {
		return err
	}

	var failures error
	for _, thread := range threads {
		if err := ctx.Err(); err != nil {
			return errors.Join(failures, err)
		}

		if !r.outdatedThread(thread) {
			continue
		}

		if err := r.resolveThread(ctx, repository, pull, thread); err != nil {
			failures = errors.Join(failures, errors.Prefix(strings.Concat("review: thread ", thread.ID), err))
			if github.IsRateLimited(err) || ctx.Err() != nil {
				return failures
			}
		}
	}

	return failures
}

func (r Reconciler) resolveThread(ctx context.Context, repository *github.Repository, pull *github.PullRequest, thread *github.ReviewThread) error {
	current, err := r.github.GetPullRequest(ctx, repository, pull.GetNumber())
	if err != nil {
		return err
	}

	if !r.github.IsSamePullRequestIdentity(repository, current, pull) {
		return errors.New("review: pull request identity changed before thread resolution")
	}

	if !r.eligibleForThreadMaintenance(repository, current) {
		return nil
	}

	fresh, err := r.github.GetReviewThread(ctx, repository, current, thread.ID)
	canResolve := err == nil && fresh != nil && r.outdatedThread(fresh)
	if !canResolve {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	// GitHub has no expected-state guard for resolution. Changes after this
	// final read can still race the mutation; outdated does not mean fixed.
	return r.github.ResolveReviewThread(ctx, repository, current, fresh)
}

func (r Reconciler) eligibleForThreadMaintenance(repository *github.Repository, pull *github.PullRequest) bool {
	return pull.GetState() == github.PullOpen && r.github.IsSameRepository(repository, pull)
}

func (r Reconciler) outdatedThread(thread *github.ReviewThread) bool {
	if !thread.IsAppAuthor || !thread.IsOutdated || thread.IsResolved || !thread.ViewerCanResolve {
		return false
	}

	feature := history.Feature(thread.ReviewBody)
	return !strings.IsEmpty(feature) && feature == r.feature
}
