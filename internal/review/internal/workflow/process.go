package workflow

import (
	"github.com/alexfalkowski/abgleich/internal/review/internal/assessment"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/errors"
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

	return errors.Join(failures, c.threads.Reconcile(ctx))
}

func (c *Command) process(ctx context.Context, task *reviewTask) error {
	task, reviews, err := c.recheck(ctx, task)
	if err != nil || task == nil {
		return err
	}

	review, err := c.collectEvidence(ctx, task, reviews)
	if err != nil {
		return c.reportFailure(ctx, task, stageSource, err)
	}

	completion, err := c.generateReview(ctx, task, review)
	if err != nil || completion == nil {
		return err
	}

	draft, err := assessment.NewDraft(assessment.Input{
		Completion: completion, Source: review.source.Report(), Followup: review.followup,
		Files: review.Files, Patches: review.patches,
		Title: c.profile.Title, HeadSHA: review.HeadSHA, BaseSHA: review.BaseSHA,
		Marker: c.history(task).ResultMarker(), RequestID: task.comment.GetID(),
	})
	if err != nil {
		return c.reportFailure(ctx, task, stageResult, err)
	}

	return c.publishReview(ctx, task, draft, review)
}
