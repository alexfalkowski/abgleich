package github

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/google/go-github/v93/github"
)

// PullOpen identifies an open pull request in REST responses and filters.
const PullOpen = "open"

// PullClosed includes merged pull requests in REST responses.
const PullClosed = "closed"

// MergeableClean indicates that GitHub reports no mergeability blockers.
const MergeableClean = "clean"

// MergeableUnstable indicates non-passing checks while the pull request remains mergeable.
const MergeableUnstable = "unstable"

// HasPullRequestNodeID checks the identity required by GraphQL operations.
// A numeric pull request ID does not substitute for its node ID.
func HasPullRequestNodeID(pull *PullRequest) bool {
	return !strings.IsEmpty(pull.GetNodeID())
}

// ValidatePullRequestMetadata requires a node ID, head SHA, branch references,
// and repository names before guarded automation. Callers decide which PRs need
// this validation, so incomplete snapshots of ineligible PRs can be ignored.
func ValidatePullRequestMetadata(pull *PullRequest) error {
	head, base := pull.GetHead(), pull.GetBase()
	if strings.IsAnyEmpty(
		pull.GetNodeID(),
		head.GetSHA(),
		head.GetRef(),
		head.GetRepo().GetFullName(),
		base.GetRef(),
		base.GetRepo().GetFullName(),
	) {
		return errors.New("github: pull request is missing branch metadata")
	}

	return nil
}

// ListOpenPullRequests returns all open pull requests for repo, oldest first by
// creation time. Every page must succeed before results are returned, so callers
// can finish pagination before mutating branches.
func (s *service) ListOpenPullRequests(ctx context.Context, repo *Repository) ([]*PullRequest, error) {
	var pulls []*PullRequest
	seen := make(map[int]bool)
	opts := &github.PullRequestListOptions{State: PullOpen, Sort: "created", Direction: "asc", PerPage: pageSize}

	for {
		page, response, err := s.client.installation.PullRequests.List(ctx, repo.GetOwner().GetLogin(), repo.GetName(), opts)
		if err != nil {
			return nil, errors.Prefix("github: list pull requests", err)
		}

		if page == nil {
			return nil, errors.New("github: pull requests returned an incomplete page")
		}

		// New pull requests can shift page boundaries while discovery is running.
		for _, pull := range page {
			if seen[pull.GetNumber()] {
				continue
			}

			seen[pull.GetNumber()] = true
			pulls = append(pulls, pull)
		}

		if response.NextPage == 0 {
			break
		}

		opts.Page = response.NextPage
	}

	return pulls, nil
}

// GetPullRequest reads a fresh pull request snapshot by number from repo,
// including its current auto-merge and branch state.
func (s *service) GetPullRequest(ctx context.Context, repo *Repository, number int) (*PullRequest, error) {
	pull, _, err := s.client.installation.PullRequests.Get(ctx, repo.GetOwner().GetLogin(), repo.GetName(), number)
	if err != nil {
		return nil, errors.Prefix("github: get pull request", err)
	}

	return pull, nil
}
