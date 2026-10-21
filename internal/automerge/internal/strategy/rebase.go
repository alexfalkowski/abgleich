package strategy

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
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
	return eligible(pull) && s.github.IsSameRepository(repository, pull) && pull.GetAutoMerge() != nil
}

// Reconcile compares branch state and rebases only after a fresh identity check.
func (s *Rebase) Reconcile(ctx context.Context, repository *github.Repository, pull *github.PullRequest) error {
	comparison, err := s.github.Compare(ctx, repository, pull.GetBase().GetRef(), pull.GetHead().GetSHA())
	if err != nil {
		return err
	}

	if comparison.GetBehindBy() <= 0 {
		return nil
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
		return errors.New("pull request changed before rebase; retry on the next run")
	}

	return s.github.Rebase(ctx, current)
}
