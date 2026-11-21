package workflow

import (
	"github.com/alexfalkowski/abgleich/internal/review/internal/assessment"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-sync"
)

func (c *Command) publishReview(ctx context.Context, task *reviewTask, draft *assessment.Draft, review *reviewEvidence) error {
	result := make(chan error, 1)
	err := c.publication.Schedule(ctx, sync.Hook{OnRun: func(ctx context.Context) error {
		result <- c.submitReview(ctx, task, draft, review)
		return nil
	}})
	if err != nil {
		return err
	}

	// The feature owns this operation through completion, including cancellation.
	return <-result
}

func (c *Command) submitReview(ctx context.Context, task *reviewTask, draft *assessment.Draft, review *reviewEvidence) error {
	task, view, err := c.recheck(ctx, task)
	if err != nil || task == nil {
		return err
	}

	reader := c.history(task)
	reason, err := reader.CheckPrevious(ctx, view, review.previous, review.followup)
	if err != nil {
		return err
	}
	if reason != "" {
		return c.publishStaleReview(ctx, task, reason)
	}

	if draft.Approves() {
		blocked, err := reader.OtherFeatureBlocks(ctx, view)
		if err != nil {
			return err
		}
		if blocked {
			draft.WithholdApproval()
		}
	}

	request, err := draft.Request()
	if err != nil {
		return c.reportFailure(ctx, task, stageResult, err)
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	return c.github.AddReview(ctx, task.repository, task.request, request)
}
