package github

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/google/go-github/v93/github"
)

// ListPullRequestFiles requires complete pagination and a file count matching
// the supplied pull request snapshot.
func (s *service) ListPullRequestFiles(ctx context.Context, repo *Repository, pull *PullRequest) ([]*CommitFile, error) {
	var files []*CommitFile
	seen := make(map[string]bool)
	opts := &github.ListOptions{PerPage: pageSize}
	for {
		page, response, err := s.client.installation.PullRequests.ListFiles(ctx, repo.GetOwner().GetLogin(), repo.GetName(), pull.GetNumber(), opts)
		if err != nil {
			return nil, err
		}

		if err := validateFilePage(page, seen); err != nil {
			return nil, err
		}

		files = append(files, page...)

		if response.NextPage == 0 {
			return completeFileListing(pull, files)
		}

		if response.NextPage <= opts.Page {
			return nil, errors.New("github: file listing has invalid pagination")
		}

		opts.Page = response.NextPage
	}
}

func validateFilePage(files []*CommitFile, seen map[string]bool) error {
	for _, file := range files {
		if strings.IsEmpty(file.GetFilename()) || seen[file.GetFilename()] {
			return errors.New("github: file listing returned an incomplete or duplicate file")
		}

		seen[file.GetFilename()] = true
	}

	return nil
}

func completeFileListing(pull *PullRequest, files []*CommitFile) ([]*CommitFile, error) {
	if pull.ChangedFiles == nil || len(files) != pull.GetChangedFiles() {
		return nil, errors.New("github: pull request file listing is incomplete or changed")
	}

	return files, nil
}
