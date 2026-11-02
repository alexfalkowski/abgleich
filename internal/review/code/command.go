package code

import (
	_ "embed"

	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/abgleich/internal/commands"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/abgleich/internal/review/internal/workflow"
)

const codeReviewCommand = "codereview"

//go:embed prompt.txt
var instructions string

// NewCommand constructs the optional code review using the shared workflow.
// Omitted configuration disables work; enabled reviews require a provider.
func NewCommand(cfg *Config, service github.Service, provider ai.Provider, sources *workflow.Sources, authorizer *commands.Authorizer) (*Command, error) {
	if cfg == nil {
		return &Command{}, nil
	}

	command, err := workflow.NewCommand(workflow.Definition{
		Name:            codeReviewCommand,
		Title:           "Code review",
		Model:           cfg.Model,
		ReasoningEffort: cfg.ReasoningEffort,
		Instructions:    instructions,
	}, service, provider, sources, authorizer)
	if err != nil {
		return nil, err
	}
	return &Command{Command: command, config: cfg}, nil
}

// Command keeps code review configuration and registration feature-owned.
// The embedded workflow command supplies consent checks and an independent queue.
type Command struct {
	// Command supplies shared consent, source access, publication, and a queue dedicated to code reviews.
	*workflow.Command
	config *Config
}
