package github

import (
	"strconv"

	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/google/go-github/v93/github"
)

// HasCommentMetadata checks for an author login and edit timestamp on a comment
// whose PR, author ID, and creation timestamp were already validated by GetComment
// or GetCommentWithHistory.
func HasCommentMetadata(comment *IssueComment) bool {
	return !strings.IsEmpty(comment.GetUser().GetLogin()) && !comment.GetUpdatedAt().IsZero()
}

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
	return comment.GetUser().GetType() == UserBot &&
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

// GetComment rejects mismatched comment, author, timestamp, and PR identities.
func (s *service) GetComment(ctx context.Context, repo *Repository, number int, expected *IssueComment) (*IssueComment, error) {
	comment, _, err := s.client.installation.Issues.GetComment(ctx, repo.GetOwner().GetLogin(), repo.GetName(), expected.GetID())
	if err != nil {
		return nil, err
	}

	if err := s.validateCommandComment(repo, number, comment, expected); err != nil {
		return nil, err
	}

	return comment, nil
}

// GetCommentWithHistory reads complete fresh discussion history so command
// consent and result detection can share one snapshot. A missing command returns
// nil with the history; a mismatched identity or incomplete history is an error.
func (s *service) GetCommentWithHistory(ctx context.Context, repo *Repository, number int, expected *IssueComment) (*IssueComment, []*IssueComment, error) {
	comments, err := s.ListComments(ctx, repo, number)
	if err != nil {
		return nil, nil, err
	}

	for _, comment := range comments {
		if comment.GetID() != expected.GetID() {
			continue
		}

		if err := s.validateCommandComment(repo, number, comment, expected); err != nil {
			return nil, nil, err
		}
		return comment, comments, nil
	}

	return nil, comments, nil
}

func (s *service) validateComment(repo *Repository, number int, comment *IssueComment) error {
	issueURL := strings.Concat(s.client.installation.BaseURL(), "repos/", repo.GetFullName(), "/issues/", strconv.Itoa(number))
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

func (s *service) validateCommandComment(repo *Repository, number int, comment, expected *IssueComment) error {
	issueURL := strings.Concat(s.client.installation.BaseURL(), "repos/", repo.GetFullName(), "/issues/", strconv.Itoa(number))
	sameComment := comment.GetID() == expected.GetID() &&
		comment.GetUser().GetID() == expected.GetUser().GetID() &&
		comment.GetCreatedAt().Equal(expected.GetCreatedAt())
	samePullRequest := strings.ToLower(comment.GetIssueURL()) == strings.ToLower(issueURL)

	if !sameComment || !samePullRequest {
		return errors.New("github: comment does not match the requested pull request and author")
	}
	return nil
}
