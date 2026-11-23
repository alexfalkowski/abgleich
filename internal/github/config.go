package github

import (
	"github.com/alexfalkowski/go-service/v3/config/client"
	"github.com/alexfalkowski/go-service/v3/time"
)

const defaultWebhookTimeout = 5 * time.Second

// Config identifies the owner whose installation repositories are reconciled.
// App credentials supply worker and comment access. The private key is loaded
// at startup; installation tokens renew automatically.
type Config struct {
	// Client configures the required REST and GraphQL endpoint, HTTP timeout, and safe-method retries.
	Client *client.Config `yaml:"client,omitempty" json:"client,omitempty" toml:"client,omitempty" validate:"required"`

	// App supplies required App and installation identity and the startup private-key source.
	App *AppConfig `yaml:"app,omitempty" json:"app,omitempty" toml:"app,omitempty" validate:"required"`

	// Webhook enables signed delivery processing when present; omission disables the endpoint.
	Webhook *WebhookConfig `yaml:"webhook,omitempty" json:"webhook,omitempty" toml:"webhook,omitempty" validate:"omitempty"`

	// Owner identifies the installation account whose owned repositories may be accessed.
	Owner string `yaml:"owner,omitempty" json:"owner,omitempty" toml:"owner,omitempty" validate:"required"`
}

// GetTimeout returns the HTTP request timeout, defaulting to one minute when
// client settings are absent or their timeout is zero.
func (c *Config) GetTimeout() time.Duration {
	if c.Client.IsEnabled() && c.Client.Timeout != 0 {
		return c.Client.Timeout
	}

	return time.Minute
}

// AppConfig identifies one installation. PrivateKey is a go-service source read
// once at startup; installation access tokens are renewed automatically.
type AppConfig struct {
	// PrivateKey is the required go-service RSA key source read once at startup.
	PrivateKey string `yaml:"private_key,omitempty" json:"private_key,omitempty" toml:"private_key,omitempty" validate:"required"`

	// ID identifies the GitHub App used for JWT authentication and installation validation.
	ID int64 `yaml:"id,omitempty" json:"id,omitempty" toml:"id,omitempty" validate:"gt=0"`

	// InstallationID selects the installation used for repository access and webhook identity checks.
	InstallationID int64 `yaml:"installation_id,omitempty" json:"installation_id,omitempty" toml:"installation_id,omitempty" validate:"gt=0"`
}

// WebhookConfig enables signed GitHub comment deliveries with App credentials.
// Secret is a go-service source read at startup. Timeout bounds synchronous
// processing and defaults to five seconds, below GitHub's ten-second deadline.
type WebhookConfig struct {
	// Secret is the required go-service signature-secret source read once at startup.
	Secret string `yaml:"secret,omitempty" json:"secret,omitempty" toml:"secret,omitempty" validate:"required"`

	// Timeout bounds synchronous delivery processing; zero uses five seconds and values must stay below ten seconds.
	Timeout time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty" toml:"timeout,omitempty"`
}

// IsEnabled reports whether webhook configuration is present.
func (c *WebhookConfig) IsEnabled() bool {
	return c != nil
}

// IsValid reports whether the configured processing budget is positive and below
// GitHub's ten-second deadline. A zero timeout uses the default budget.
func (c *WebhookConfig) IsValid() bool {
	timeout := c.GetTimeout()
	return timeout > 0 && timeout < 10*time.Second
}

// GetTimeout returns the delivery processing budget, defaulting to five seconds.
func (c *WebhookConfig) GetTimeout() time.Duration {
	if c.Timeout == 0 {
		return defaultWebhookTimeout
	}

	return c.Timeout
}
