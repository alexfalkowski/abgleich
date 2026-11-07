package workflow

import (
	"fmt"

	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/abgleich/internal/commands"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/di"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/alexfalkowski/go-sync"
)

// FactoryParams supplies the dependencies shared by independent review commands.
type FactoryParams struct {
	// In marks this struct as an injected dependency group.
	di.In

	// GitHub provides repository access and authenticated review publication.
	GitHub github.Service

	// Provider generates reviews using each command's prompt and model.
	Provider ai.Provider

	// Sources supplies credential-free dependency and language reference access.
	Sources *Sources

	// Authorizer rechecks human command identity and current write permission.
	Authorizer *commands.Authorizer
}

// NewFactory binds shared review dependencies and creates their publication gate.
// Construction performs no remote calls; a missing provider is rejected only
// when an enabled feature requests a command.
func NewFactory(params FactoryParams) *Factory {
	return &Factory{
		github:      params.GitHub,
		provider:    params.Provider,
		sources:     params.Sources,
		authorizer:  params.Authorizer,
		publication: sync.NewWorker(1),
	}
}

// Factory constructs independent review commands and dispatches aggregate requests.
// Commands from one factory serialize final verdict checks and publication
// within the worker process. Construct all commands before serving requests.
type Factory struct {
	github      github.Service
	provider    ai.Provider
	sources     *Sources
	authorizer  *commands.Authorizer
	publication *sync.Worker
	commands    []*Command
}

// NewCommand constructs an enabled review with its own bounded queue. Invalid
// reasoning effort or a missing provider returns an error without remote calls.
// Successful construction includes the command in aggregate review requests.
func (f *Factory) NewCommand(definition Definition) (*Command, error) {
	switch definition.ReasoningEffort {
	case strings.Empty, "none", "minimal", "low", "medium", "high", "xhigh", "max":
	default:
		return nil, errors.New("review: invalid reasoning effort")
	}

	if f.provider == nil {
		return nil, errors.New(definition.Name + ": requires AI configuration")
	}

	command := &Command{
		definition:   definition,
		github:       f.github,
		provider:     f.provider,
		sources:      f.sources,
		publication:  f.publication,
		queue:        commands.NewQueue(),
		authorizer:   f.authorizer,
		staleMessage: fmt.Sprintf("🔄 The pull request revision changed. Post a new /ab %s request.", definition.Name),
	}
	f.commands = append(f.commands, command)
	return command, nil
}

// Handle queues every configured review using the original comment as consent.
// It attempts every command and preserves their errors. Acceptance requires at
// least one configured command and acceptance by all of them; accepted work is
// retained when another command declines or fails.
func (f *Factory) Handle(ctx context.Context, event *github.IssueCommentEvent) (bool, error) {
	if len(f.commands) == 0 {
		return false, nil
	}

	allAccepted := true

	var failures error
	for _, command := range f.commands {
		accepted, err := command.Handle(ctx, event)
		if !accepted || err != nil {
			allAccepted = false
		}
		if err != nil {
			failures = errors.Join(failures, fmt.Errorf("%s: %w", command.definition.Name, err))
		}
	}

	return allAccepted, failures
}
