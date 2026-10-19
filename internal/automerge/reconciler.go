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

// Reconciler enables auto-merge or merges ready Dependabot PRs in opted-in
// repositories and keeps auto-merge branches current. Failures are isolated.
type Reconciler struct {
	github github.Service
}

// Reconcile discovers PRs, enables Dependabot auto-merge, merges ready Dependabot
// PRs, and rebases stale eligible branches. It joins failures and respects
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

	if !r.eligibleForAutomation(repository, pull) {
		return nil
	}

	head, base := pull.GetHead(), pull.GetBase()
	if strings.IsAnyEmpty(pull.GetNodeID(), head.GetSHA(), base.GetRef()) {
		return errors.New("pull request is missing branch metadata")
	}

	if isDependabot(pull) && (readyToMerge(pull) || pull.GetAutoMerge() == nil) {
		return r.mergeDependabotIfUnchanged(ctx, repository, pull)
	}

	return r.rebaseIfBehind(ctx, repository, pull)
}

func (r *Reconciler) eligibleForAutomation(repository *github.Repository, pull *github.PullRequest) bool {
	return r.eligible(repository, pull) && (pull.GetAutoMerge() != nil || isDependabot(pull))
}

func (r *Reconciler) eligible(repository *github.Repository, pull *github.PullRequest) bool {
	return pull.GetState() == "open" &&
		!pull.GetDraft() &&
		r.github.IsSameRepository(repository, pull)
}

func eligibleRepository(repository *github.Repository) bool {
	return !repository.GetArchived() && !repository.GetDisabled() && repository.GetAllowAutoMerge()
}

func samePullRequest(current, expected *github.PullRequest) bool {
	currentHead, expectedHead := current.GetHead(), expected.GetHead()
	sameID := current.GetNodeID() == expected.GetNodeID() && current.GetNumber() == expected.GetNumber()
	sameHead := currentHead.GetRef() == expectedHead.GetRef() && currentHead.GetSHA() == expectedHead.GetSHA()
	sameBase := current.GetBase().GetRef() == expected.GetBase().GetRef()

	return sameID && sameHead && sameBase
}
