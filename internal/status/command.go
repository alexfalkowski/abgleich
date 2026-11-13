package status

import (
	"encoding/hex"

	"github.com/alexfalkowski/abgleich/internal/automerge"
	"github.com/alexfalkowski/abgleich/internal/commands"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v3/bytes"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/errors"
	"github.com/alexfalkowski/go-service/v3/time"
)

const statusCommand = "status"

var _ commands.Handler = (*Command)(nil)

// NewCommand constructs authorized status reporting using automerge policy assessments.
func NewCommand(service github.Service, reconciler *automerge.Reconciler, authorizer *commands.Authorizer) *Command {
	return &Command{github: service, automerge: reconciler, authorizer: authorizer}
}

// Command publishes one bounded policy explanation within the delivery
// deadline. It never requests automation; redelivery can publish another reply.
type Command struct {
	github     github.Service
	automerge  *automerge.Reconciler
	authorizer *commands.Authorizer
}

// Handle revalidates the requester and observed state before publishing. A
// changed snapshot or unconfirmed publication fails without a mutation retry.
func (c *Command) Handle(ctx context.Context, event *github.IssueCommentEvent) (bool, error) {
	repo, err := c.github.GetRepository(ctx, event.GetRepo())
	if err != nil {
		return false, err
	}

	pull, err := c.github.GetPullRequest(ctx, repo, event.GetIssue().GetNumber())
	if err != nil {
		return false, err
	}

	if pull.GetNumber() != event.GetIssue().GetNumber() || !c.github.IsBaseRepository(repo, pull) {
		return false, commands.ErrPullRequestChanged
	}

	if comment, err := c.authorizer.Authorize(ctx, statusCommand, repo, pull.GetNumber(), event.GetComment()); err != nil || comment == nil {
		return false, err
	}

	// Only the validated SHA is interpolated from PR text. All policy prose is
	// fixed, so remote titles, branch names, and errors cannot expand the reply.
	sha := pull.GetHead().GetSHA()
	if _, err := hex.DecodeString(sha); err != nil || len(sha) != 40 {
		return false, errors.New("status: pull request has invalid head SHA")
	}

	assessment := c.automerge.Assess(ctx, repo, pull)
	checkedAt := time.Now().UTC()
	if err := c.recheck(ctx, repo, pull); err != nil {
		return false, err
	}

	message := reply(assessment, pull, event.GetComment().GetID(), checkedAt)
	return c.publish(ctx, repo, pull, event.GetComment(), message)
}

func (c *Command) publish(ctx context.Context, repo *github.Repository, pull *github.PullRequest, comment *github.IssueComment, message Reply) (bool, error) {
	if current, err := c.authorizer.Authorize(ctx, statusCommand, repo, pull.GetNumber(), comment); err != nil || current == nil {
		return false, err
	}

	if err := ctx.Err(); err != nil {
		return false, err
	}

	var body bytes.Buffer
	if err := statusTemplate.Execute(&body, message); err != nil {
		return false, err
	}

	if err := c.github.AddComment(ctx, repo, pull.GetNumber(), body.String()); err != nil {
		return false, err
	}

	return true, nil
}

func (c *Command) recheck(ctx context.Context, repo *github.Repository, pull *github.PullRequest) error {
	currentRepo, err := c.github.GetRepository(ctx, repo)
	if err != nil {
		return err
	}

	current, err := c.github.GetPullRequest(ctx, currentRepo, pull.GetNumber())
	if err != nil {
		return err
	}

	if !c.automerge.SamePolicyState(currentRepo, repo, current, pull) {
		return commands.ErrPullRequestChanged
	}

	return nil
}
