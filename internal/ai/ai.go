package ai

// Prompt supplies text and generation limits for one task. Instructions and Input
// are sent to the model; Task identifies the feature in logs. MaxOutputTokens
// includes reasoning tokens as well as visible output.
type Prompt struct {
	Model           string
	Instructions    string
	Input           string
	Task            string
	MaxOutputTokens int64
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
