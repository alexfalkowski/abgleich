package github

import (
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/errors"
	"github.com/alexfalkowski/go-service/v3/strings"
	"github.com/google/go-github/v93/github"
)

const branchPrefix = "refs/heads/"

// ListBranches finishes pagination and rejects missing protection/head metadata.
func (s *service) ListBranches(ctx context.Context, repo *Repository) ([]*Branch, error) {
	opts := &github.BranchListOptions{}
	opts.PerPage = pageSize

	var branches []*Branch
	seen := make(map[string]string)

	for {
		page, response, err := s.client.installation.Repositories.ListBranches(ctx, repo.GetOwner().GetLogin(), repo.GetName(), opts)
		if err != nil {
			return nil, errors.Prefix("github: list branches", err)
		}

		if page == nil {
			return nil, errors.New("github: branches returned an incomplete page")
		}

		for _, branch := range page {
			if err := validateBranch(branch); err != nil {
				return nil, err
			}

			if sha, exists := seen[branch.GetName()]; exists {
				if sha != branch.GetCommit().GetSHA() {
					return nil, errors.New("github: branch changed during pagination")
				}

				continue
			}

			seen[branch.GetName()] = branch.GetCommit().GetSHA()
			branches = append(branches, branch)
		}

		if response.NextPage == 0 {
			return branches, nil
		}

		opts.Page = response.NextPage
	}
}

// GetBranch rejects redirected or mismatched branches rather than following a rename.
func (s *service) GetBranch(ctx context.Context, repo *Repository, name string) (*Branch, error) {
	// go-github's GetBranch calls RoundTrip directly, bypassing http.Client.Timeout.
	ctx, cancel := context.WithTimeout(ctx, s.config.GetTimeout())
	defer cancel()

	branch, _, err := s.client.installation.Repositories.GetBranch(ctx, repo.GetOwner().GetLogin(), repo.GetName(), name, 0)
	if err != nil {
		return nil, errors.Prefix("github: get branch", err)
	}

	if err := validateBranch(branch); err != nil {
		return nil, err
	}

	if branch.GetName() != name {
		return nil, errors.New("github: branch returned a different name")
	}

	return branch, nil
}

// ListBranchPullRequests reads all states so an open PR blocks merged/inactive cleanup.
func (s *service) ListBranchPullRequests(ctx context.Context, repo *Repository, branch string) ([]*PullRequest, error) {
	opts := &github.PullRequestListOptions{
		State:     "all",
		Head:      repo.GetOwner().GetLogin() + ":" + branch,
		Sort:      "created",
		Direction: "asc",
		PerPage:   pageSize,
	}

	var pulls []*PullRequest

	for {
		page, response, err := s.client.installation.PullRequests.List(ctx, repo.GetOwner().GetLogin(), repo.GetName(), opts)
		if err != nil {
			return nil, errors.Prefix("github: branch pull requests", err)
		}

		if page == nil {
			return nil, errors.New("github: branch pull requests returned an incomplete page")
		}

		for _, pull := range page {
			if err := s.validateBranchPullRequest(repo, branch, pull); err != nil {
				return nil, err
			}

			pulls = append(pulls, pull)
		}

		if response.NextPage == 0 {
			return pulls, nil
		}

		opts.Page = response.NextPage
	}
}

func (s *service) validateBranchPullRequest(repo *Repository, branch string, pull *PullRequest) error {
	if err := ValidatePullRequestMetadata(pull); err != nil {
		return err
	}

	if !s.IsSameRepository(repo, pull) || pull.GetHead().GetRef() != branch ||
		(pull.GetState() != PullOpen && pull.GetState() != PullClosed) {
		return errors.New("github: branch pull request returned mismatched metadata")
	}

	return nil
}

func validateBranch(branch *Branch) error {
	if branch == nil || strings.IsAnyEmpty(branch.GetName(), branch.GetCommit().GetSHA()) || branch.Protected == nil {
		return errors.New("github: branch is missing name, head, or protection metadata")
	}

	return nil
}
