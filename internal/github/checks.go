package github

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/google/go-github/v93/github"
)

// ListCheckRuns reads all latest check runs for an immutable head SHA. Missing
// pages, inconsistent totals, and checks for another revision reject the result.
func (s *service) ListCheckRuns(ctx context.Context, repo *Repository, sha string) ([]*CheckRun, error) {
	opts := &github.ListCheckRunsOptions{Filter: new("latest")}
	opts.PerPage = pageSize

	var checks []*CheckRun
	seen := make(map[int64]bool)

	for {
		page, response, err := s.client.installation.Checks.ListCheckRunsForRef(ctx, repo.GetOwner().GetLogin(), repo.GetName(), sha, opts)
		if err != nil {
			return nil, errors.Prefix("github: check runs", err)
		}

		if err := validateCheckPage(page, sha, seen); err != nil {
			return nil, err
		}

		checks = append(checks, page.CheckRuns...)

		if response.NextPage == 0 {
			if len(checks) != page.GetTotal() {
				return nil, errors.New("github: check runs returned incomplete pagination")
			}

			return checks, nil
		}

		if response.NextPage <= opts.Page {
			return nil, errors.New("github: check runs returned invalid pagination")
		}

		opts.Page = response.NextPage
	}
}

// ListCommitStatuses reads all latest contexts for an immutable head SHA.
// Missing pages, duplicate contexts, and mismatched revisions reject the result.
func (s *service) ListCommitStatuses(ctx context.Context, repo *Repository, sha string) ([]*CommitStatus, error) {
	opts := &github.ListOptions{PerPage: pageSize}
	var statuses []*CommitStatus
	seen := make(map[string]bool)

	for {
		page, response, err := s.client.installation.Repositories.GetCombinedStatus(ctx, repo.GetOwner().GetLogin(), repo.GetName(), sha, opts)
		if err != nil {
			return nil, errors.Prefix("github: commit statuses", err)
		}

		if err := validateStatusPage(page, sha, seen); err != nil {
			return nil, err
		}

		statuses = append(statuses, page.Statuses...)

		if response.NextPage == 0 {
			if len(statuses) != page.GetTotalCount() {
				return nil, errors.New("github: commit statuses returned incomplete pagination")
			}

			return statuses, nil
		}

		if response.NextPage <= opts.Page {
			return nil, errors.New("github: commit statuses returned invalid pagination")
		}

		opts.Page = response.NextPage
	}
}

func validateCheckPage(page *github.ListCheckRunsResults, sha string, seen map[int64]bool) error {
	if page == nil || page.Total == nil || page.CheckRuns == nil {
		return errors.New("github: check runs returned an incomplete page")
	}

	for _, check := range page.CheckRuns {
		id := check.GetID()
		sameHead := check.GetHeadSHA() == sha
		hasIdentity := id > 0 && !strings.IsAnyEmpty(check.GetName(), check.GetStatus())

		if !hasIdentity || seen[id] || !sameHead {
			return errors.New("github: check run has incomplete or mismatched identity")
		}

		seen[id] = true
	}

	return nil
}

func validateStatusPage(page *github.CombinedStatus, sha string, seen map[string]bool) error {
	if page == nil || page.GetSHA() != sha || page.TotalCount == nil || page.Statuses == nil {
		return errors.New("github: commit statuses returned an incomplete or mismatched page")
	}

	for _, status := range page.Statuses {
		name := status.GetContext()
		if strings.IsAnyEmpty(name, status.GetState()) || seen[name] {
			return errors.New("github: commit status has incomplete or duplicate identity")
		}

		seen[name] = true
	}

	return nil
}
