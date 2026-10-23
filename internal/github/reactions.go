package github

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
)

// AddCommentReaction validates the returned reaction after the command has
// validated installation ownership.
func (s *service) AddCommentReaction(ctx context.Context, repo *Repository, comment *IssueComment, content string) error {
	reaction, _, err := s.client.installation.Reactions.CreateIssueCommentReaction(ctx, repo.GetOwner().GetLogin(), repo.GetName(), comment.GetID(), content)
	if err != nil {
		return err
	}

	if reaction.GetID() <= 0 || reaction.GetContent() != content {
		return errors.New("github comment reaction returned an incomplete or mismatched response")
	}

	return nil
}
