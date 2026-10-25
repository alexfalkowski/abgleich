package config

import (
	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/abgleich/internal/health"
	"github.com/alexfalkowski/abgleich/internal/scheduler"
	"github.com/alexfalkowski/abgleich/internal/summarise"
	"github.com/alexfalkowski/go-service/v2/config"
)

// Config for the service.
type Config struct {
	AI             *ai.Config        `yaml:"ai,omitempty" json:"ai,omitempty" toml:"ai,omitempty" validate:"omitempty"`
	Summarise      *summarise.Config `yaml:"summarise,omitempty" json:"summarise,omitempty" toml:"summarise,omitempty" validate:"omitempty"`
	Scheduler      *scheduler.Config `yaml:"scheduler,omitempty" json:"scheduler,omitempty" toml:"scheduler,omitempty" validate:"omitempty"`
	GitHub         *github.Config    `yaml:"github,omitempty" json:"github,omitempty" toml:"github,omitempty" validate:"required"`
	Health         *health.Config    `yaml:"health,omitempty" json:"health,omitempty" toml:"health,omitempty" validate:"required"`
	*config.Config `yaml:",inline" json:",inline" toml:",inline" validate:"required"`
}

func aiConfig(cfg *Config) *ai.Config {
	return cfg.AI
}

func summariseConfig(cfg *Config) *summarise.Config {
	return cfg.Summarise
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
