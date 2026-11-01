package config

import (
	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/abgleich/internal/automerge"
	"github.com/alexfalkowski/abgleich/internal/cleanup"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/abgleich/internal/health"
	"github.com/alexfalkowski/abgleich/internal/review"
	"github.com/alexfalkowski/abgleich/internal/scheduler"
	"github.com/alexfalkowski/abgleich/internal/summarise"
	"github.com/alexfalkowski/go-service/v2/config"
)

// Config for the service.
type Config struct {
	GitHub         *github.Config    `yaml:"github,omitempty" json:"github,omitempty" toml:"github,omitempty" validate:"required"`
	AI             *ai.Config        `yaml:"ai,omitempty" json:"ai,omitempty" toml:"ai,omitempty" validate:"omitempty"`
	Automerge      *automerge.Config `yaml:"automerge,omitempty" json:"automerge,omitempty" toml:"automerge,omitempty" validate:"omitempty"`
	Cleanup        *cleanup.Config   `yaml:"cleanup,omitempty" json:"cleanup,omitempty" toml:"cleanup,omitempty" validate:"omitempty"`
	Summarise      *summarise.Config `yaml:"summarise,omitempty" json:"summarise,omitempty" toml:"summarise,omitempty" validate:"omitempty"`
	Review         *review.Config    `yaml:"review,omitempty" json:"review,omitempty" toml:"review,omitempty" validate:"omitempty"`
	Scheduler      *scheduler.Config `yaml:"scheduler,omitempty" json:"scheduler,omitempty" toml:"scheduler,omitempty" validate:"omitempty"`
	Health         *health.Config    `yaml:"health,omitempty" json:"health,omitempty" toml:"health,omitempty" validate:"required"`
	*config.Config `yaml:",inline" json:",inline" toml:",inline" validate:"required"`
}

func decorateConfig(cfg *Config) *config.Config {
	return cfg.Config
}

func githubConfig(cfg *Config) *github.Config {
	return cfg.GitHub
}

func aiConfig(cfg *Config) *ai.Config {
	return cfg.AI
}

func automergeConfig(cfg *Config) *automerge.Config {
	return cfg.Automerge
}

func cleanupConfig(cfg *Config) *cleanup.Config {
	return cfg.Cleanup
}

func summariseConfig(cfg *Config) *summarise.Config {
	return cfg.Summarise
}

func reviewConfig(cfg *Config) *review.Config {
	return cfg.Review
}

func schedulerConfig(cfg *Config) *scheduler.Config {
	return cfg.Scheduler
}

func healthConfig(cfg *Config) *health.Config {
	return cfg.Health
}
