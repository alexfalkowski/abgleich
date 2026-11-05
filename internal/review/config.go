package review

import (
	"github.com/alexfalkowski/abgleich/internal/review/code"
	"github.com/alexfalkowski/abgleich/internal/review/internal/workflow"
	"github.com/alexfalkowski/abgleich/internal/review/sec"
)

// Config composes independent code and security review settings with shared
// source access. Omitted Code or Sec disables only that review command and job.
type Config struct {
	// Code enables code reviews and their independent model and schedule when present.
	Code *code.Config `yaml:"code,omitempty" json:"code,omitempty" toml:"code,omitempty" validate:"omitempty"`
	// Sec enables security reviews and their independent model and schedule when present.
	Sec *sec.Config `yaml:"sec,omitempty" json:"sec,omitempty" toml:"sec,omitempty" validate:"omitempty"`
	// Config shares trusted source endpoint settings between both review features.
	workflow.Config `yaml:",inline" json:",inline" toml:",inline"`
}

// IsEnabled reports whether review configuration is present. Each feature's
// configuration independently enables its command and job.
func (c *Config) IsEnabled() bool {
	return c != nil
}

func codeConfig(cfg *Config) *code.Config {
	if !cfg.IsEnabled() {
		return nil
	}

	return cfg.Code
}

func secConfig(cfg *Config) *sec.Config {
	if !cfg.IsEnabled() {
		return nil
	}

	return cfg.Sec
}

func workflowConfig(cfg *Config) *workflow.Config {
	if !cfg.IsEnabled() {
		return nil
	}

	return &cfg.Config
}
