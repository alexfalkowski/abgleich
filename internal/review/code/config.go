package code

import (
	"github.com/alexfalkowski/abgleich/internal/scheduler"
	"github.com/alexfalkowski/go-service/v2/time"
)

const (
	defaultInterval = 30 * time.Second
	defaultTimeout  = 2 * time.Minute
)

// Config selects the code review model and independent job schedule.
// Omit the block to disable code reviews. Zero durations use 30s and 2m.
type Config struct {
	// ReasoningEffort optionally selects provider reasoning effort; empty leaves its default.
	ReasoningEffort string `yaml:"reasoning_effort,omitempty" json:"reasoning_effort,omitempty" toml:"reasoning_effort,omitempty"`

	// Model selects the required provider model for code reviews.
	Model string `yaml:"model,omitempty" json:"model,omitempty" toml:"model,omitempty" validate:"required"`

	// Config provides independent job timing; zero values use a thirty-second interval and two-minute timeout.
	scheduler.Config `yaml:",inline" json:",inline" toml:",inline"`
}

// IsEnabled reports whether code review configuration is present.
func (c *Config) IsEnabled() bool {
	return c != nil
}

// GetInterval returns the code review interval, defaulting to thirty seconds when zero.
// Config must be non-nil.
func (c *Config) GetInterval() time.Duration {
	if c.Interval == 0 {
		return defaultInterval
	}
	return c.Interval
}

// GetTimeout returns the complete code review job's timeout, defaulting to two
// minutes when zero. Config must be non-nil.
func (c *Config) GetTimeout() time.Duration {
	if c.Timeout == 0 {
		return defaultTimeout
	}
	return c.Timeout
}
