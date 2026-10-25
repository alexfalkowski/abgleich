package summarise

import (
	"github.com/alexfalkowski/abgleich/internal/scheduler"
	"github.com/alexfalkowski/go-service/v2/time"
)

const (
	defaultInterval = 30 * time.Second
	defaultTimeout  = 2 * time.Minute
)

// Config selects the summary model and independent job schedule.
// Omit the block to disable summaries. Zero durations use 30s and 2m.
type Config struct {
	Model            string `yaml:"model,omitempty" json:"model,omitempty" toml:"model,omitempty" validate:"required"`
	scheduler.Config `yaml:",inline" json:",inline" toml:",inline"`
}

// GetInterval returns the summary interval, defaulting to thirty seconds when zero.
// Config must be non-nil.
func (c *Config) GetInterval() time.Duration {
	if c.Interval == 0 {
		return defaultInterval
	}

	return c.Interval
}

// GetTimeout returns the complete summary job's timeout, defaulting to two
// minutes when zero. Config must be non-nil.
func (c *Config) GetTimeout() time.Duration {
	if c.Timeout == 0 {
		return defaultTimeout
	}

	return c.Timeout
}
