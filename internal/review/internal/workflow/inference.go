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

	// Keep checks read-only; the failure path owns feedback publication.
	guard := func(ctx context.Context) error {
		return c.guard(ctx, task)
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
