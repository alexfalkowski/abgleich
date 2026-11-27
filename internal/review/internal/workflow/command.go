package workflow

import (
	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/abgleich/internal/commands"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/abgleich/internal/review/internal/history"
	"github.com/alexfalkowski/abgleich/internal/review/internal/sources"
	"github.com/alexfalkowski/abgleich/internal/review/internal/threads"
	"github.com/alexfalkowski/go-service/v3/context"
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

	// ModelConfig selects this feature's provider, model, and optional effort.
	ai.ModelConfig

	// Instructions supplies the feature-owned review prompt.
	Instructions string
}

type reviewTask struct {
	repository *github.Repository
	request    *github.PullRequest
	comment    *github.IssueComment
	command    string
}

func newReviewTask(event *github.IssueCommentEvent, command string) *reviewTask {
	return &reviewTask{
		repository: event.GetRepo(),
		request:    &github.PullRequest{Number: event.GetIssue().Number},
		comment:    event.GetComment(),
		command:    command,
	}
}

// Command owns shared review consent, revision checks, deduplication, and queued
// work. Each instance has independent state; successful results retain its name.
type Command struct {
	github      github.Service
	provider    ai.Provider
	sources     *sources.Registry
	publication *sync.Worker
	queue       *commands.Queue
	consent     consent
	threads     threads.Reconciler
	profile     ReviewProfile
}

// Handle refreshes human consent and queues one revision-specific review.
// A full queue returns [commands.ErrCommandInProgress].
// A nil Command represents a disabled feature and declines the request.
func (c *Command) Handle(ctx context.Context, event *github.IssueCommentEvent) (bool, error) {
	if c == nil {
		return false, nil
	}

	task, comments, err := c.consent.authorize(ctx, newReviewTask(event, c.profile.Name))
	if err != nil || task == nil {
		return false, err
	}

	return c.enqueue(ctx, task, comments)
}

func (c *Command) enqueue(ctx context.Context, task *reviewTask, comments []*github.IssueComment) (bool, error) {
	reader := c.history(task)
	if exists, _, err := reader.Read(ctx, comments); err != nil || exists {
		return exists, err
	}

	if err := ctx.Err(); err != nil {
		return false, err
	}

	err := c.queue.Enqueue(reader.ResultMarker(), func(ctx context.Context) error {
		return c.process(ctx, task)
	})

	return err == nil, err
}

func (c *Command) history(task *reviewTask) history.Reader {
	return history.NewReader(c.github, task.repository, task.request, c.profile.Name)
}
