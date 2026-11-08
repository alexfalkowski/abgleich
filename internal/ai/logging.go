package ai

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/slices"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/alexfalkowski/go-service/v2/telemetry/logger"
	"github.com/alexfalkowski/go-service/v2/time"
)

type loggingProvider struct {
	Provider
	logger *logger.Logger
}

// Generate delegates to Provider and logs duration, outcome, and available token
// usage without recording prompt or completion text.
func (p *loggingProvider) Generate(ctx context.Context, prompt Prompt) (*Completion, error) {
	started := time.Now()
	largestTool, largestBytes := strings.Empty, 0
	prompt.Tools = slices.Clone(prompt.Tools)

	for i, tool := range prompt.Tools {
		if tool.Call == nil {
			continue
		}

		prompt.Tools[i].Call = func(ctx context.Context, input string) (string, error) {
			output, err := tool.Call(ctx, input)
			if err == nil && len(output) > largestBytes {
				largestTool, largestBytes = tool.Name, len(output)
			}

			return output, err
		}
	}

	completion, err := p.Provider.Generate(ctx, prompt)
	fields := []any{
		"task", prompt.Task,
		"model", prompt.Model,
		"duration", time.Since(started),
		"largest_tool", largestTool,
		"largest_tool_output_bytes", largestBytes,
	}

	if err != nil {
		fields = append(fields, "error", err)
		if failure, ok := errors.AsType[*GenerationError](err); ok {
			fields = append(fields, "http_status", failure.StatusCode,
				"provider_code", failure.Code, "provider_type", failure.Type, "provider_request_id", failure.RequestID)
		}
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
