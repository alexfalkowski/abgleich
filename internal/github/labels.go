package github

import (
	"strconv"

	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
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
// whose PR, author ID, and creation timestamp were already validated by GetComment.
func HasCommentMetadata(comment *IssueComment) bool {
	return !strings.IsEmpty(comment.GetUser().GetLogin()) && !comment.GetUpdatedAt().IsZero()
}

// GetComment rejects mismatched comment, author, timestamp, and PR identities.
func (s *service) GetComment(ctx context.Context, repo *Repository, number int, expected *IssueComment) (*IssueComment, error) {
	comment, _, err := s.client.installation.Issues.GetComment(ctx, repo.GetOwner().GetLogin(), repo.GetName(), expected.GetID())
	if err != nil {
		return nil, err
	}

	issueURL := s.client.installation.BaseURL() + "repos/" + repo.GetFullName() + "/issues/" + strconv.Itoa(number)
	sameComment := comment.GetID() == expected.GetID() && comment.GetUser().GetID() == expected.GetUser().GetID() &&
		comment.GetCreatedAt().Equal(expected.GetCreatedAt())
	samePullRequest := strings.ToLower(comment.GetIssueURL()) == strings.ToLower(issueURL)

	if !sameComment || !samePullRequest {
		return nil, errors.New("github comment does not match the requested pull request and author")
	}

	return comment, nil
}

// HasWritePermission reads effective permissions, including maintain and admin.
func (s *service) HasWritePermission(ctx context.Context, repo *Repository, author *User) (bool, error) {
	permission, _, err := s.client.installation.Repositories.GetPermissionLevel(ctx, repo.GetOwner().GetLogin(), repo.GetName(),
		author.GetLogin())
	if err != nil {
		return false, err
	}

	if permission.GetUser().GetID() != author.GetID() {
		return false, errors.New("github permissions returned a different author")
	}

	return permission.GetPermission() == "write" || permission.GetPermission() == "admin", nil
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

	return errors.New("github label response does not contain the requested label")
}
