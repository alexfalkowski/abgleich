package config

import (
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/abgleich/internal/health"
	"github.com/alexfalkowski/abgleich/internal/scheduler"
	"github.com/alexfalkowski/go-service/v2/config"
)

// Config for the service.
type Config struct {
	Scheduler      *scheduler.Config `yaml:"scheduler,omitempty" json:"scheduler,omitempty" toml:"scheduler,omitempty" validate:"omitempty"`
	GitHub         *github.Config    `yaml:"github,omitempty" json:"github,omitempty" toml:"github,omitempty" validate:"required"`
	Health         *health.Config    `yaml:"health,omitempty" json:"health,omitempty" toml:"health,omitempty" validate:"required"`
	*config.Config `yaml:",inline" json:",inline" toml:",inline" validate:"required"`
}

func decorateConfig(cfg *Config) *config.Config {
	return cfg.Config
}

func healthConfig(cfg *Config) *health.Config {
	return cfg.Health
}

func githubConfig(cfg *Config) *github.Config {
	return cfg.GitHub
}

func schedulerConfig(cfg *Config) *scheduler.Config {
	return cfg.Scheduler
}
