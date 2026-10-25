package scheduler

import "github.com/alexfalkowski/go-service/v2/time"

const (
	defaultInterval = 15 * time.Minute
	defaultTimeout  = 5 * time.Minute
)

// Schedule supplies a job's effective, positive interval and timeout.
// Feature configurations own their defaults through these getters; Config
// supplies the scheduler's shared defaults.
type Schedule interface {
	GetInterval() time.Duration
	GetTimeout() time.Duration
}

// Config controls a job's schedule. Zero values use a fifteen-minute interval
// and a five-minute timeout.
// Timeout cancels a run; a handler must observe its context before another run
// can begin. Scheduled starts missed during a run are skipped rather than queued.
type Config struct {
	Interval time.Duration `yaml:"interval,omitempty" json:"interval,omitempty" toml:"interval,omitempty" validate:"omitempty,gt=0"`
	Timeout  time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty" toml:"timeout,omitempty" validate:"omitempty,gt=0"`
}

// GetInterval returns the schedule interval, defaulting to fifteen minutes when
// configuration is absent or the interval is zero.
func (c *Config) GetInterval() time.Duration {
	if c == nil || c.Interval == 0 {
		return defaultInterval
	}

	return c.Interval
}

// GetTimeout returns the run timeout, defaulting to five minutes when
// configuration is absent or the timeout is zero.
func (c *Config) GetTimeout() time.Duration {
	if c == nil || c.Timeout == 0 {
		return defaultTimeout
	}

	return c.Timeout
}
