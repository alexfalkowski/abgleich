package ai

import (
	"github.com/alexfalkowski/abgleich/internal/transport/http/client"
	"github.com/alexfalkowski/go-service/v2/bytes"
	"github.com/alexfalkowski/go-service/v2/context"
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

// NewProvider reads the configured key once without making network requests.
// A nil configuration returns nil, nil. Invalid settings or missing/empty key
// sources prevent startup; changing credentials requires a process restart.
// Idle connections close during shutdown after scheduled work stops.
func NewProvider(cfg *Config, fs *os.FS, log *logger.Logger, connections *client.Factory) (Provider, error) {
	if !cfg.IsEnabled() {
		return nil, nil
	}

	key, err := fs.ReadSource(cfg.OpenAI.APIKey)
	apiKey := strings.TrimSpace(bytes.String(key))
	if err != nil || strings.IsEmpty(apiKey) {
		return nil, errors.New("ai: api key unavailable")
	}

	// Inference POSTs never use transport retries, including when shared client
	// retry settings are present. The SDK's retries are also disabled below.
	connection, err := connections.NewClient(cfg.Client, cfg.GetTimeout())
	if err != nil {
		return nil, errors.Prefix("ai: client", err)
	}

	// Construct the Responses service directly: NewClient inherits OPENAI_* values,
	// including custom headers, while this constructor only uses explicit options.
	service := responses.NewResponseService(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(cfg.OpenAI.GetURL()),
		option.WithHTTPClient(connection),
		option.WithMaxRetries(0),
	)
	return &loggingProvider{
		Provider: &provider{service: &service},
		logger:   log,
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
