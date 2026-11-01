package ai

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/time"
)

// Prompt supplies text and generation limits for one task. Instructions and Input
// are sent to the model; Task identifies the feature in logs. MaxOutputTokens
// includes reasoning tokens as well as visible output; zero omits the API limit.
type Prompt struct {
	Model           string
	Instructions    string
	Input           string
	Task            string
	ReasoningEffort string
	// BeforeRequest rechecks consent before each source submission, including continuations.
	BeforeRequest   func(context.Context) error
	Tools           []Tool
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
	Parameters  map[string]any
	Call        func(context.Context, string) (string, error)
	Name        string
	Description string
}

// Completion contains completed text and optional provider-reported usage.
// Missing or inconsistent usage is nil rather than an invented zero count.
type Completion struct {
	Usage *Usage
	Model string
	Text  string
}

// Usage separates provider-reported input categories. Cached and cache-write tokens are
// subsets of InputTokens; OutputTokens already includes reasoning tokens.
type Usage struct {
	InputTokens       int64
	CachedInputTokens int64
	CacheWriteTokens  int64
	OutputTokens      int64
}
