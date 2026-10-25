package ai

import (
	"github.com/alexfalkowski/go-service/v2/config/client"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/alexfalkowski/go-service/v2/time"
)

// Config selects the optional text generation provider. Omission disables AI.
type Config struct {
	Client   *client.Config `yaml:"client,omitempty" json:"client,omitempty" toml:"client,omitempty"`
	OpenAI   *OpenAIConfig  `yaml:"openai,omitempty" json:"openai,omitempty" toml:"openai,omitempty" validate:"required"`
	Provider string         `yaml:"provider,omitempty" json:"provider,omitempty" toml:"provider,omitempty" validate:"required,oneof=openai"`
}

// OpenAIConfig supplies a go-service API key source read once at startup. URL
// optionally overrides https://api.openai.com/v1/ for compatible endpoints.
type OpenAIConfig struct {
	APIKey string `yaml:"api_key,omitempty" json:"api_key,omitempty" toml:"api_key,omitempty" validate:"required"`
	URL    string `yaml:"url,omitempty" json:"url,omitempty" toml:"url,omitempty" validate:"omitempty,http_url"`
}

// IsEnabled reports whether AI configuration is present.
func (c *Config) IsEnabled() bool {
	return c != nil
}

// GetTimeout returns the HTTP request timeout, defaulting to one minute when
// client settings are absent or their timeout is zero. The task's inference
// deadline remains an independent upper bound.
func (c *Config) GetTimeout() time.Duration {
	if c.Client.IsEnabled() && c.Client.Timeout != 0 {
		return c.Client.Timeout
	}

	return time.Minute
}

// GetURL returns the API base URL, defaulting to the OpenAI Responses endpoint.
func (c *OpenAIConfig) GetURL() string {
	if strings.IsEmpty(c.URL) {
		return "https://api.openai.com/v1/"
	}

	return c.URL
}
