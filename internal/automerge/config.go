package automerge

// Config identifies the final CI signal required before an event check may rebase.
// CompletionCheck is an exact commit-status context or check-run name.
// Omitted or empty configuration leaves rebasing to scheduled recovery sweeps.
type Config struct {
	CompletionCheck string `yaml:"completion_check,omitempty" json:"completion_check,omitempty" toml:"completion_check,omitempty"`
}

// GetCompletionCheck returns the configured final CI signal, or an empty string
// when configuration is nil or event-driven rebasing is disabled. Scheduled
// sweeps do not require this signal.
func (c *Config) GetCompletionCheck() string {
	if c == nil {
		return ""
	}

	return c.CompletionCheck
}
