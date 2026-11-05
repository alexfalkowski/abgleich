package ai

import (
	"github.com/alexfalkowski/abgleich/internal/transport/http/client"
	"github.com/alexfalkowski/go-service/v2/bytes"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/di"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/os"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/alexfalkowski/go-service/v2/telemetry/logger"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

// Provider generates text using configuration resolved once at startup. Features
// own source authorization, instructions, and generation budgets.
type Provider interface {
	// Generate runs one conversation with optional caller-owned tools, without
	// response storage or automatic retries. It observes cancellation and the prompt's
	// conversation deadline (one minute when omitted). Refused, failed,
	// incomplete, and empty responses return a nil completion and an error.
	Generate(ctx context.Context, prompt Prompt) (*Completion, error)
}

// ProviderParams supplies provider settings, credential resolution, and transport dependencies.
type ProviderParams struct {
	// In marks this struct as an injected dependency group.
	di.In

	// Config enables the provider and selects credentials and HTTP settings.
	Config *Config
	// FS resolves the configured credential source once at startup.
	FS *os.FS
	// Logger records generation outcomes without source, response text, or credentials.
	Logger *logger.Logger
	// Connections creates the AI connection pool and registers its shutdown cleanup.
	Connections *client.Factory
}

// NewProvider reads the configured key once without making network requests.
// A nil configuration returns nil, nil. Invalid settings or missing/empty key
// sources prevent startup; changing credentials requires a process restart.
// Idle connections close during shutdown after scheduled work stops.
func NewProvider(params ProviderParams) (Provider, error) {
	if !params.Config.IsEnabled() {
		return nil, nil
	}

	key, err := params.FS.ReadSource(params.Config.OpenAI.APIKey)
	apiKey := strings.TrimSpace(bytes.String(key))
	if err != nil || strings.IsEmpty(apiKey) {
		return nil, errors.New("ai: api key unavailable")
	}

	// Inference POSTs never use transport retries, including when shared client
	// retry settings are present. The SDK's retries are also disabled below.
	connection, err := params.Connections.NewClient(params.Config.Client, params.Config.GetTimeout())
	if err != nil {
		return nil, errors.Prefix("ai: client", err)
	}

	// Construct the Responses service directly: NewClient inherits OPENAI_* values,
	// including custom headers, while this constructor only uses explicit options.
	service := responses.NewResponseService(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(params.Config.OpenAI.GetURL()),
		option.WithHTTPClient(connection),
		option.WithMaxRetries(0),
	)
	return &loggingProvider{
		Provider: &provider{service: &service},
		logger:   params.Logger,
	}, nil
}

type provider struct {
	service *responses.ResponseService
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

	if !strings.IsEmpty(prompt.ReasoningEffort) {
		params.Reasoning.Effort = shared.ReasoningEffort(prompt.ReasoningEffort)
	}

	completion, err := p.generate(ctx, prompt, params)
	if err != nil {
		return nil, err
	}

	if strings.IsEmpty(completion.Model) {
		completion.Model = prompt.Model
	}
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
