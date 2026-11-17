package workflow

import (
	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/abgleich/internal/commands"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/abgleich/internal/review/internal/sources"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/errors"
	"github.com/alexfalkowski/go-signal/v2"
	"github.com/alexfalkowski/go-sync"
)

// FactoryParams supplies the dependencies shared by independent review commands.
type FactoryParams struct {
	// In marks this struct as an injected dependency group.
	signal.In

	// GitHub provides repository access and authenticated review publication.
	GitHub github.Service

	// Providers resolves the connection selected by each review feature.
	Providers *ai.Registry

	// Sources supplies credential-free dependency and language reference access.
	Sources *sources.Registry

	// Authorizer rechecks human command identity and current write permission.
	Authorizer *commands.Authorizer
}

// NewFactory binds shared review dependencies and creates their publication gate.
// Construction performs no remote calls; a missing provider is rejected only
// when an enabled feature requests a command.
func NewFactory(params FactoryParams) *Factory {
	return &Factory{
		github:      params.GitHub,
		providers:   params.Providers,
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
	providers   *ai.Registry
	sources     *sources.Registry
	authorizer  *commands.Authorizer
	publication *sync.Worker
	commands    []*Command
}

// NewCommand constructs an enabled review with its own bounded queue.
// A missing or unknown provider returns an error without remote calls.
// Successful construction includes the command in aggregate review requests.
func (f *Factory) NewCommand(profile ReviewProfile) (*Command, error) {
	provider, err := f.providers.Select(profile.Provider)
	if err != nil {
		return nil, errors.Prefix(profile.Name, err)
	}

	command := &Command{
		github:      f.github,
		provider:    provider,
		sources:     f.sources,
		publication: f.publication,
		queue:       commands.NewQueue(),
		authorizer:  f.authorizer,
		profile:     profile,
	}
	f.commands = append(f.commands, command)
	return command, nil
}

// Handle queues every configured review using the original comment as consent.
// Initial consent and PR eligibility are shared within this delivery; each job
// independently rechecks them before source access and publication.
// It attempts every command and preserves their errors. Acceptance requires at
// least one configured command and acceptance by all of them; accepted work is
// retained when another command declines or fails.
func (f *Factory) Handle(ctx context.Context, event *github.IssueCommentEvent) (bool, error) {
	if len(f.commands) == 0 {
		return false, nil
	}

	// Every command uses the same GitHub service, authorizer, and aggregate consent.
	// Repeating these remote reads per feature can exhaust the webhook deadline.
	task, comments, err := f.commands[0].authorize(ctx, &reviewTask{
		repository: event.GetRepo(),
		request:    &github.PullRequest{Number: event.GetIssue().Number},
		comment:    event.GetComment(),
	})
	if err != nil || task == nil {
		return false, err
	}

	allAccepted := true

	var failures error
	for _, command := range f.commands {
		accepted, err := command.enqueue(ctx, task, comments)
		if !accepted || err != nil {
			allAccepted = false
		}
		if err != nil {
			failures = errors.Join(failures, errors.Prefix(command.profile.Name, err))
		}
	}

	return allAccepted, failures
}
