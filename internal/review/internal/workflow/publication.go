package workflow

import (
	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/slices"
	"github.com/alexfalkowski/go-sync"
)

func (c *Command) publishReview(ctx context.Context, task *reviewTask, completion *ai.Completion, review *reviewContext) error {
	reply, err := c.reply(task, completion, review)
	if err != nil {
		return c.reportFailure(ctx, task, stageResult, err)
	}

	result := make(chan error, 1)
	err = c.publication.Schedule(ctx, sync.Hook{OnRun: func(ctx context.Context) error {
		result <- c.submitReview(ctx, task, reply, review)
		return nil
	}})
	if err != nil {
		return err
	}

	// The feature owns this operation through completion, including cancellation.
	return <-result
}

func (c *Command) submitReview(ctx context.Context, task *reviewTask, reply *reviewReply, review *reviewContext) error {
	task, history, err := c.recheck(ctx, task)
	if err != nil || task == nil {
		return err
	}

	reason, err := c.reviewChangeReason(ctx, task, review, history)
	if err != nil {
		return err
	}
	if reason != "" {
		return c.publishStaleReview(ctx, task, reason)
	}

	if reply.Event == github.ReviewApprove {
		blocked, err := c.otherReviewRequestsChanges(ctx, history.reviews)
		if err != nil {
			return err
		}
		if blocked {
			reply.Event = github.ReviewComment
			reply.Decision = "Comment. Approval withheld because the other review still requests changes."
		}
	}

	body, err := reply.body()
	if err != nil {
		return c.reportFailure(ctx, task, stageResult, err)
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	return c.github.AddReview(ctx, task.repository, task.request, &github.PullRequestReviewRequest{
		Body: new(body), Comments: reply.Comments, Event: new(reply.Event.String()),
	})
}

func (c *Command) reviewChangeReason(ctx context.Context, task *reviewTask, review *reviewContext, history reviewHistory) (string, error) {
	authored, err := c.historyAssessments(ctx, history)
	if err != nil {
		return "", err
	}
	if !sameAssessment(review.assessment, lastAssessment(authored)) {
		return "The App review history changed. Findings from this attempt were discarded.", nil
	}
	if review.followup == nil || len(review.followup.references) == 0 {
		return "", nil
	}

	current := c.readBlockers(ctx, task, review.followup.references, authored)
	if !slices.Equal(current, review.followup.Blockers) {
		return "Earlier finding text changed or became unavailable. Findings from this attempt were discarded.", nil
	}
	return "", nil
}

func sameAssessment(previous, current *github.PullRequestReview) bool {
	return previous.GetID() == current.GetID() && previous.GetState() == current.GetState() &&
		previous.GetCommitID() == current.GetCommitID() && previous.GetBody() == current.GetBody()
}
