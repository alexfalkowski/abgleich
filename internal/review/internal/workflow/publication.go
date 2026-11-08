package workflow

import (
	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-sync"
)

func (c *Command) publishReview(ctx context.Context, task *reviewTask, completion *ai.Completion, review *reviewContext) error {
	reply, err := c.reply(task, completion, review)
	if err != nil {
		return c.reportFailure(ctx, task, stageResult, err)
	}

	result := make(chan error, 1)
	err = c.publication.Schedule(ctx, sync.Hook{OnRun: func(ctx context.Context) error {
		result <- c.submitReview(ctx, task, reply)
		return nil
	}})
	if err != nil {
		return err
	}

	// The feature owns this operation through completion, including cancellation.
	return <-result
}

func (c *Command) submitReview(ctx context.Context, task *reviewTask, reply *reviewReply) error {
	task, err := c.recheck(ctx, task)
	if err != nil || task == nil {
		return err
	}

	if reply.Event == github.ReviewApprove {
		blocked, err := c.otherReviewRequestsChanges(ctx, task)
		if err != nil {
			return err
		}
		if blocked {
			reply.Event = github.ReviewComment
			reply.Decision = "Comment. Approval withheld because the other review still requests changes."
		}
	}

	body, err := render("review.md.tmpl", reply)
	if err != nil {
		return c.reportFailure(ctx, task, stageResult, err)
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	return c.github.AddReview(ctx, task.repository, task.request, &github.PullRequestReviewRequest{
		Body: new(body), Comments: reply.Comments, Event: new(reply.Event),
	})
}
