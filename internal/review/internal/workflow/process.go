package workflow

import (
	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/time"
)

const inferenceBudgetDivisor = 2

var errReviewExists = errors.New("review: a matching result is already published")

// Process drains this command's queue under its independent scheduled job.
// The caller provides the job deadline and prevents overlapping Process calls.
func (c *Command) Process(ctx context.Context) error {
	return c.queue.Process(ctx)
}

func (c *Command) process(ctx context.Context, task *reviewTask) error {
	task, err := c.recheck(ctx, task)
	if err != nil || task == nil {
		return err
	}

	review, err := c.context(ctx, task)
	if err != nil {
		return c.reportFailure(ctx, task, err)
	}

	task, err = c.recheck(ctx, task)
	if err != nil || task == nil {
		return err
	}

	budget := time.Minute
	if deadline, ok := ctx.Deadline(); ok {
		budget = time.Until(deadline) / inferenceBudgetDivisor
	}

	guard := func(ctx context.Context) error { return c.guard(ctx, task) }
	tools := sourceTools{session: review.source, beforeLookup: guard}
	completion, err := c.provider.Generate(ctx, ai.Prompt{
		Task:            c.definition.Name,
		Model:           c.definition.Model,
		ReasoningEffort: c.definition.ReasoningEffort,
		Instructions:    c.definition.Instructions,
		Input:           review.input,
		Tools:           tools.definitions(),
		BeforeRequest:   guard,
		Timeout:         budget,
	})
	if errors.Is(err, errReviewExists) {
		return nil
	}
	if err != nil {
		return c.reportFailure(ctx, task, err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	return c.publishReview(ctx, task, completion, review)
}

func (c *Command) publishReview(ctx context.Context, task *reviewTask, completion *ai.Completion, review *reviewContext) error {
	publication, err := c.reply(task, completion, review)
	if err != nil {
		return c.reportFailure(ctx, task, err)
	}
	task, err = c.recheck(ctx, task)
	if err != nil || task == nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return c.github.AddReview(ctx, task.repository, task.pullRequest, publication)
}

// recheck returns a refreshed task only while its authorized revisions still match.
// A nil task stops processing, including existing results and stale revisions.
func (c *Command) recheck(ctx context.Context, task *reviewTask) (*reviewTask, error) {
	current, err := c.authorize(ctx, task)
	if err != nil || current == nil {
		return nil, err
	}

	if c.revisionChanged(task, current) {
		return nil, c.publish(ctx, task, c.staleMessage)
	}

	if exists, err := c.hasReview(ctx, current); err != nil || exists {
		return nil, err
	}

	return current, nil
}

func (c *Command) guard(ctx context.Context, task *reviewTask) error {
	current, err := c.authorize(ctx, task)
	if err != nil {
		return err
	}
	if current == nil || c.revisionChanged(task, current) {
		return errors.New("review: authorization or revision changed")
	}

	exists, err := c.hasReview(ctx, current)
	if err != nil {
		return err
	}
	if exists {
		return errReviewExists
	}

	return ctx.Err()
}

func (c *Command) publish(ctx context.Context, task *reviewTask, body string) error {
	current, err := c.authorize(ctx, task)
	if err != nil || current == nil {
		return err
	}

	if c.revisionChanged(task, current) {
		body = c.staleMessage
	}

	if exists, err := c.hasReview(ctx, current); err != nil || exists {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	return c.github.AddComment(ctx, current.repository, current.pullRequest.GetNumber(), body)
}

// reportFailure preserves the operation error and skips feedback after cancellation.
func (c *Command) reportFailure(ctx context.Context, task *reviewTask, err error) error {
	if ctx.Err() != nil {
		return err
	}
	return errors.Join(err, c.publish(ctx, task, c.failureMessage))
}
