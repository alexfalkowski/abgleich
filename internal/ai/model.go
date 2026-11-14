package ai

import "github.com/alexfalkowski/go-service/v3/strings"

// ModelConfig selects a named provider connection, model, and optional effort.
type ModelConfig struct {
	// Provider names a connection under ai.providers.
	Provider string `yaml:"provider,omitempty" json:"provider,omitempty" toml:"provider,omitempty" validate:"required"`

	// Model selects the required provider model for generation.
	Model string `yaml:"model,omitempty" json:"model,omitempty" toml:"model,omitempty" validate:"required"`

	// Effort optionally selects model-supported effort; empty leaves its default.
	Effort Effort `yaml:"effort,omitempty" json:"effort,omitempty" toml:"effort,omitempty" validate:"omitempty,oneof=none minimal low medium high xhigh max"`
}

// Effort selects provider effort; the zero value preserves the model default.
type Effort string

// EffortNone requests no reasoning when supported by the model.
const EffortNone Effort = "none"

// EffortMinimal requests minimal reasoning when supported by the model.
const EffortMinimal Effort = "minimal"

// EffortLow requests low effort when supported by the model.
const EffortLow Effort = "low"

// EffortMedium requests medium effort when supported by the model.
const EffortMedium Effort = "medium"

// EffortHigh requests high effort when supported by the model.
const EffortHigh Effort = "high"

// EffortExtraHigh requests extra-high effort when supported by the model.
const EffortExtraHigh Effort = "xhigh"

// EffortMax requests maximum effort when supported by the model.
const EffortMax Effort = "max"

// String returns the configured effort, or empty for the model default.
func (e Effort) String() string {
	return string(e)
}

// IsDefined reports whether effort was explicitly selected.
func (e Effort) IsDefined() bool {
	return !strings.IsEmpty(e.String())
}
