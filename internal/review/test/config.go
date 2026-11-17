package test

import (
	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/abgleich/internal/scheduler"
	"github.com/alexfalkowski/go-service/v3/time"
)

const (
	defaultInterval = 30 * time.Second
	defaultTimeout  = 2 * time.Minute
)

// Config selects the test review model and independent job schedule.
// Omit the block to disable test reviews. Zero durations use 30s and 2m.
type Config struct {
	// ModelConfig selects the test review provider, model, and optional effort.
	ai.ModelConfig `yaml:",inline" json:",inline" toml:",inline"`

	// Config provides independent job timing; zero values use a thirty-second interval and two-minute timeout.
	scheduler.Config `yaml:",inline" json:",inline" toml:",inline"`
}

// IsEnabled reports whether test review configuration is present.
func (c *Config) IsEnabled() bool {
	return c != nil
}

// GetInterval returns the test review interval, defaulting to thirty seconds when zero.
// Config must be non-nil.
func (c *Config) GetInterval() time.Duration {
	if c.Interval == 0 {
		return defaultInterval
	}

	return c.Interval
}

// GetTimeout returns the complete test review job's timeout, defaulting to two
// minutes when zero. Config must be non-nil.
func (c *Config) GetTimeout() time.Duration {
	if c.Timeout == 0 {
		return defaultTimeout
	}

	return c.Timeout
}
