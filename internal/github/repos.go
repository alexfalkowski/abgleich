package github

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/google/go-github/v93/github"
)

// ListRepositories validates the installation and discovers owned repositories
// it can access, including private repositories.
// Archived, disabled, and foreign repositories are excluded. Every page must
// succeed before any repositories are returned.
func (s *service) ListRepositories(ctx context.Context) ([]*Repository, error) {
	if err := s.ValidateInstallation(ctx); err != nil {
		return nil, err
	}

	opts := &github.ListOptions{PerPage: pageSize}

	var repositories []*Repository

	for {
		page, response, err := s.client.installation.Apps.ListRepos(ctx, opts)
		if err != nil {
			return nil, errors.Prefix("github: list repositories", err)
		}

		if page.GetRepositories() == nil {
			return nil, errors.New("github: installation repositories returned an incomplete page")
		}

		for _, repo := range page.GetRepositories() {
			if !s.isActiveOwnedRepository(repo) {
				continue
			}

			name := repo.GetOwner().GetLogin() + "/" + repo.GetName()
			if strings.IsEmpty(repo.GetName()) || strings.ToLower(repo.GetFullName()) != strings.ToLower(name) {
				return nil, errors.New("github: repository has incomplete or inconsistent identity")
			}

			repositories = append(repositories, repo)
		}

		if response.NextPage == 0 {
			return repositories, nil
		}

		opts.Page = response.NextPage
	}
}

// FindRepository resolves a webhook's repository name through fresh, complete
// installation discovery. Absent or ineligible repositories return nil; failed
// discovery returns an error without a repository.
func (s *service) FindRepository(ctx context.Context, name string) (*Repository, error) {
	repositories, err := s.ListRepositories(ctx)
	if err != nil {
		return nil, err
	}

	for _, repository := range repositories {
		if strings.ToLower(repository.GetFullName()) == strings.ToLower(name) {
			return repository, nil
		}
	}

	return nil, nil
}

// GetRepository reads full repository settings, including AllowAutoMerge and
// AllowSquashMerge, which GitHub omits from repository lists. A mismatched
// repository identity is an error rather than permission to process its PRs.
func (s *service) GetRepository(ctx context.Context, repo *Repository) (*Repository, error) {
	current, _, err := s.client.installation.Repositories.Get(ctx, repo.GetOwner().GetLogin(), repo.GetName())
	if err != nil {
		return nil, errors.Prefix("github: get repository", err)
	}

	sameName := strings.ToLower(current.GetName()) == strings.ToLower(repo.GetName())
	sameOwner := strings.ToLower(current.GetOwner().GetLogin()) == strings.ToLower(repo.GetOwner().GetLogin())
	sameFullName := strings.ToLower(current.GetFullName()) == strings.ToLower(repo.GetFullName())
	if !sameName || !sameOwner || !sameFullName {
		return nil, errors.New("github: repository settings returned no matching repository")
	}

	return current, nil
}

func (s *service) isActiveOwnedRepository(repo *Repository) bool {
	return !repo.GetArchived() && !repo.GetDisabled() &&
		strings.ToLower(repo.GetOwner().GetLogin()) == strings.ToLower(s.config.Owner)
}
