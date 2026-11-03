package workflow

import (
	"crypto/sha256"
	"fmt"
	"strconv"

	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/abgleich/internal/commands"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
)

// Definition supplies the identity, model, and instructions for
// one advisory review command. Feature packages own these values and registration.
// Name is the exact /ab command suffix and the persisted result namespace.
type Definition struct {
	// Name is the exact /ab command suffix and the successful result namespace.
	Name string
	// Title labels the review in published replies and failure feedback.
	Title string
	// Model selects the provider model for this feature's review conversations.
	Model string
	// Instructions supplies the feature-owned advisory review prompt.
	Instructions string
	// ReasoningEffort optionally selects provider reasoning effort; empty leaves its default.
	ReasoningEffort string
}

type reviewTask struct {
	repository     *github.Repository
	pullRequest    *github.PullRequest
	commandComment *github.IssueComment
}

// NewCommand constructs an enabled advisory review with its own bounded queue.
// A missing provider returns an error; construction performs no remote calls.
func NewCommand(definition Definition, service github.Service, provider ai.Provider, sources *Sources, authorizer *commands.Authorizer) (*Command, error) {
	switch definition.ReasoningEffort {
	case strings.Empty, "none", "minimal", "low", "medium", "high", "xhigh", "max":
	default:
		return nil, errors.New("review: invalid reasoning effort")
	}

	if provider == nil {
		return nil, errors.New(definition.Name + ": requires AI configuration")
	}

	return &Command{
		definition:     definition,
		github:         service,
		provider:       provider,
		sources:        sources,
		queue:          commands.NewQueue(),
		authorizer:     authorizer,
		staleMessage:   fmt.Sprintf("The pull request revision changed. Post a new /ab %s request.", definition.Name),
		failureMessage: fmt.Sprintf("%s failed. Post a new /ab %s request after recovery.", definition.Title, definition.Name),
	}, nil
}

// Command owns shared review consent, revision checks, deduplication, and queued
// work. Each instance has independent state; successful results retain its name.
type Command struct {
	definition     Definition
	github         github.Service
	provider       ai.Provider
	sources        *Sources
	queue          *commands.Queue
	authorizer     *commands.Authorizer
	staleMessage   string
	failureMessage string
}

// Handle refreshes human consent and queues one revision-specific review. A full
// queue preserves existing command backpressure; inference never holds the slot.
// A nil Command represents a disabled feature and declines the request.
func (c *Command) Handle(ctx context.Context, event *github.IssueCommentEvent) (bool, error) {
	if c == nil {
		return false, nil
	}

	task, comments, err := c.authorize(ctx, &reviewTask{
		repository:     event.GetRepo(),
		pullRequest:    &github.PullRequest{Number: event.GetIssue().Number},
		commandComment: event.GetComment(),
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

func (c *Command) authorize(ctx context.Context, task *reviewTask) (*reviewTask, []*github.IssueComment, error) {
	if err := c.github.ValidateInstallation(ctx); err != nil {
		return nil, nil, err
	}

	repo, err := c.github.GetRepository(ctx, task.repository)
	if err != nil {
		return nil, nil, err
	}

	pull, err := c.github.GetPullRequest(ctx, repo, task.pullRequest.GetNumber())
	if err != nil {
		return nil, nil, err
	}

	if pull.GetNumber() != task.pullRequest.GetNumber() || !c.eligible(repo, pull) {
		return nil, nil, nil
	}

	comment, comments, err := c.authorizer.AuthorizeWithHistory(ctx, c.definition.Name, repo, pull.GetNumber(), task.commandComment)
	if err != nil || comment == nil {
		return nil, nil, err
	}

	return &reviewTask{
		repository:     repo,
		pullRequest:    pull,
		commandComment: comment,
	}, comments, nil
}

func (c *Command) eligible(repo *github.Repository, pull *github.PullRequest) bool {
	return pull.GetState() == "open" &&
		!pull.GetDraft() &&
		c.github.IsBaseRepository(repo, pull) &&
		!strings.IsEmpty(pull.GetBase().GetSHA()) &&
		github.ValidatePullRequestMetadata(pull) == nil
}

func (c *Command) revisionChanged(expected, current *reviewTask) bool {
	return !c.github.IsSamePullRequest(current.pullRequest, expected.pullRequest) ||
		current.pullRequest.GetBase().GetSHA() != expected.pullRequest.GetBase().GetSHA()
}

func (c *Command) hasReview(ctx context.Context, task *reviewTask, comments []*github.IssueComment) (bool, error) {
	marker := c.resultMarker(task)
	for _, comment := range comments {
		if c.github.IsAppComment(comment) && strings.Contains(comment.GetBody(), "\n"+marker) {
			return true, nil
		}
	}

	return c.hasNativeReview(ctx, task, marker)
}

func (c *Command) hasNativeReview(ctx context.Context, task *reviewTask, marker string) (bool, error) {
	reviews, err := c.github.ListReviews(ctx, task.repository, task.pullRequest)
	if err != nil {
		return false, err
	}

	var candidates []*github.PullRequestReview
	for _, review := range reviews {
		if review.GetState() == "COMMENTED" && strings.Contains(review.GetBody(), "\n"+marker) {
			candidates = append(candidates, review)
		}
	}

	return c.github.HasAppReview(ctx, candidates)
}

func (c *Command) resultMarker(task *reviewTask) string {
	identity := strings.Join("\n",
		c.definition.Name,
		strings.ToLower(task.repository.GetFullName()),
		strconv.Itoa(task.pullRequest.GetNumber()),
		task.pullRequest.GetNodeID(),
		task.pullRequest.GetHead().GetSHA(),
		task.pullRequest.GetBase().GetSHA(),
	)
	return fmt.Sprintf("<!-- abgleich:%s:v2:%x -->", c.definition.Name, sha256.Sum256(strings.Bytes(identity)))
}
