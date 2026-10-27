package ai

import (
	"github.com/alexfalkowski/abgleich/internal/transport/http/client"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/os"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/alexfalkowski/go-service/v2/telemetry/logger"
	"github.com/alexfalkowski/go-service/v2/time"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

// Provider generates text using configuration resolved once at startup. Features
// own source authorization, instructions, and generation budgets.
type Provider interface {
	// Generate submits one prompt without tools, response storage, or automatic
	// retries. It observes cancellation and a one-minute limit. Refused, failed,
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
	if err != nil || strings.IsEmpty(strings.TrimSpace(string(key))) {
		return nil, errors.New("ai: api key unavailable")
	}

	// Inference POSTs never use transport retries, including when shared client
	// retry settings are present. The SDK's retries are also disabled below.
	connection, err := connections.NewClient(cfg.Client, cfg.GetTimeout())
	if err != nil {
		return nil, err
	}

	// Construct the Responses service directly: NewClient inherits OPENAI_* values,
	// including custom headers, while this constructor only uses explicit options.
	service := responses.NewResponseService(
		option.WithAPIKey(strings.TrimSpace(string(key))),
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

// Generate performs one cancellable provider request and rejects partial,
// refused, failed, empty, or unexpected output.
// Task configuration and request budgets belong to the calling feature.
func (p *provider) Generate(ctx context.Context, prompt Prompt) (*Completion, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	response, err := p.service.New(ctx, responses.ResponseNewParams{
		Model:           prompt.Model,
		Instructions:    openai.String(prompt.Instructions),
		Input:           responses.ResponseNewParamsInputUnion{OfString: openai.String(prompt.Input)},
		MaxOutputTokens: openai.Int(prompt.MaxOutputTokens),
		Store:           openai.Bool(false),
		Background:      openai.Bool(false),
		ToolChoice: responses.ResponseNewParamsToolChoiceUnion{
			OfToolChoiceMode: openai.Opt(responses.ToolChoiceOptionsNone),
		},
	})

	if ctx.Err() != nil {
		return nil, errors.New("ai: cancelled or timed out")
	}

	if err != nil {
		return nil, errors.New("ai: request failed")
	}

	completion, err := completedText(response)
	if err != nil {
		return nil, err
	}

	if strings.IsEmpty(completion.Model) {
		completion.Model = prompt.Model
	}

	return completion, nil
}

func completedText(response *responses.Response) (*Completion, error) {
	if response == nil {
		return nil, errors.New("ai: empty response")
	}

	if !strings.IsEmpty(string(response.Error.Code)) || !strings.IsEmpty(response.Error.Message) {
		return nil, errors.New("ai: generation failed")
	}

	if response.Status != responses.ResponseStatusCompleted {
		return nil, errors.New("ai: incomplete response")
	}

	text := response.OutputText()
	if strings.IsEmpty(strings.TrimSpace(text)) {
		return nil, errors.New("ai: empty response")
	}

	return &Completion{
		Text:  text,
		Model: response.Model,
		Usage: responseUsage(response),
	}, nil
}

// Missing or inconsistent usage metadata must not suppress a valid summary.
func responseUsage(response *responses.Response) *Usage {
	usage := response.Usage

	if !usage.JSON.InputTokens.Valid() || !usage.JSON.OutputTokens.Valid() ||
		!usage.InputTokensDetails.JSON.CachedTokens.Valid() {
		return nil
	}

	cached, written := usage.InputTokensDetails.CachedTokens, usage.InputTokensDetails.CacheWriteTokens

	if usage.InputTokens < 0 || usage.OutputTokens < 0 || cached < 0 || cached > usage.InputTokens ||
		written < 0 || written > usage.InputTokens-cached {
		return nil
	}

	return &Usage{
		InputTokens:       usage.InputTokens,
		CachedInputTokens: cached,
		CacheWriteTokens:  written,
		OutputTokens:      usage.OutputTokens,
	}
}
