package ai

import (
	"github.com/alexfalkowski/go-service/v3/config/client"
	"github.com/alexfalkowski/go-service/v3/strings"
	"github.com/alexfalkowski/go-service/v3/time"
)

// ProviderKind identifies the adapter used by a connection.
type ProviderKind string

// ProviderOpenAI selects the OpenAI Responses adapter.
const ProviderOpenAI ProviderKind = "openai"

// Config defines optional named AI connections. Omission disables AI.
type Config struct {
	// Providers defines connections selected by each feature's provider setting.
	Providers map[string]*ProviderConfig `yaml:"providers,omitempty" json:"providers,omitempty" toml:"providers,omitempty" validate:"required,dive,required"`
}

// IsEnabled reports whether AI configuration is present.
func (c *Config) IsEnabled() bool {
	return c != nil
}

// ProviderConfig defines a provider connection shared by any number of models.
type ProviderConfig struct {
	// Client sets connection settings and HTTP deadlines; retries stay disabled.
	Client *client.Config `yaml:"client,omitempty" json:"client,omitempty" toml:"client,omitempty"`

	// Kind selects the adapter: openai.
	Kind ProviderKind `yaml:"kind,omitempty" json:"kind,omitempty" toml:"kind,omitempty" validate:"required,oneof=openai"`

	// APIKey is a required credential source resolved once at startup.
	APIKey string `yaml:"api_key,omitempty" json:"api_key,omitempty" toml:"api_key,omitempty" validate:"required"`

	// URL optionally overrides the adapter's trusted API endpoint.
	URL string `yaml:"url,omitempty" json:"url,omitempty" toml:"url,omitempty" validate:"omitempty,http_url"`
}

// GetTimeout returns the per-request deadline, defaulting to one minute.
func (c *ProviderConfig) GetTimeout() time.Duration {
	if c.Client.IsEnabled() && c.Client.Timeout != 0 {
		return c.Client.Timeout
	}
	return time.Minute
}

// GetURL returns the explicit endpoint or the selected provider's default.
func (c *ProviderConfig) GetURL() string {
	if !strings.IsEmpty(c.URL) {
		return c.URL
	}
	switch c.Kind {
	case ProviderOpenAI:
		return "https://api.openai.com/v1/"
	default:
		return ""
	}
}
