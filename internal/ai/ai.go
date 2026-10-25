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

// Completion contains nonempty generated text from a completed response and its
// provider-reported token usage. Callers interpret the text within their task's domain.
type Completion struct {
	Text         string
	InputTokens  int64
	OutputTokens int64
}
