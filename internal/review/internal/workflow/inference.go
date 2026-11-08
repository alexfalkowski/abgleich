package workflow

import (
	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/time"
)

// inferenceTimeoutDivisor limits inference to half the remaining job time,
// leaving time for final checks and publication.
const inferenceTimeoutDivisor = 2

var errReviewStopped = errors.New("review: no further processing is required")

func (c *Command) generateReview(ctx context.Context, task *reviewTask, review *reviewContext) error {
	inferenceTimeout := time.Minute
	if deadline, ok := ctx.Deadline(); ok {
		inferenceTimeout = time.Until(deadline) / inferenceTimeoutDivisor
	}

	needsSubmissionCheck := true
	var submissionCheckErr error
	guard := func(ctx context.Context) error {
		if !needsSubmissionCheck {
			return c.guard(ctx, task)
		}
		// The initial source-submission check can publish cancellation or stale-revision feedback.
		needsSubmissionCheck = false

		task, submissionCheckErr = c.recheck(ctx, task)
		if submissionCheckErr != nil {
			return submissionCheckErr
		}
		if task == nil {
			return errReviewStopped
		}

		return ctx.Err()
	}

	tools := sourceTools{session: review.source, beforeLookup: guard}
	completion, err := c.provider.Generate(ctx, ai.Prompt{
		Task:            c.profile.Name,
		Model:           c.profile.Model,
		ReasoningEffort: c.profile.ReasoningEffort,
		Instructions:    c.profile.Instructions,
		Input:           review.input,
		Tools:           tools.definitions(),
		BeforeRequest:   guard,
		Timeout:         inferenceTimeout,
	})
	// Return submission-check errors directly: feedback may already have been
	// published, including after an ambiguous response.
	if submissionCheckErr != nil {
		return submissionCheckErr
	}
	if errors.Is(err, errReviewStopped) {
		return nil
	}
	if err != nil {
		return c.reportFailure(ctx, task, stageGeneration, err)
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	return c.publishReview(ctx, task, completion, review)
}
