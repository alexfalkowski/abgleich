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
	Model    string        `yaml:"model,omitempty" json:"model,omitempty" toml:"model,omitempty" validate:"required"`
	Interval time.Duration `yaml:"interval,omitempty" json:"interval,omitempty" toml:"interval,omitempty" validate:"omitempty,gt=0"`
	Timeout  time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty" toml:"timeout,omitempty" validate:"omitempty,gt=0"`
}

func (c *Config) schedule() *scheduler.Config {
	interval := c.Interval
	if interval == 0 {
		interval = defaultInterval
	}

	timeout := c.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	return &scheduler.Config{Interval: interval, Timeout: timeout}
}
