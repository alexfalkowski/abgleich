package automerge

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
)

func (r *Reconciler) mergeDependabotIfUnchanged(ctx context.Context, repository *github.Repository, expected *github.PullRequest) error {
	current, err := r.github.GetPullRequest(ctx, repository, expected.GetNumber())
	if err != nil {
		return err
	}

	if !r.eligible(repository, current) || !isDependabot(current) {
		return nil
	}

	if !samePullRequest(current, expected) {
		return errors.New("pull request changed before merge; retry on the next run")
	}

	if !repository.GetAllowSquashMerge() {
		return errors.New("repository does not allow squash merging")
	}

	if readyToMerge(current) {
		return r.github.Merge(ctx, repository, current)
	}

	if current.GetAutoMerge() == nil {
		return r.github.EnableAutoMerge(ctx, current)
	}

	return nil
}

func isDependabot(pull *github.PullRequest) bool {
	return pull.GetUser().GetLogin() == "dependabot[bot]"
}

func readyToMerge(pull *github.PullRequest) bool {
	return pull.GetMergeable() && pull.GetMergeableState() == "clean"
}
