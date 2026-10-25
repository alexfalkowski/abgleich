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
	if err != nil {
		p.logger.Error("ai generation",
			"task", prompt.Task,
			"model", prompt.Model,
			"duration", time.Since(started),
			"error", err)

		return nil, err
	}

	p.logger.Info("ai generation",
		"task", prompt.Task,
		"model", prompt.Model,
		"duration", time.Since(started),
		"input_tokens", completion.InputTokens,
		"output_tokens", completion.OutputTokens)

	return completion, nil
}
