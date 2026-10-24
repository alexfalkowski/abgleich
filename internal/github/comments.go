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

	issueURL := s.client.installation.BaseURL() + "repos/" + repo.GetFullName() + "/issues/" + strconv.Itoa(number)
	if comment.GetID() <= 0 || comment.GetBody() != body || strings.ToLower(comment.GetIssueURL()) != strings.ToLower(issueURL) {
		return errors.New("github comment publication returned an incomplete or mismatched response")
	}

	return nil
}
