package automerge

// Config controls Dependabot enrollment and the final CI signal for event rebasing.
// CompletionCheck is an exact commit-status context or check-run name; an empty
// value leaves rebasing to scheduled recovery. Omitted Dependabot settings
// preserve automatic enrollment in all opted-in repositories.
type Config struct {
	// Dependabot optionally excludes repositories from automatic author-based merging.
	Dependabot *DependabotConfig `yaml:"dependabot,omitempty" json:"dependabot,omitempty" toml:"dependabot,omitempty" validate:"omitempty"`

	// CompletionCheck names the exact final CI signal for event rebasing; empty leaves rebasing to scheduled sweeps.
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

// GetDependabotExclusions returns the excluded repository names, or nil when
// the automerge or Dependabot configuration is omitted.
func (c *Config) GetDependabotExclusions() []string {
	if c == nil || c.Dependabot == nil {
		return nil
	}

	return c.Dependabot.Exclude
}

// DependabotConfig excludes repositories from author-based merge automation.
// Entries are case-insensitive owner/repository full names. Nil configuration
// or an empty list preserves automatic enrollment in every opted-in repository.
type DependabotConfig struct {
	// Exclude lists case-insensitive owner/repository names requiring explicit merge opt-in.
	Exclude []string `yaml:"exclude_repositories,omitempty" json:"exclude_repositories,omitempty" toml:"exclude_repositories,omitempty" validate:"dive,required"`
}
