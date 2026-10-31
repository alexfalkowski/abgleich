package codereview

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

const codeReviewCommand = "codereview"

type reviewTask struct {
	repository     *github.Repository
	pullRequest    *github.PullRequest
	commandComment *github.IssueComment
}

// NewCommand constructs the optional task. Enabled reviews require a provider;
// construction never contacts GitHub or the model.
func NewCommand(cfg *Config, service github.Service, provider ai.Provider) (*Command, error) {
	if cfg != nil && provider == nil {
		return nil, errors.New("codereview: requires AI configuration")
	}

	return &Command{
		config:   cfg,
		github:   service,
		provider: provider,
		queue:    commands.NewQueue(),
	}, nil
}

// Command owns advisory code review consent and background work. Pending work is
// in memory; successful App comments deduplicate the same PR head and base.
type Command struct {
	config   *Config
	github   github.Service
	provider ai.Provider
	queue    *commands.Queue
}

// Handle refreshes human consent and queues one revision-specific review. A full
// queue preserves existing command backpressure; inference never holds the slot.
func (c *Command) Handle(ctx context.Context, event *github.IssueCommentEvent) (bool, error) {
	if c.config == nil {
		return false, nil
	}

	task, err := c.authorize(ctx, &reviewTask{
		repository:     event.GetRepo(),
		pullRequest:    &github.PullRequest{Number: event.GetIssue().Number},
		commandComment: event.GetComment(),
	})
	if err != nil || task == nil {
		return false, err
	}

	if exists, err := c.hasReview(ctx, task); err != nil || exists {
		return exists, err
	}

	if err := ctx.Err(); err != nil {
		return false, err
	}

	if err := c.queue.Enqueue(resultMarker(task), func(ctx context.Context) error {
		return c.process(ctx, task)
	}); err != nil {
		return false, err
	}

	return true, nil
}

func (c *Command) authorize(ctx context.Context, task *reviewTask) (*reviewTask, error) {
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

	if pull.GetNumber() != task.pullRequest.GetNumber() || !c.eligible(repo, pull) {
		return nil, nil
	}

	comment, err := c.github.GetComment(ctx, repo, pull.GetNumber(), task.commandComment)
	if err != nil {
		return nil, err
	}
	if !validComment(comment) {
		return nil, nil
	}

	allowed, err := c.github.HasWritePermission(ctx, repo, comment.GetUser())
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, commands.ErrWriteAccessRequired
	}

	return &reviewTask{
		repository:     repo,
		pullRequest:    pull,
		commandComment: comment,
	}, nil
}

func (c *Command) eligible(repo *github.Repository, pull *github.PullRequest) bool {
	return pull.GetState() == "open" &&
		!pull.GetDraft() &&
		c.github.IsBaseRepository(repo, pull) &&
		!strings.IsEmpty(pull.GetBase().GetSHA()) &&
		github.ValidatePullRequestMetadata(pull) == nil
}

func validComment(comment *github.IssueComment) bool {
	return commands.Parse(comment.GetBody()) == codeReviewCommand &&
		comment.GetUser().GetType() == "User" &&
		github.HasCommentMetadata(comment) &&
		!comment.GetUpdatedAt().After(comment.GetCreatedAt().Time)
}

func (c *Command) revisionChanged(expected, current *reviewTask) bool {
	return !c.github.IsSamePullRequest(current.pullRequest, expected.pullRequest) ||
		current.pullRequest.GetBase().GetSHA() != expected.pullRequest.GetBase().GetSHA()
}

func resultMarker(task *reviewTask) string {
	identity := strings.Join("\n",
		codeReviewCommand,
		strings.ToLower(task.repository.GetFullName()),
		strconv.Itoa(task.pullRequest.GetNumber()),
		task.pullRequest.GetNodeID(),
		task.pullRequest.GetHead().GetSHA(),
		task.pullRequest.GetBase().GetSHA(),
	)

	return fmt.Sprintf("<!-- abgleich:codereview:v1:%x -->", sha256.Sum256([]byte(identity)))
}

func (c *Command) hasReview(ctx context.Context, task *reviewTask) (bool, error) {
	comments, err := c.github.ListComments(ctx, task.repository, task.pullRequest.GetNumber())
	if err != nil {
		return false, err
	}

	marker := resultMarker(task)
	for _, comment := range comments {
		if c.github.IsAppComment(comment) && strings.Contains(comment.GetBody(), "\n"+marker) {
			return true, nil
		}
	}

	return false, nil
}
