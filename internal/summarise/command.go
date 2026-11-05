package summarise

import (
	"strconv"

	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/abgleich/internal/commands"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/di"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
)

const (
	summaryCommand = "summarise"
	summaryHeading = "### 🤖 Abgleich summary\n"
)

// CommandParams supplies summary configuration and command-processing dependencies.
type CommandParams struct {
	// In marks this struct as an injected dependency group.
	di.In

	// Config enables summaries and selects their model and schedule.
	Config *Config
	// GitHub provides repository access and authenticated summary publication.
	GitHub github.Service
	// Provider generates summaries using the feature-owned prompt and model.
	Provider ai.Provider
	// Authorizer rechecks human command identity and current write permission.
	Authorizer *commands.Authorizer
}

// NewCommand constructs optional summaries using the configured provider.
// Omitted configuration returns nil; enabled summaries require a provider.
func NewCommand(params CommandParams) (*Command, error) {
	if !params.Config.IsEnabled() {
		return nil, nil
	}
	if params.Provider == nil {
		return nil, errors.New("summarise: requires AI configuration")
	}

	return &Command{
		config:     params.Config,
		github:     params.GitHub,
		provider:   params.Provider,
		queue:      commands.NewQueue(),
		authorizer: params.Authorizer,
	}, nil
}

// Command accepts bounded background work. Pending work is held in memory;
// completed App comments prevent another summary of the same pull request.
// Feature registration and lifecycle belong to the existing scheduler.
type Command struct {
	config     *Config
	github     github.Service
	provider   ai.Provider
	queue      *commands.Queue
	authorizer *commands.Authorizer
}

// Handle authorizes the current comment and queues a summary. Existing summaries
// and queued requests are acknowledged without another provider call. A nil
// receiver declines disabled summaries while preserving command reaction feedback.
func (c *Command) Handle(ctx context.Context, event *github.IssueCommentEvent) (bool, error) {
	if c == nil {
		return false, nil
	}

	task := &summaryTask{
		repository:     event.GetRepo(),
		pullRequest:    &github.PullRequest{Number: event.GetIssue().Number},
		commandComment: event.GetComment(),
	}

	current, err := c.authorize(ctx, task)
	if err != nil || current == nil {
		return false, err
	}

	if exists, err := c.hasSummary(ctx, current); err != nil || exists {
		return exists, err
	}

	if err := ctx.Err(); err != nil {
		return false, err
	}

	key := strings.ToLower(current.repository.GetFullName()) + "#" + strconv.Itoa(current.pullRequest.GetNumber())
	err = c.queue.Enqueue(key, func(ctx context.Context) error {
		return c.process(ctx, current)
	})

	return err == nil, err
}

func (c *Command) authorize(ctx context.Context, task *summaryTask) (*summaryTask, error) {
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

	comment, err := c.authorizer.Authorize(ctx, summaryCommand, repo, pull.GetNumber(), task.commandComment)
	if err != nil || comment == nil {
		return nil, err
	}

	return &summaryTask{
		repository:     repo,
		pullRequest:    pull,
		commandComment: comment,
	}, nil
}

func (c *Command) eligible(repo *github.Repository, pull *github.PullRequest) bool {
	return pull.GetState() == "open" &&
		!pull.GetDraft() &&
		c.github.IsBaseRepository(repo, pull) &&
		github.ValidatePullRequestMetadata(pull) == nil
}

func (c *Command) hasSummary(ctx context.Context, task *summaryTask) (bool, error) {
	comments, err := c.github.ListComments(ctx, task.repository, task.pullRequest.GetNumber())
	if err != nil {
		return false, err
	}

	for _, comment := range comments {
		if c.github.IsAppComment(comment) && strings.Contains(comment.GetBody(), summaryHeading) {
			return true, nil
		}
	}

	return false, nil
}

func (c *Command) revisionChanged(expected, current *summaryTask) bool {
	return !c.github.IsSamePullRequest(current.pullRequest, expected.pullRequest) ||
		current.pullRequest.GetBase().GetSHA() != expected.pullRequest.GetBase().GetSHA()
}
