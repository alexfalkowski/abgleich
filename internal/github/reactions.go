package github

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
)

// Reaction selects GitHub reaction content for command feedback.
type Reaction string

// ReactionEyes acknowledges receipt of a command.
const ReactionEyes Reaction = "eyes"

// ReactionThumbsUp acknowledges an accepted command.
const ReactionThumbsUp Reaction = "+1"

// ReactionConfused reports a declined command.
const ReactionConfused Reaction = "confused"

// String returns the reaction content accepted by GitHub.
func (r Reaction) String() string {
	return string(r)
}

// AddCommentReaction validates the returned reaction after the command has
// validated installation ownership.
func (s *service) AddCommentReaction(ctx context.Context, repo *Repository, comment *IssueComment, content Reaction) error {
	reaction, _, err := s.client.installation.Reactions.CreateIssueCommentReaction(ctx, repo.GetOwner().GetLogin(), repo.GetName(), comment.GetID(), content.String())
	if err != nil {
		return err
	}

	if reaction.GetID() <= 0 || reaction.GetContent() != content.String() {
		return errors.New("github: comment reaction returned an incomplete or mismatched response")
	}

	return nil
}
