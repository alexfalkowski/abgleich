package workflow

import (
	"fmt"

	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/abgleich/internal/commands"
	"github.com/alexfalkowski/abgleich/internal/github"
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

// Factory constructs independent review commands with shared dependencies.
// Commands from one factory serialize final verdict checks and publication
// within the worker process.
type Factory struct {
	github      github.Service
	provider    ai.Provider
	sources     *Sources
	authorizer  *commands.Authorizer
	publication *sync.Worker
}

// NewCommand constructs an enabled review with its own bounded queue. Invalid
// reasoning effort or a missing provider returns an error without remote calls.
func (f *Factory) NewCommand(definition Definition) (*Command, error) {
	switch definition.ReasoningEffort {
	case strings.Empty, "none", "minimal", "low", "medium", "high", "xhigh", "max":
	default:
		return nil, errors.New("review: invalid reasoning effort")
	}

	if f.provider == nil {
		return nil, errors.New(definition.Name + ": requires AI configuration")
	}

	return &Command{
		definition:   definition,
		github:       f.github,
		provider:     f.provider,
		sources:      f.sources,
		publication:  f.publication,
		queue:        commands.NewQueue(),
		authorizer:   f.authorizer,
		staleMessage: fmt.Sprintf("🔄 The pull request revision changed. Post a new /ab %s request.", definition.Name),
	}, nil
}
