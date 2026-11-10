package workflow

import (
	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/abgleich/internal/commands"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/abgleich/internal/review/internal/sources"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-sync"
)

// ReviewCommand is the exact /ab suffix authorizing all configured review features.
const ReviewCommand = "review"

// ReviewProfile supplies the identity, model, and instructions for
// one review command. Feature packages own these values and registration.
// Name is the exact /ab command suffix and the persisted result namespace.
type ReviewProfile struct {
	// Name is the exact /ab command suffix and the successful result namespace.
	Name string

	// Title labels the review in published replies and failure feedback.
	Title string

	// Model selects the provider model for this feature's review conversations.
	Model string

	// Instructions supplies the feature-owned review prompt.
	Instructions string

	// ReasoningEffort optionally selects provider reasoning effort; empty leaves its default.
	ReasoningEffort ai.ReasoningEffort
}

type reviewTask struct {
	repository *github.Repository
	request    *github.PullRequest
	comment    *github.IssueComment
}

// Command owns shared review consent, revision checks, deduplication, and queued
// work. Each instance has independent state; successful results retain its name.
type Command struct {
	github      github.Service
	provider    ai.Provider
	sources     *sources.Registry
	publication *sync.Worker
	queue       *commands.Queue
	authorizer  *commands.Authorizer
	profile     ReviewProfile
}

// Handle refreshes human consent and queues one revision-specific review. A full
// queue preserves existing command backpressure; inference never holds the slot.
// A nil Command represents a disabled feature and declines the request.
func (c *Command) Handle(ctx context.Context, event *github.IssueCommentEvent) (bool, error) {
	if c == nil {
		return false, nil
	}

	task, comments, err := c.authorize(ctx, &reviewTask{
		repository: event.GetRepo(),
		request:    &github.PullRequest{Number: event.GetIssue().Number},
		comment:    event.GetComment(),
	})
	if err != nil || task == nil {
		return false, err
	}

	if exists, err := c.hasReview(ctx, task, comments); err != nil || exists {
		return exists, err
	}

	if err := ctx.Err(); err != nil {
		return false, err
	}

	err = c.queue.Enqueue(c.resultMarker(task), func(ctx context.Context) error {
		return c.process(ctx, task)
	})

	return err == nil, err
}
