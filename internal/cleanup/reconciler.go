package cleanup

import (
	"fmt"
	"slices"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/alexfalkowski/go-service/v2/time"
)

// NewReconciler constructs stateless cleanup using the shared GitHub service.
// Configuration is optional; the module registers no job when it is nil.
func NewReconciler(cfg *Config, service github.Service) *Reconciler {
	return &Reconciler{config: cfg, github: service}
}

// Reconciler deletes unchanged merged-PR heads and inactive branches without open
// PRs. Default, protected, and excluded branches are retained. Repository settings
// and branch state are rechecked before each guarded deletion.
type Reconciler struct {
	config *Config
	github github.Service
}

// Reconcile discovers repositories and branches afresh, isolates branch failures,
// and stops on cancellation or rate limits. An ambiguous mutation is not retried;
// the next run reads current GitHub state. Health is independent of job outcomes.
func (r *Reconciler) Reconcile(ctx context.Context) error {
	repositories, err := r.github.ListRepositories(ctx)
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-r.config.GetInactiveAfter().Duration())
	var failures error

	for _, repository := range repositories {
		if ctx.Err() != nil {
			return errors.Join(failures, context.Cause(ctx))
		}

		if err := r.reconcileRepository(ctx, repository, cutoff); err != nil {
			failures = errors.Join(failures, fmt.Errorf("cleanup: repository %s: %w", repository.GetFullName(), err))

			if github.IsRateLimited(err) || ctx.Err() != nil {
				return failures
			}
		}
	}

	return failures
}

func (r *Reconciler) reconcileRepository(ctx context.Context, repository *github.Repository, cutoff time.Time) error {
	repository, err := r.github.GetRepository(ctx, repository)
	if err != nil || !eligibleRepository(repository) {
		return err
	}

	if strings.IsAnyEmpty(repository.GetNodeID(), repository.GetDefaultBranch()) {
		return errors.New("cleanup: repository is missing node ID or default branch")
	}

	branches, err := r.listUnusedBranches(ctx, repository)
	if err != nil {
		return err
	}

	var failures error

	for _, branch := range branches {
		if ctx.Err() != nil {
			return errors.Join(failures, context.Cause(ctx))
		}

		if err := r.reconcileBranch(ctx, repository, branch, cutoff); err != nil {
			failures = errors.Join(failures, fmt.Errorf("cleanup: branch %s: %w", branch.GetName(), err))

			if github.IsRateLimited(err) || ctx.Err() != nil {
				return failures
			}
		}
	}

	return failures
}

func (r *Reconciler) listUnusedBranches(ctx context.Context, repository *github.Repository) ([]*github.Branch, error) {
	branches, err := r.github.ListBranches(ctx, repository)
	if err != nil {
		return nil, err
	}

	open, err := r.github.ListOpenPullRequests(ctx, repository)
	if err != nil {
		return nil, err
	}

	if err := r.validateOpenPullRequests(repository, open); err != nil {
		return nil, err
	}

	var unused []*github.Branch

	for _, branch := range branches {
		if r.eligibleBranch(repository, branch) && !r.usedByOpenPullRequest(repository, branch, open) {
			unused = append(unused, branch)
		}
	}

	return unused, nil
}

func (r *Reconciler) reconcileBranch(ctx context.Context, repository *github.Repository, branch *github.Branch, cutoff time.Time) error {
	pulls, err := r.github.ListBranchPullRequests(ctx, repository, branch.GetName())
	if err != nil {
		return err
	}

	if r.usedByOpenPullRequest(repository, branch, pulls) {
		return nil
	}

	if merged := mergedPullRequest(branch, pulls); merged != nil {
		if err := r.validateMergedPullRequest(ctx, repository, merged); err != nil {
			return err
		}

		return r.deleteBranch(ctx, repository, branch)
	}

	commit, err := r.github.GetCommit(ctx, repository, branch.GetCommit().GetSHA())
	if err != nil {
		return err
	}

	if !commit.GetCommit().GetCommitter().GetDate().Before(cutoff) {
		return nil
	}

	return r.deleteBranch(ctx, repository, branch)
}

func (r *Reconciler) validateMergedPullRequest(ctx context.Context, repository *github.Repository, merged *github.PullRequest) error {
	current, err := r.github.GetPullRequest(ctx, repository, merged.GetNumber())
	if err != nil {
		return err
	}

	if !r.github.IsSamePullRequest(current, merged) || current.GetState() != "closed" || !current.GetMerged() {
		return errors.New("cleanup: merged pull request changed before deletion")
	}

	return nil
}

func (r *Reconciler) deleteBranch(ctx context.Context, repository *github.Repository, branch *github.Branch) error {
	currentRepo, err := r.github.GetRepository(ctx, repository)
	if err != nil {
		return err
	}

	if !eligibleRepository(currentRepo) || currentRepo.GetNodeID() != repository.GetNodeID() {
		return errors.New("cleanup: repository changed before deletion")
	}

	current, err := r.github.GetBranch(ctx, currentRepo, branch.GetName())
	if err != nil {
		return err
	}

	if !r.eligibleBranch(currentRepo, current) || current.GetCommit().GetSHA() != branch.GetCommit().GetSHA() {
		return errors.New("cleanup: branch changed before deletion")
	}

	open, err := r.github.ListOpenPullRequests(ctx, currentRepo)
	if err != nil {
		return err
	}

	if err := r.validateOpenPullRequests(currentRepo, open); err != nil {
		return err
	}

	if r.usedByOpenPullRequest(currentRepo, current, open) {
		return nil
	}

	return r.github.DeleteBranch(ctx, currentRepo, current)
}

func (r *Reconciler) eligibleBranch(repository *github.Repository, branch *github.Branch) bool {
	return !branch.GetProtected() && branch.GetName() != repository.GetDefaultBranch() &&
		!slices.Contains(r.config.Exclude, branch.GetName())
}

func (r *Reconciler) validateOpenPullRequests(repository *github.Repository, pulls []*github.PullRequest) error {
	for _, pull := range pulls {
		if err := github.ValidatePullRequestMetadata(pull); err != nil {
			return err
		}

		if !r.github.IsBaseRepository(repository, pull) || pull.GetState() != "open" {
			return errors.New("cleanup: open pull request returned mismatched metadata")
		}
	}

	return nil
}

func (r *Reconciler) usedByOpenPullRequest(repository *github.Repository, branch *github.Branch, pulls []*github.PullRequest) bool {
	for _, pull := range pulls {
		if pull.GetState() != "open" {
			continue
		}

		if pull.GetBase().GetRef() == branch.GetName() ||
			(r.github.IsSameRepository(repository, pull) && pull.GetHead().GetRef() == branch.GetName()) {
			return true
		}
	}

	return false
}

func mergedPullRequest(branch *github.Branch, pulls []*github.PullRequest) *github.PullRequest {
	var merged *github.PullRequest

	for _, pull := range pulls {
		if !pull.GetMergedAt().IsZero() && pull.GetHead().GetSHA() == branch.GetCommit().GetSHA() {
			merged = pull
		}
	}

	return merged
}

func eligibleRepository(repository *github.Repository) bool {
	return !repository.GetArchived() && !repository.GetDisabled() && repository.GetAllowAutoMerge()
}
