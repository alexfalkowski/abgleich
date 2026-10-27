package strategy

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
)

var _ Strategy = (*Rebase)(nil)

// NewRebase constructs the policy for keeping auto-merge branches current.
func NewRebase(service github.Service) *Rebase {
	return &Rebase{github: service}
}

// Rebase updates stale auto-merge branches within the discovered repository.
type Rebase struct {
	github github.Service
}

// Matches identifies eligible same-repository pull requests with auto-merge enabled.
func (s *Rebase) Matches(repository *github.Repository, pull *github.PullRequest) bool {
	return Eligible(pull) && s.github.IsSameRepository(repository, pull) && pull.GetAutoMerge() != nil
}

// Assess compares branch freshness without requesting an update.
func (s *Rebase) Assess(ctx context.Context, repository *github.Repository, pull *github.PullRequest) (Decision, error) {
	comparison, err := s.github.Compare(ctx, repository, pull.GetBase().GetRef(), pull.GetHead().GetSHA())
	if err != nil {
		return Decision{Policy: "rebase", Reason: ReasonComparisonUnavailable}, err
	}

	if comparison.GetBehindBy() <= 0 {
		return Decision{Policy: "rebase", Reason: ReasonBranchCurrent}, nil
	}

	return Decision{Policy: "rebase", Action: "rebase", Reason: ReasonBranchBehind}, nil
}

// Reconcile compares branch state and rebases only after a fresh identity check.
func (s *Rebase) Reconcile(ctx context.Context, repository *github.Repository, pull *github.PullRequest) error {
	return s.reconcile(ctx, repository, pull, "")
}

// ReconcileAfterChecks requires an explicit final CI signal and no active checks
// on this head before requesting a rebase. An empty completionCheck defers to the
// scheduled recovery path. Check conclusions need not be successful.
func (s *Rebase) ReconcileAfterChecks(ctx context.Context, repository *github.Repository, pull *github.PullRequest, completionCheck string) error {
	if strings.IsEmpty(completionCheck) {
		return nil
	}

	return s.reconcile(ctx, repository, pull, completionCheck)
}

func (s *Rebase) reconcile(ctx context.Context, repository *github.Repository, pull *github.PullRequest, completionCheck string) error {
	decision, err := s.Assess(ctx, repository, pull)
	if err != nil {
		return err
	}

	if !decision.IsRebase() {
		return nil
	}

	if !strings.IsEmpty(completionCheck) {
		completed, err := s.checksCompleted(ctx, repository, pull, completionCheck)
		if err != nil || !completed {
			return err
		}
	}

	return s.rebaseIfUnchanged(ctx, repository, pull)
}

// rebaseIfUnchanged rechecks eligibility and branch identity after comparison
// so concurrent changes defer the rebase to a later sweep.
func (s *Rebase) rebaseIfUnchanged(ctx context.Context, repository *github.Repository, expected *github.PullRequest) error {
	current, err := s.github.GetPullRequest(ctx, repository, expected.GetNumber())
	if err != nil {
		return err
	}

	if !s.Matches(repository, current) {
		return nil
	}

	if !s.github.IsSamePullRequest(current, expected) {
		return errors.New("strategy: pull request changed before rebase; retry on the next run")
	}

	return s.github.Rebase(ctx, current)
}
