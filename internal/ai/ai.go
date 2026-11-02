package ai

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/time"
)

// Prompt supplies text and generation limits for one task. Instructions and Input
// are sent to the model; Task identifies the feature in logs. MaxOutputTokens
// includes reasoning tokens as well as visible output; zero omits the API limit.
type Prompt struct {
	// Model selects the provider model for this conversation.
	Model string
	// Instructions supplies task instructions separately from untrusted source input.
	Instructions string
	// Input contains the task's initial user message, including any source context.
	Input string
	// Task identifies the owning feature in generation outcome logs.
	Task string
	// ReasoningEffort optionally selects the provider's reasoning effort; empty leaves its default.
	ReasoningEffort string
	// BeforeRequest rechecks consent before each source submission, including continuations.
	BeforeRequest func(context.Context) error
	// Tools lists the only local capabilities the model may request; empty disables tool use.
	Tools []Tool
	// MaxOutputTokens bounds reasoning and visible output tokens; zero omits the provider limit.
	MaxOutputTokens int64
	// Timeout bounds the whole conversation; GetTimeout supplies its default.
	Timeout time.Duration
}

// GetTimeout returns the conversation timeout, defaulting to one minute when zero.
func (p *Prompt) GetTimeout() time.Duration {
	if p.Timeout == 0 {
		return time.Minute
	}
	return p.Timeout
}

// Tool exposes a caller-owned capability. Generate accepts only these exact
// names and invokes Call locally; no credentials are sent to the model.
type Tool struct {
	// Parameters defines the JSON schema accepted by the tool's strict function interface.
	Parameters map[string]any
	// Call executes the local capability with JSON arguments and returns text for the model.
	Call func(context.Context, string) (string, error)
	// Name is the exact, unique function name accepted from model tool calls.
	Name string
	// Description explains the capability and its input constraints to the model.
	Description string
}

// Completion contains generated text and optional provider-reported usage.
// Missing or inconsistent usage is nil rather than an invented zero count.
type Completion struct {
	// Usage contains actual aggregate token counts, or nil when any usage is unavailable or inconsistent.
	Usage *Usage
	// Model identifies the response model, falling back to the requested model when absent.
	Model string
	// Text contains the final nonempty model output for feature-owned parsing and rendering.
	Text string
}

// Usage separates provider-reported input categories. Cached and cache-write tokens are
// subsets of InputTokens; OutputTokens already includes reasoning tokens.
type Usage struct {
	// InputTokens counts all input tokens, including cache reads and writes.
	InputTokens int64
	// CachedInputTokens counts the subset of input tokens served from the provider's cache.
	CachedInputTokens int64
	// CacheWriteTokens counts the subset of input tokens written to the provider's cache.
	CacheWriteTokens int64
	// OutputTokens counts generated tokens, including reasoning tokens.
	OutputTokens int64
}
