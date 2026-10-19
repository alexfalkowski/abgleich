package automerge

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
)

func (r *Reconciler) rebaseIfBehind(ctx context.Context, repository *github.Repository, pull *github.PullRequest) error {
	comparison, err := r.github.Compare(ctx, repository, pull.GetBase().GetRef(), pull.GetHead().GetSHA())
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

	if !r.eligible(repository, current) || current.GetAutoMerge() == nil {
		return nil
	}

	if !samePullRequest(current, expected) {
		return errors.New("pull request changed before rebase; retry on the next run")
	}

	return r.github.Rebase(ctx, current)
}
