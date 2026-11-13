package github

import (
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/errors"
	"github.com/google/go-github/v93/github"
)

// GetCommit requires a complete committer timestamp and the requested immutable SHA.
func (s *service) GetCommit(ctx context.Context, repo *Repository, sha string) (*RepositoryCommit, error) {
	commit, _, err := s.client.installation.Repositories.GetCommit(ctx, repo.GetOwner().GetLogin(), repo.GetName(), sha, nil)
	if err != nil {
		return nil, errors.Prefix("github: get commit", err)
	}

	if commit.GetSHA() != sha || commit.GetCommit().GetCommitter().GetDate().IsZero() {
		return nil, errors.New("github: commit returned no matching SHA or committer date")
	}

	return commit, nil
}

// Compare compares head with base in repo and reports missing base commits
// through BehindBy. An absent behind_by field is an error.
func (s *service) Compare(ctx context.Context, repo *Repository, base, head string) (*CommitsComparison, error) {
	comparison, _, err := s.client.installation.Repositories.CompareCommits(ctx, repo.GetOwner().GetLogin(), repo.GetName(),
		base, head, &github.ListOptions{PerPage: 1})
	if err != nil {
		return nil, errors.Prefix("github: compare commits", err)
	}

	if comparison == nil || comparison.BehindBy == nil {
		return nil, errors.New("github: comparison is missing behind_by")
	}

	return comparison, nil
}
