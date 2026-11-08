package workflow

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
)

// Ready signals accepted manual work for this command's existing job.
func (c *Command) Ready() <-chan struct{} {
	return c.queue.Ready()
}

// ProcessQueue drains manual requests without discovering repositories or
// maintaining review threads. It shares the scheduled job's timeout and exclusion.
func (c *Command) ProcessQueue(ctx context.Context) error {
	return c.queue.Process(ctx)
}

// Process drains this command's queue, then maintains its outdated threads
// within the remaining job budget, preserving failures from both operations.
// The caller provides the job deadline and prevents overlapping Process calls.
func (c *Command) Process(ctx context.Context) error {
	failures := c.queue.Process(ctx)
	if err := ctx.Err(); err != nil {
		return errors.Join(failures, err)
	}
	return errors.Join(failures, c.reconcileThreads(ctx))
}

func (c *Command) process(ctx context.Context, task *reviewTask) error {
	task, err := c.recheck(ctx, task)
	if err != nil || task == nil {
		return err
	}

	review, err := c.context(ctx, task)
	if err != nil {
		return c.reportFailure(ctx, task, stageSource, err)
	}

	return c.generateReview(ctx, task, review)
}
