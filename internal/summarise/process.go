package summarise

import (
	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/time"
)

const (
	staleMessage           = "The pull request revision changed. Post a new /ab summarise request."
	failureMessage         = "Summary failed. Post a new /ab summarise request after recovery."
	summaryOutputTokens    = 2048
	inferenceBudgetDivisor = 2
)

func (c *Command) process(ctx context.Context, task *summaryTask) error {
	current, err := c.recheck(ctx, task)
	if err != nil || current == nil {
		return err
	}

	input, files, err := c.context(ctx, current)
	if err != nil {
		return c.reportFailure(ctx, task, err)
	}

	if current, err := c.recheck(ctx, task); err != nil || current == nil {
		return err
	}

	budget := time.Minute
	if deadline, ok := ctx.Deadline(); ok {
		budget = min(budget, time.Until(deadline)/inferenceBudgetDivisor)
	}

	inference, cancel := context.WithTimeout(ctx, budget)
	defer cancel()

	completion, err := c.provider.Generate(inference, ai.Prompt{
		Task:            summaryCommand,
		Model:           c.config.Model,
		Instructions:    instructions,
		Input:           input,
		MaxOutputTokens: summaryOutputTokens,
	})
	if err != nil {
		return c.reportFailure(ctx, task, err)
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	body, err := reply(task, completion.Text, files)
	if err != nil {
		return err
	}

	return c.publish(ctx, task, body)
}

// recheck refreshes authorization and skips existing summaries. A changed
// revision publishes a request to start again and returns no task to process.
func (c *Command) recheck(ctx context.Context, task *summaryTask) (*summaryTask, error) {
	current, err := c.authorize(ctx, task)
	if err != nil || current == nil {
		return nil, err
	}

	if c.revisionChanged(task, current) {
		return nil, c.publish(ctx, task, staleMessage)
	}

	if exists, err := c.hasSummary(ctx, current); err != nil || exists {
		return nil, err
	}

	return current, nil
}

func (c *Command) publish(ctx context.Context, task *summaryTask, body string) error {
	current, err := c.authorize(ctx, task)
	if err != nil || current == nil {
		return err
	}

	if c.revisionChanged(task, current) {
		body = staleMessage
	}

	if exists, err := c.hasSummary(ctx, current); err != nil || exists {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	return c.github.AddComment(ctx, current.repository, current.pullRequest.GetNumber(), body)
}

// reportFailure preserves the operation error and skips feedback after cancellation.
func (c *Command) reportFailure(ctx context.Context, task *summaryTask, err error) error {
	if ctx.Err() != nil {
		return err
	}

	return errors.Join(err, c.publish(ctx, task, failureMessage))
}
