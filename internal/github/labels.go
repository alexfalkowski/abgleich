package github

import (
	"strconv"

	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
)

const (
	permissionWrite = "write"
	permissionAdmin = "admin"
)

// HasLabel reports exact, case-sensitive membership. Nil or empty label lists
// contain no matching label.
func HasLabel(labels []*Label, name string) bool {
	for _, label := range labels {
		if label.GetName() == name {
			return true
		}
	}

	return false
}

// HasCommentMetadata checks for an author login and edit timestamp on a comment
// whose PR, author ID, and creation timestamp were already validated by GetComment
// or GetCommentWithHistory.
func HasCommentMetadata(comment *IssueComment) bool {
	return !strings.IsEmpty(comment.GetUser().GetLogin()) && !comment.GetUpdatedAt().IsZero()
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

// HasWritePermission reads effective permissions, including maintain and admin.
func (s *service) HasWritePermission(ctx context.Context, repo *Repository, author *User) (bool, error) {
	permission, _, err := s.client.installation.Repositories.GetPermissionLevel(ctx, repo.GetOwner().GetLogin(), repo.GetName(),
		author.GetLogin())
	if err != nil {
		return false, err
	}

	if permission.GetUser().GetID() != author.GetID() {
		return false, errors.New("github: permissions returned a different author")
	}

	return permission.GetPermission() == permissionWrite || permission.GetPermission() == permissionAdmin, nil
}

// AddLabel accepts success only when the response contains the requested label.
func (s *service) AddLabel(ctx context.Context, repo *Repository, number int, label string) error {
	labels, _, err := s.client.installation.Issues.AddLabelsToIssue(ctx, repo.GetOwner().GetLogin(), repo.GetName(), number, []string{label})
	if err != nil {
		return err
	}

	if HasLabel(labels, label) {
		return nil
	}

	return errors.New("github: label response does not contain the requested label")
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
