package cleanup

import (
	"github.com/alexfalkowski/abgleich/internal/scheduler"
	"github.com/alexfalkowski/go-service/v2/time"
)

const (
	defaultInactiveAfter = 30 * 24 * time.Hour
	defaultInterval      = 24 * time.Hour
)

// Config enables branch cleanup when present. InactiveAfter defaults to thirty
// days, Interval to one day, and Timeout to five minutes. Exclude contains exact,
// case-sensitive branch names to preserve in addition to protected/default branches.
type Config struct {
	Exclude          []string      `yaml:"exclude,omitempty" json:"exclude,omitempty" toml:"exclude,omitempty" validate:"dive,required"`
	InactiveAfter    time.Duration `yaml:"inactive_after,omitempty" json:"inactive_after,omitempty" toml:"inactive_after,omitempty" validate:"omitempty,gt=0"`
	scheduler.Config `yaml:",inline" json:",inline" toml:",inline"`
}

// GetInterval returns the cleanup interval, defaulting to one day when zero.
// Config must be non-nil.
func (c *Config) GetInterval() time.Duration {
	if c.Interval == 0 {
		return defaultInterval
	}

	return c.Interval
}

// GetInactiveAfter returns the latest-commit age required for inactive cleanup,
// defaulting to thirty days when zero. Config must be non-nil.
func (c *Config) GetInactiveAfter() time.Duration {
	if c.InactiveAfter == 0 {
		return defaultInactiveAfter
	}

	return c.InactiveAfter
}
