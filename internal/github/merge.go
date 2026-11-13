package github

import (
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/errors"
	"github.com/alexfalkowski/go-service/v3/strings"
	"github.com/google/go-github/v93/github"
)

const squashMethod = "squash"

// Merge requests a squash merge with the expected head SHA. GitHub enforces
// repository rules. HTTP success without a confirmed merge is an error.
func (s *service) Merge(ctx context.Context, repo *Repository, pull *PullRequest) error {
	result, _, err := s.client.installation.PullRequests.Merge(ctx, repo.GetOwner().GetLogin(), repo.GetName(), pull.GetNumber(),
		strings.Empty, &github.PullRequestOptions{SHA: pull.GetHead().GetSHA(), MergeMethod: squashMethod})
	if err != nil {
		return errors.Prefix("github: merge", err)
	}

	if result == nil || !result.GetMerged() {
		return errors.New("github: merge returned no confirmed merge")
	}

	return nil
}
