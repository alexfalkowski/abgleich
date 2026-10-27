package ai

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/telemetry/logger"
	"github.com/alexfalkowski/go-service/v2/time"
)

type loggingProvider struct {
	Provider
	logger *logger.Logger
}

func (p *loggingProvider) Generate(ctx context.Context, prompt Prompt) (*Completion, error) {
	started := time.Now()
	completion, err := p.Provider.Generate(ctx, prompt)
	fields := []any{
		"task", prompt.Task,
		"model", prompt.Model,
		"duration", time.Since(started),
	}

	if err != nil {
		fields = append(fields, "error", err)
		p.logger.Error("ai generation", fields...)

		return nil, err
	}

	if usage := completion.Usage; usage != nil {
		fields = append(fields,
			"input_tokens", usage.InputTokens,
			"cached_input_tokens", usage.CachedInputTokens,
			"cache_write_tokens", usage.CacheWriteTokens,
			"output_tokens", usage.OutputTokens)
	}

	p.logger.Info("ai generation", fields...)

	return completion, nil
}
