package github

import (
	"strconv"

	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/google/go-github/v93/github"
)

// AddComment validates publication identity and body without retrying the POST.
func (s *service) AddComment(ctx context.Context, repo *Repository, number int, body string) error {
	comment, _, err := s.client.installation.Issues.CreateComment(ctx, repo.GetOwner().GetLogin(), repo.GetName(), number,
		github.IssueCommentRequest{Body: body})
	if err != nil {
		return err
	}

	if err := s.validateComment(repo, number, comment); err != nil {
		return err
	}

	if comment.GetBody() != body {
		return errors.New("github: comment publication returned a mismatched body")
	}

	return nil
}

// IsAppComment uses API attribution rather than a user-controlled login or marker.
func (s *service) IsAppComment(comment *IssueComment) bool {
	return comment.GetUser().GetType() == "Bot" &&
		comment.GetPerformedViaGithubApp().GetID() == s.config.App.ID
}

// ListComments returns complete history, validating each comment's PR.
func (s *service) ListComments(ctx context.Context, repo *Repository, number int) ([]*IssueComment, error) {
	var comments []*IssueComment
	opts := &github.IssueListCommentsOptions{}
	opts.PerPage = pageSize

	for {
		page, response, err := s.client.installation.Issues.ListComments(ctx, repo.GetOwner().GetLogin(), repo.GetName(), number, opts)
		if err != nil {
			return nil, err
		}

		if err := s.validateComments(repo, number, page); err != nil {
			return nil, err
		}

		comments = append(comments, page...)
		if response.NextPage == 0 {
			return comments, nil
		}

		if response.NextPage <= opts.Page {
			return nil, errors.New("github: comment history has invalid pagination")
		}

		opts.Page = response.NextPage
	}
}

func (s *service) validateComment(repo *Repository, number int, comment *IssueComment) error {
	issueURL := s.client.installation.BaseURL() + "repos/" + repo.GetFullName() + "/issues/" + strconv.Itoa(number)
	if comment.GetID() <= 0 || strings.ToLower(comment.GetIssueURL()) != strings.ToLower(issueURL) {
		return errors.New("github: comment returned an incomplete or mismatched identity")
	}

	return nil
}

func (s *service) validateComments(repo *Repository, number int, comments []*IssueComment) error {
	for _, comment := range comments {
		if err := s.validateComment(repo, number, comment); err != nil {
			return err
		}
	}

	return nil
}
