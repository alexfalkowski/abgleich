package workflow

import (
	"fmt"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/strings"
)

const staleRevisionReason = "The pull request revision changed. Any findings for the previous revision were discarded."

type staleReview struct {
	Title     string
	Command   string
	Reason    string
	HeadSHA   string
	BaseSHA   string
	RequestID int64
}

func (c *Command) publish(ctx context.Context, task *reviewTask, body string) error {
	current, comments, err := c.authorize(ctx, task)
	if err != nil {
		return err
	}
	if current == nil {
		return c.publishCancellation(ctx, task)
	}

	if c.revisionChanged(task, current) {
		body, err = c.staleMessage(task, staleRevisionReason)
		if err != nil {
			return err
		}
	}

	exists, err := c.hasReview(ctx, current, comments)
	if err != nil || exists {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	return c.github.AddComment(ctx, current.repository, current.request.GetNumber(), body)
}

// Cancellation feedback requires fresh consent, but permits closed and draft PRs.
// Source submission and review publication keep their stricter eligibility checks.
func (c *Command) publishCancellation(ctx context.Context, task *reviewTask) error {
	current, comments, err := c.authorizeCancellation(ctx, task)
	if err != nil || current == nil {
		return err
	}

	body := c.cancellationMessage(current)
	if body == "" {
		return nil
	}

	exists, err := c.hasReview(ctx, current, comments)
	if err != nil || exists {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	return c.github.AddComment(ctx, current.repository, current.request.GetNumber(), body)
}

func (c *Command) publishStaleReview(ctx context.Context, task *reviewTask, reason string) error {
	body, err := c.staleMessage(task, reason)
	if err != nil {
		return err
	}
	return c.publish(ctx, task, body)
}

func (c *Command) staleMessage(task *reviewTask, reason string) (string, error) {
	return render("stale.md.tmpl", staleReview{
		Title: c.profile.Title, Command: c.profile.Name, Reason: reason,
		HeadSHA: task.request.GetHead().GetSHA(), BaseSHA: task.request.GetBase().GetSHA(),
		RequestID: task.comment.GetID(),
	})
}

func (c *Command) cancellationMessage(task *reviewTask) string {
	pull := task.request

	var reason string
	switch {
	case pull.GetState() == github.PullClosed && pull.GetMerged():
		reason = "the pull request was merged"
	case pull.GetState() == github.PullClosed:
		reason = "the pull request was closed"
	case pull.GetState() == github.PullOpen && pull.GetDraft():
		reason = "the pull request became a draft"
	default:
		return strings.Empty
	}

	return fmt.Sprintf("⏹️ %s canceled because %s. Request: [original request](#issuecomment-%d).", c.profile.Title, reason, task.comment.GetID())
}
