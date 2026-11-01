package ai

import (
	"math"

	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/encoding/json"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

// conversation owns only provider protocol state; features own tool authority.
type conversation struct {
	params        responses.ResponseNewParams
	provider      *provider
	tools         map[string]Tool
	seen          map[string]bool
	usage         *Usage
	input         responses.ResponseInputParam
	prompt        Prompt
	usageComplete bool
}

func (p *provider) generate(ctx context.Context, prompt Prompt, params responses.ResponseNewParams) (*Completion, error) {
	c := &conversation{
		provider: p, prompt: prompt, params: params, usageComplete: true,
		tools: make(map[string]Tool), seen: make(map[string]bool),
		input: responses.ResponseInputParam{responses.ResponseInputItemParamOfMessage(prompt.Input, responses.EasyInputMessageRoleUser)},
	}

	if err := c.configureTools(); err != nil {
		return nil, err
	}

	for {
		response, err := c.request(ctx)
		if err != nil {
			return nil, err
		}

		requested, err := c.accept(response)
		if err != nil {
			return nil, err
		}

		if len(requested) == 0 {
			if strings.IsEmpty(strings.TrimSpace(response.OutputText())) {
				return nil, errors.New("ai: empty response")
			}

			return &Completion{
				Text: response.OutputText(), Model: response.Model,
				Usage: c.usage,
			}, nil
		}

		if err := c.call(ctx, requested); err != nil {
			return nil, err
		}
	}
}

func (c *conversation) configureTools() error {
	for _, tool := range c.prompt.Tools {
		if _, exists := c.tools[tool.Name]; exists || strings.IsEmpty(tool.Name) || tool.Call == nil {
			return errors.New("ai: invalid tool definition")
		}

		c.tools[tool.Name] = tool
		definition := responses.ToolParamOfFunction(tool.Name, tool.Parameters, true)
		definition.OfFunction.Description = openai.String(tool.Description)
		c.params.Tools = append(c.params.Tools, definition)
	}

	if len(c.tools) > 0 {
		c.params.ToolChoice = responses.ResponseNewParamsToolChoiceUnion{OfToolChoiceMode: openai.Opt(responses.ToolChoiceOptionsAuto)}
		c.params.Include = []responses.ResponseIncludable{responses.ResponseIncludableReasoningEncryptedContent}
	}
	return nil
}

func (c *conversation) request(ctx context.Context) (*responses.Response, error) {
	if c.prompt.BeforeRequest != nil {
		if err := c.prompt.BeforeRequest(ctx); err != nil {
			return nil, err
		}
	}

	if len(c.tools) > 0 {
		c.params.Input = responses.ResponseNewParamsInputUnion{OfInputItemList: c.input}
	}

	response, err := c.provider.service.New(ctx, c.params)
	if ctx.Err() != nil {
		return nil, errors.New("ai: cancelled or timed out")
	}

	if err != nil {
		return nil, errors.New("ai: request failed")
	}

	if err := validateResponse(response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *conversation) accept(response *responses.Response) ([]responses.ResponseFunctionToolCall, error) {
	c.recordUsage(responseUsage(response))

	var requested []responses.ResponseFunctionToolCall
	for _, output := range response.Output {
		switch output.Type {
		case "function_call":
			call := output.AsFunctionCall()
			if err := c.validateCall(call); err != nil {
				return nil, err
			}

			requested = append(requested, call)
		case "message":
			for _, content := range output.AsMessage().Content {
				if content.Type != "output_text" {
					return nil, errors.New("ai: refused or unexpected response")
				}
			}
		case "reasoning":
		default:
			return nil, errors.New("ai: refused or unexpected response")
		}

		if err := c.remember(output); err != nil {
			return nil, err
		}
	}
	return requested, nil
}

// remember preserves reasoning, encrypted content, and tool-call identities
// across stateless requests using the SDK's response-to-input conversion.
func (c *conversation) remember(output responses.ResponseOutputItemUnion) error {
	if len(c.tools) == 0 {
		return nil
	}

	var item responses.ResponseInputItemUnion
	if err := json.Unmarshal([]byte(output.RawJSON()), &item); err != nil {
		return errors.New("ai: invalid conversation item")
	}

	c.input = append(c.input, item.ToParam())
	return nil
}

func (c *conversation) validateCall(call responses.ResponseFunctionToolCall) error {
	_, registered := c.tools[call.Name]
	missingCallID := strings.IsEmpty(call.CallID)
	hasNamespace := !strings.IsEmpty(call.Namespace)
	duplicate := c.seen[call.CallID]
	if !registered || missingCallID || call.Async || hasNamespace || duplicate {
		return errors.New("ai: unexpected or duplicate tool call")
	}

	c.seen[call.CallID] = true
	return nil
}

func (c *conversation) call(ctx context.Context, requested []responses.ResponseFunctionToolCall) error {
	for _, call := range requested {
		if err := ctx.Err(); err != nil {
			return err
		}

		result, err := c.tools[call.Name].Call(ctx, call.Arguments)
		if err != nil {
			return err
		}

		item := responses.ResponseInputItemParamOfFunctionCallOutput(result)
		item.OfFunctionCallOutput.CallID = openai.String(call.CallID)
		c.input = append(c.input, item)
	}
	return nil
}

func (c *conversation) recordUsage(current *Usage) {
	switch {
	case current == nil:
		c.usage, c.usageComplete = nil, false
	case !c.usageComplete:
		return
	case c.usage == nil:
		c.usage = current
	default:
		if c.usage.InputTokens > math.MaxInt64-current.InputTokens || c.usage.OutputTokens > math.MaxInt64-current.OutputTokens {
			c.usage, c.usageComplete = nil, false
			return
		}

		c.usage.InputTokens += current.InputTokens
		c.usage.CachedInputTokens += current.CachedInputTokens
		c.usage.CacheWriteTokens += current.CacheWriteTokens
		c.usage.OutputTokens += current.OutputTokens
	}
}

func validateResponse(response *responses.Response) error {
	if response == nil || response.Status != responses.ResponseStatusCompleted ||
		!strings.IsEmpty(string(response.Error.Code)) || !strings.IsEmpty(response.Error.Message) {
		return errors.New("ai: incomplete or failed response")
	}
	return nil
}
