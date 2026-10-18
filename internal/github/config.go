package github

import (
	"github.com/alexfalkowski/go-service/v2/config/client"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/alexfalkowski/go-service/v2/time"
)

// Config identifies the account whose owned repositories are reconciled.
// Token is a go-service source (for example env:GITHUB_TOKEN or
// file:/run/secrets/github-token), resolved once at startup. Credential changes
// require a process restart. URL optionally overrides the REST API base URL;
// an empty URL uses api.github.com.
type Config struct {
	Client *client.Config `yaml:"client,omitempty" json:"client,omitempty" toml:"client,omitempty"`
	Owner  string         `yaml:"owner,omitempty" json:"owner,omitempty" toml:"owner,omitempty" validate:"required"`
	Token  string         `yaml:"token,omitempty" json:"token,omitempty" toml:"token,omitempty" validate:"required"`
	URL    string         `yaml:"url,omitempty" json:"url,omitempty" toml:"url,omitempty" validate:"omitempty,http_url"`
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
