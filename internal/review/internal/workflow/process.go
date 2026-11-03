package workflow

import (
	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/time"
)

const inferenceBudgetDivisor = 2

var errReviewExists = errors.New("review: a matching result is already published")

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
		return c.reportFailure(ctx, task, err)
	}

	task, err = c.recheck(ctx, task)
	if err != nil || task == nil {
		return err
	}

	budget := time.Minute
	if deadline, ok := ctx.Deadline(); ok {
		budget = time.Until(deadline) / inferenceBudgetDivisor
	}

	guard := func(ctx context.Context) error { return c.guard(ctx, task) }
	tools := sourceTools{session: review.source, beforeLookup: guard}
	completion, err := c.provider.Generate(ctx, ai.Prompt{
		Task:            c.definition.Name,
		Model:           c.definition.Model,
		ReasoningEffort: c.definition.ReasoningEffort,
		Instructions:    c.definition.Instructions,
		Input:           review.input,
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

	return c.publishReview(ctx, task, completion, review)
}

func (c *Command) publishReview(ctx context.Context, task *reviewTask, completion *ai.Completion, review *reviewContext) error {
	publication, err := c.reply(task, completion, review)
	if err != nil {
		return c.reportFailure(ctx, task, err)
	}
	task, err = c.recheck(ctx, task)
	if err != nil || task == nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return c.github.AddReview(ctx, task.repository, task.pullRequest, publication)
}

// recheck returns a refreshed task only while its authorized revisions still match.
// A nil task stops processing, including existing results and stale revisions.
func (c *Command) recheck(ctx context.Context, task *reviewTask) (*reviewTask, error) {
	current, err := c.authorize(ctx, task)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, c.publishCancellation(ctx, task)
	}

	if c.revisionChanged(task, current) {
		return nil, c.publish(ctx, task, c.staleMessage)
	}

	exists, err := c.hasReview(ctx, current)
	if err != nil || exists {
		return nil, err
	}

	return current, nil
}

func (c *Command) guard(ctx context.Context, task *reviewTask) error {
	current, err := c.authorize(ctx, task)
	if err != nil {
		return err
	}
	sameRevision := current != nil && !c.revisionChanged(task, current)
	if !sameRevision {
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
	if err != nil {
		return err
	}
	if current == nil {
		return c.publishCancellation(ctx, task)
	}

	if c.revisionChanged(task, current) {
		body = c.staleMessage
	}

	exists, err := c.hasReview(ctx, current)
	if err != nil || exists {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	return c.github.AddComment(ctx, current.repository, current.pullRequest.GetNumber(), body)
}

// Cancellation feedback requires fresh consent, but permits closed and draft PRs.
// Source submission and review publication keep their stricter eligibility checks.
func (c *Command) publishCancellation(ctx context.Context, task *reviewTask) error {
	current, err := c.authorizeCancellation(ctx, task)
	if err != nil || current == nil {
		return err
	}

	body := c.cancellationMessage(current)
	if body == "" {
		return nil
	}

	exists, err := c.hasReview(ctx, current)
	if err != nil || exists {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	return c.github.AddComment(ctx, current.repository, current.pullRequest.GetNumber(), body)
}

func (c *Command) authorizeCancellation(ctx context.Context, task *reviewTask) (*reviewTask, error) {
	if err := c.github.ValidateInstallation(ctx); err != nil {
		return nil, err
	}

	repo, err := c.github.GetRepository(ctx, task.repository)
	if err != nil {
		return nil, err
	}

	pull, err := c.github.GetPullRequest(ctx, repo, task.pullRequest.GetNumber())
	if err != nil {
		return nil, err
	}

	sameNumber := pull.GetNumber() == task.pullRequest.GetNumber()
	sameNodeID := pull.GetNodeID() == task.pullRequest.GetNodeID()
	sameRepository := c.github.IsBaseRepository(repo, pull)
	if !sameNumber || !sameNodeID || !sameRepository {
		return nil, errors.New("review: pull request identity changed before cancellation")
	}

	comment, err := c.authorizer.Authorize(ctx, c.definition.Name, repo, pull.GetNumber(), task.commandComment)
	if err != nil || comment == nil {
		return nil, err
	}

	return &reviewTask{repository: repo, pullRequest: pull, commandComment: comment}, nil
}

// reportFailure preserves the operation error and skips feedback after cancellation.
func (c *Command) reportFailure(ctx context.Context, task *reviewTask, err error) error {
	if ctx.Err() != nil {
		return err
	}
	return errors.Join(err, c.publish(ctx, task, c.failureMessage))
}
