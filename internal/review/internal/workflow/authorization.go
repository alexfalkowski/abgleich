package workflow

import (
	"github.com/alexfalkowski/abgleich/internal/commands"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
)

func (c *Command) authorize(ctx context.Context, task *reviewTask) (*reviewTask, []*github.IssueComment, error) {
	current, err := c.refreshTask(ctx, task)
	if err != nil {
		return nil, nil, err
	}

	if current.request.GetNumber() != task.request.GetNumber() || !c.eligible(current.repository, current.request) {
		return nil, nil, nil
	}

	comment, comments, err := c.authorizer.AuthorizeWithHistory(ctx, c.requestCommand(task), current.repository, current.request.GetNumber(), task.comment)
	if err != nil || comment == nil {
		return nil, nil, err
	}

	current.comment = comment
	return current, comments, nil
}

// Cancellation still requires consent and PR identity, while permitting closed and draft PRs.
func (c *Command) authorizeCancellation(ctx context.Context, task *reviewTask) (*reviewTask, []*github.IssueComment, error) {
	current, err := c.refreshTask(ctx, task)
	if err != nil {
		return nil, nil, err
	}

	if !c.github.IsSamePullRequestIdentity(current.repository, current.request, task.request) {
		return nil, nil, errors.New("review: pull request identity changed before cancellation")
	}

	comment, comments, err := c.authorizer.AuthorizeWithHistory(ctx, c.requestCommand(task), current.repository, current.request.GetNumber(), task.comment)
	if err != nil || comment == nil {
		return nil, nil, err
	}

	current.comment = comment
	return current, comments, nil
}

func (c *Command) refreshTask(ctx context.Context, task *reviewTask) (*reviewTask, error) {
	if err := c.github.ValidateInstallation(ctx); err != nil {
		return nil, err
	}

	repo, err := c.github.GetRepository(ctx, task.repository)
	if err != nil {
		return nil, err
	}

	pull, err := c.github.GetPullRequest(ctx, repo, task.request.GetNumber())
	if err != nil {
		return nil, err
	}

	return &reviewTask{repository: repo, request: pull, comment: task.comment}, nil
}

// Preserve the original command across all consent rechecks, including cancellation.
func (c *Command) requestCommand(task *reviewTask) string {
	if commands.Parse(task.comment.GetBody()) == ReviewCommand {
		return ReviewCommand
	}
	return c.profile.Name
}

func (c *Command) eligible(repo *github.Repository, pull *github.PullRequest) bool {
	return pull.GetState() == github.PullOpen &&
		!pull.GetDraft() &&
		c.github.IsSameRepository(repo, pull) &&
		!strings.IsEmpty(pull.GetBase().GetSHA()) &&
		github.ValidatePullRequestMetadata(pull) == nil
}

func (c *Command) revisionChanged(expected, current *reviewTask) bool {
	return !c.github.IsSamePullRequest(current.request, expected.request) ||
		current.request.GetBase().GetSHA() != expected.request.GetBase().GetSHA()
}

// recheck returns a refreshed task only while its authorized revisions still match.
// A nil task stops processing, including existing results and stale revisions.
func (c *Command) recheck(ctx context.Context, task *reviewTask) (*reviewTask, error) {
	current, comments, err := c.authorize(ctx, task)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, c.publishCancellation(ctx, task)
	}

	if c.revisionChanged(task, current) {
		return nil, c.publish(ctx, task, c.staleMessage())
	}

	exists, err := c.hasReview(ctx, current, comments)
	if err != nil || exists {
		return nil, err
	}

	return current, nil
}

func (c *Command) guard(ctx context.Context, task *reviewTask) error {
	current, comments, err := c.authorize(ctx, task)
	if err != nil {
		return err
	}

	if current == nil || c.revisionChanged(task, current) {
		return errors.New("review: authorization or revision changed")
	}

	exists, err := c.hasReview(ctx, current, comments)
	if err != nil {
		return err
	}
	if exists {
		return errReviewStopped
	}

	return ctx.Err()
}
