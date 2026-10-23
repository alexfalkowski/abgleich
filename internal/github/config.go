package github

import (
	"github.com/alexfalkowski/go-service/v2/config/client"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/alexfalkowski/go-service/v2/time"
)

const defaultWebhookTimeout = 5 * time.Second

// Config identifies the owner whose installation repositories are reconciled.
// App credentials supply worker and comment access. The private key is loaded
// at startup; installation tokens renew automatically.
// URL optionally overrides the REST API base URL; an empty URL uses api.github.com.
type Config struct {
	Client  *client.Config `yaml:"client,omitempty" json:"client,omitempty" toml:"client,omitempty"`
	App     *AppConfig     `yaml:"app,omitempty" json:"app,omitempty" toml:"app,omitempty" validate:"required"`
	Webhook *WebhookConfig `yaml:"webhook,omitempty" json:"webhook,omitempty" toml:"webhook,omitempty" validate:"omitempty"`
	Owner   string         `yaml:"owner,omitempty" json:"owner,omitempty" toml:"owner,omitempty" validate:"required"`
	URL     string         `yaml:"url,omitempty" json:"url,omitempty" toml:"url,omitempty" validate:"omitempty,http_url"`
}

// AppConfig identifies one installation. PrivateKey is a go-service source read
// once at startup; installation access tokens are renewed automatically.
type AppConfig struct {
	PrivateKey     string `yaml:"private_key,omitempty" json:"private_key,omitempty" toml:"private_key,omitempty" validate:"required"`
	ID             int64  `yaml:"id,omitempty" json:"id,omitempty" toml:"id,omitempty" validate:"gt=0"`
	InstallationID int64  `yaml:"installation_id,omitempty" json:"installation_id,omitempty" toml:"installation_id,omitempty" validate:"gt=0"`
}

// WebhookConfig enables signed GitHub comment deliveries with App credentials.
// Secret is a go-service source read at startup. Timeout bounds synchronous
// processing and defaults to five seconds, below GitHub's ten-second deadline.
type WebhookConfig struct {
	Secret  string        `yaml:"secret,omitempty" json:"secret,omitempty" toml:"secret,omitempty" validate:"required"`
	Timeout time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty" toml:"timeout,omitempty"`
}

// GetTimeout returns the delivery processing budget, defaulting to five seconds.
func (c *WebhookConfig) GetTimeout() time.Duration {
	if c.Timeout == 0 {
		return defaultWebhookTimeout
	}
	return c.Timeout
}

// GetTimeout returns the HTTP request timeout, defaulting to one minute when
// client settings are absent or their timeout is zero.
func (c *Config) GetTimeout() time.Duration {
	if c.Client.IsEnabled() && c.Client.Timeout != 0 {
		return c.Client.Timeout
	}

	return time.Minute
}

// GetURL returns the GitHub REST API URL, defaulting to https://api.github.com/
// when no URL is configured.
func (c *Config) GetURL() string {
	if strings.IsEmpty(c.URL) {
		return "https://api.github.com/"
	}

	return c.URL
}
