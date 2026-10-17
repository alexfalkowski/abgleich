package health

import "github.com/alexfalkowski/go-service/v2/time"

// Config controls how often process health is sampled and how long the shared
// external connectivity check may take. Both durations must be positive. Job
// execution deadlines are configured separately in each job's schedule.
type Config struct {
	Duration time.Duration `yaml:"duration,omitempty" json:"duration,omitempty" toml:"duration,omitempty" validate:"gt=0"`
	Timeout  time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty" toml:"timeout,omitempty" validate:"gt=0"`
}
