package ai

import (
	"github.com/alexfalkowski/abgleich/internal/transport/http/client"
	"github.com/alexfalkowski/go-service/v3/bytes"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/errors"
	"github.com/alexfalkowski/go-service/v3/os"
	"github.com/alexfalkowski/go-service/v3/strings"
	"github.com/alexfalkowski/go-service/v3/telemetry/logger"
	"github.com/alexfalkowski/go-signal/v2"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

// Provider runs one conversation without retries or persistent response storage.
// Features own instructions, source authorization, tools, and generation budgets.
type Provider interface {
	// Generate rejects refused, partial, failed, or empty results and observes
	// cancellation, the conversation deadline, and BeforeRequest on every turn.
	Generate(ctx context.Context, prompt Prompt) (*Completion, error)
}

// RegistryParams supplies startup configuration and shared transport construction.
type RegistryParams struct {
	// In marks this struct as an injected dependency group.
	signal.In

	// Config defines optional provider connections and credential sources.
	Config *Config

	// FS resolves explicit credential sources once at startup.
	FS *os.FS

	// Logger records outcomes without prompts, responses, or credentials.
	Logger *logger.Logger

	// Connections registers connection cleanup after feature jobs stop.
	Connections *client.Factory
}

// NewRegistry constructs named provider connections without contacting providers.
// Missing or empty credentials fail startup. A nil configuration disables AI.
func NewRegistry(params RegistryParams) (*Registry, error) {
	if !params.Config.IsEnabled() {
		return nil, nil
	}

	connections := params.Config.Providers
	registry := &Registry{providers: make(map[string]Provider)}
	if len(connections) == 0 {
		return nil, errors.New("ai: requires provider connections")
	}

	for name, config := range connections {
		if strings.IsEmpty(strings.TrimSpace(name)) {
			return nil, errors.New("ai: invalid provider connection")
		}

		key, err := params.FS.ReadSource(config.APIKey)
		apiKey := strings.TrimSpace(bytes.String(key))
		if err != nil || strings.IsEmpty(apiKey) {
			return nil, errors.Prefix("ai: "+name, errors.New("api key unavailable"))
		}

		connection, err := params.Connections.NewClient(config.Client, config.GetTimeout())
		if err != nil {
			return nil, errors.Prefix("ai: "+name+" client", err)
		}

		var backend Provider
		switch config.Kind {
		case ProviderOpenAI:
			// Construct the service directly so only explicit worker configuration
			// supplies credentials, endpoints, and headers. Inference never retries.
			service := responses.NewResponseService(
				option.WithAPIKey(apiKey),
				option.WithBaseURL(config.URL.String()),
				option.WithHTTPClient(connection),
				option.WithMaxRetries(0),
			)
			backend = &provider{service: &service, name: name}
		default:
			return nil, errors.New("ai: unsupported provider kind")
		}

		registry.providers[name] = &loggingProvider{
			Provider: backend,
			logger:   params.Logger, name: name, kind: config.Kind,
		}
	}

	return registry, nil
}

// Registry resolves feature selections to startup-owned provider connections.
type Registry struct {
	providers map[string]Provider
}

// Select resolves an enabled feature's named provider connection.
func (r *Registry) Select(name string) (Provider, error) {
	if r == nil {
		return nil, errors.New("requires AI configuration")
	}
	selected, ok := r.providers[name]
	if !ok {
		return nil, errors.New("ai: unknown or missing provider connection")
	}
	return selected, nil
}

type provider struct {
	service *responses.ResponseService
	name    string
}

// Generate performs one cancellable conversation and rejects partial,
// refused, failed, empty, or unexpected output.
// Task configuration and request budgets belong to the calling feature.
func (p *provider) Generate(ctx context.Context, prompt Prompt) (*Completion, error) {
	ctx, cancel := context.WithTimeout(ctx, prompt.GetTimeout())
	defer cancel()
	params := responses.ResponseNewParams{
		Model:        prompt.Model,
		Instructions: openai.String(prompt.Instructions),
		Input:        responses.ResponseNewParamsInputUnion{OfString: openai.String(prompt.Input)},
		Store:        openai.Bool(false),
		Background:   openai.Bool(false),
		ToolChoice: responses.ResponseNewParamsToolChoiceUnion{
			OfToolChoiceMode: openai.Opt(responses.ToolChoiceOptionsNone),
		},
	}

	if prompt.MaxOutputTokens > 0 {
		params.MaxOutputTokens = openai.Int(prompt.MaxOutputTokens)
	}

	if prompt.Effort.IsDefined() {
		params.Reasoning.Effort = shared.ReasoningEffort(prompt.Effort)
	}

	completion, err := p.generate(ctx, prompt, params)
	if err != nil {
		return nil, err
	}

	if strings.IsEmpty(completion.Model) {
		completion.Model = prompt.Model
	}
	completion.Provider = p.name
	return completion, nil
}

// Missing or inconsistent usage metadata must not suppress a valid summary.
func responseUsage(response *responses.Response) *Usage {
	usage := response.Usage
	hasInputTokens := usage.JSON.InputTokens.Valid()
	hasOutputTokens := usage.JSON.OutputTokens.Valid()
	hasCachedTokens := usage.InputTokensDetails.JSON.CachedTokens.Valid()
	if !hasInputTokens || !hasOutputTokens || !hasCachedTokens {
		return nil
	}

	cached, written := usage.InputTokensDetails.CachedTokens, usage.InputTokensDetails.CacheWriteTokens
	validInput := usage.InputTokens >= 0
	validOutput := usage.OutputTokens >= 0
	validCached := cached >= 0 && cached <= usage.InputTokens
	if !validInput || !validOutput || !validCached {
		return nil
	}

	validWritten := written >= 0 && written <= usage.InputTokens-cached
	if !validWritten {
		return nil
	}
	return &Usage{
		InputTokens:       usage.InputTokens,
		CachedInputTokens: cached,
		CacheWriteTokens:  written,
		OutputTokens:      usage.OutputTokens,
	}
}
