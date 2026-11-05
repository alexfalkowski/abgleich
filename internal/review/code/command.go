package code

import (
	_ "embed"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/abgleich/internal/review/internal/workflow"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/di"
)

const codeReviewCommand = "codereview"

//go:embed prompt.txt
var instructions string

// CommandParams supplies code review configuration and shared workflow dependencies.
type CommandParams struct {
	// In marks this struct as an injected dependency group.
	di.In

	// Config enables code reviews and selects their model and schedule.
	Config *Config
	// Workflow constructs commands sharing source access and verdict publication.
	Workflow *workflow.Factory
}

// NewCommand constructs the optional code review using the shared workflow.
// Omitted configuration returns nil; enabled reviews require a provider.
func NewCommand(params CommandParams) (*Command, error) {
	if !params.Config.IsEnabled() {
		return nil, nil
	}

	command, err := params.Workflow.NewCommand(workflow.Definition{
		Name:            codeReviewCommand,
		Title:           "Code review",
		Model:           params.Config.Model,
		ReasoningEffort: params.Config.ReasoningEffort,
		Instructions:    instructions,
	})
	if err != nil {
		return nil, err
	}

	return &Command{Command: command, config: params.Config}, nil
}

// Command keeps code review configuration and registration feature-owned.
// The embedded workflow command supplies consent checks and an independent queue.
type Command struct {
	// Command supplies shared consent, source access, publication, and a queue dedicated to code reviews.
	*workflow.Command
	config *Config
}

// Handle declines disabled reviews through a nil receiver, preserving command
// reaction feedback. Enabled commands delegate authorization to the workflow.
func (c *Command) Handle(ctx context.Context, event *github.IssueCommentEvent) (bool, error) {
	if c == nil {
		return false, nil
	}

	return c.Command.Handle(ctx, event)
}
