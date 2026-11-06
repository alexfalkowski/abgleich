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
	// GitHub supplies required owner, App credentials, and optional webhook and HTTP settings.
	GitHub *github.Config `yaml:"github,omitempty" json:"github,omitempty" toml:"github,omitempty" validate:"required"`

	// AI optionally enables the text generation provider used by summaries and reviews.
	AI *ai.Config `yaml:"ai,omitempty" json:"ai,omitempty" toml:"ai,omitempty" validate:"omitempty"`

	// Automerge optionally customizes Dependabot enrollment and event-driven rebase checks.
	Automerge *automerge.Config `yaml:"automerge,omitempty" json:"automerge,omitempty" toml:"automerge,omitempty" validate:"omitempty"`

	// Cleanup enables branch cleanup and its independent schedule when present.
	Cleanup *cleanup.Config `yaml:"cleanup,omitempty" json:"cleanup,omitempty" toml:"cleanup,omitempty" validate:"omitempty"`

	// Summarise enables AI summaries with their own model and job schedule when present.
	Summarise *summarise.Config `yaml:"summarise,omitempty" json:"summarise,omitempty" toml:"summarise,omitempty" validate:"omitempty"`

	// Review configures optional code and security reviews and their shared source access.
	Review *review.Config `yaml:"review,omitempty" json:"review,omitempty" toml:"review,omitempty" validate:"omitempty"`

	// Scheduler sets the auto-merge sweep schedule; omission uses scheduler defaults.
	Scheduler *scheduler.Config `yaml:"scheduler,omitempty" json:"scheduler,omitempty" toml:"scheduler,omitempty" validate:"omitempty"`

	// Health supplies required sampling and connectivity-check timing independently of job outcomes.
	Health *health.Config `yaml:"health,omitempty" json:"health,omitempty" toml:"health,omitempty" validate:"required"`

	// Config embeds shared go-service configuration for transport, telemetry, and other service facilities.
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
