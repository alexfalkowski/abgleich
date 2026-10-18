package automerge

import (
	"fmt"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
)

// NewReconciler constructs owner-wide reconciliation. Each invocation discovers
// fresh repositories and pull requests without retaining progress between runs.
func NewReconciler(service github.Service) *Reconciler {
	return &Reconciler{github: service}
}

// Reconciler keeps non-draft, same-repository auto-merge branches current with
// their bases. It continues processing other pull requests after a failure.
type Reconciler struct {
	github github.Service
}

// Reconcile discovers current pull requests and requests rebases for stale eligible
// branches. It returns joined failures and respects cancellation. Mutations are
// attempted once per pull request; a later invocation discovers remote state
// after partial success or an ambiguous response.
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

		if err := r.reconcileRepository(ctx, repository); err != nil {
			failures = errors.Join(failures, fmt.Errorf("repository %s: %w", repository.GetFullName(), err))

			if github.IsRateLimited(err) || ctx.Err() != nil {
				return failures
			}
		}
	}

	return failures
}

func (r *Reconciler) reconcileRepository(ctx context.Context, repository *github.Repository) error {
	pulls, err := r.github.ListOpenPullRequests(ctx, repository)
	if err != nil {
		return err
	}

	var failures error
	for _, pull := range pulls {
		if ctx.Err() != nil {
			return errors.Join(failures, context.Cause(ctx))
		}

		if !r.eligible(repository, pull) {
			continue
		}

		if err := r.reconcilePullRequest(ctx, repository, pull.GetNumber()); err != nil {
			failures = errors.Join(failures, fmt.Errorf("pull request #%d: %w", pull.GetNumber(), err))

			if github.IsRateLimited(err) || ctx.Err() != nil {
				return failures
			}
		}
	}

	return failures
}

func (r *Reconciler) reconcilePullRequest(ctx context.Context, repository *github.Repository, number int) error {
	pull, err := r.github.GetPullRequest(ctx, repository, number)
	if err != nil {
		return err
	}

	if !r.eligible(repository, pull) {
		return nil
	}

	head, base := pull.GetHead(), pull.GetBase()
	if strings.IsAnyEmpty(pull.GetNodeID(), head.GetSHA(), base.GetRef()) {
		return errors.New("pull request is missing branch metadata")
	}

	comparison, err := r.github.Compare(ctx, repository, base.GetRef(), head.GetSHA())
	if err != nil {
		return err
	}

	if comparison.GetBehindBy() <= 0 {
		return nil
	}

	return r.rebaseIfUnchanged(ctx, repository, pull)
}

func (r *Reconciler) rebaseIfUnchanged(ctx context.Context, repository *github.Repository, expected *github.PullRequest) error {
	current, err := r.github.GetPullRequest(ctx, repository, expected.GetNumber())
	if err != nil {
		return err
	}

	if !r.eligible(repository, current) {
		return nil
	}

	currentHead, expectedHead := current.GetHead(), expected.GetHead()
	sameID := current.GetNodeID() == expected.GetNodeID()
	sameHead := currentHead.GetRef() == expectedHead.GetRef() && currentHead.GetSHA() == expectedHead.GetSHA()
	sameBase := current.GetBase().GetRef() == expected.GetBase().GetRef()
	if !sameID || !sameHead || !sameBase {
		return errors.New("pull request changed before rebase; retry on the next run")
	}

	return r.github.Rebase(ctx, current)
}

func (r *Reconciler) eligible(repository *github.Repository, pull *github.PullRequest) bool {
	return pull.GetState() == "open" &&
		!pull.GetDraft() &&
		pull.GetAutoMerge() != nil &&
		r.github.IsSameRepository(repository, pull)
}
