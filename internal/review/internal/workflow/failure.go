package workflow

import (
	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
)

type failureStage string

const (
	stageSource     failureStage = "Source collection"
	stageGeneration failureStage = "AI generation"
	stageResult     failureStage = "Result validation"
)

// String returns the workflow stage's display text.
func (s failureStage) String() string {
	return string(s)
}

// reviewFailure supplies failure reply data with exported fields for template access.
type reviewFailure struct {
	// Provider supplies safe provider diagnostics; nil omits provider details.
	Provider *ai.GenerationError

	// Title labels the review feature in the failure heading and request link.
	Title string

	// Command is the exact /ab suffix shown when requesting another review.
	Command string

	// Model identifies the configured model for the failed review attempt.
	Model string

	// HeadSHA identifies the pull request head revision associated with the failure.
	HeadSHA string

	// BaseSHA identifies the pull request base revision associated with the failure.
	BaseSHA string

	// Stage names the workflow phase that failed.
	Stage failureStage

	// Reason explains the failure without exposing the underlying error text.
	Reason string

	// Recovery tells the maintainer what to check before requesting another review.
	Recovery string

	// RequestID identifies the authorizing comment linked from the failure reply.
	RequestID int64
}

func (c *Command) failureReply(task *reviewTask, stage failureStage, err error) (string, error) {
	failure := reviewFailure{
		Title: c.profile.Title, Command: c.profile.Name, Model: c.profile.Model,
		RequestID: task.comment.GetID(), HeadSHA: task.request.GetHead().GetSHA(), BaseSHA: task.request.GetBase().GetSHA(),
		Stage: stage, Reason: "The review conversation could not finish.",
		Recovery: "Check the worker logs for source lookup or authorization failures.",
	}

	switch stage {
	case stageSource:
		failure.Reason = "The pull request source could not be collected or validated."
		failure.Recovery = "Check GitHub availability, App source permissions, and source validation errors in the worker logs."
	case stageResult:
		failure.Reason = "The model returned an unreadable or invalid review result."
		failure.Recovery = "Check the model's review output format before requesting another review."
	case stageGeneration:
		// The initial reason and recovery describe generation failures.
	}
	if provider, ok := errors.AsType[*ai.GenerationError](err); ok {
		failure.Provider = provider
		failure.Reason, failure.Recovery = provider.Reason, provider.Recovery
	}

	return render("failure.md.tmpl", failure)
}

// reportFailure preserves the operation error and skips feedback after cancellation.
func (c *Command) reportFailure(ctx context.Context, task *reviewTask, stage failureStage, cause error) error {
	if ctx.Err() != nil {
		return cause
	}

	body, err := c.failureReply(task, stage, cause)
	if err != nil {
		return errors.Join(cause, err)
	}

	return errors.Join(cause, c.publish(ctx, task, body))
}
