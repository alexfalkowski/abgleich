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
	current, err := c.recheck(ctx, task)
	if current == nil {
		return err
	}

	input, source, err := c.context(ctx, current)
	if err != nil {
		return c.reportFailure(ctx, task, err)
	}

	if current, err := c.recheck(ctx, task); current == nil {
		return err
	}

	budget := time.Minute
	if deadline, ok := ctx.Deadline(); ok {
		budget = time.Until(deadline) / inferenceBudgetDivisor
	}

	guard := func(ctx context.Context) error { return c.guard(ctx, current) }
	tools := sourceTools{session: source, beforeLookup: guard}
	completion, err := c.provider.Generate(ctx, ai.Prompt{
		Task:            c.definition.Name,
		Model:           c.definition.Model,
		ReasoningEffort: c.definition.ReasoningEffort,
		Instructions:    c.definition.Instructions,
		Input:           input,
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

	body, err := c.reply(task, completion, source)
	if err != nil {
		return err
	}
	return c.publish(ctx, task, body)
}

// recheck refreshes authorization and skips existing reviews. A changed
// revision publishes a request to start again and returns no task to process.
// A nil task ends processing, including on error.
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
