package provider

import "github.com/alexfalkowski/go-service/v3/config/client"

// Config owns trusted, credential-free HTTP clients for shared review source access.
// Go and Ruby dependencies are selected from reviewed manifests and verified
// before any archive source is shared with the provider.
type Config struct {
	// GemRegistry configures the required trusted RubyGems registry or mirror client.
	GemRegistry *client.Config `yaml:"gem_registry,omitempty" json:"gem_registry,omitempty" toml:"gem_registry,omitempty" validate:"required"`

	// GoDocs configures the required trusted Go documentation client.
	GoDocs *client.Config `yaml:"go_docs,omitempty" json:"go_docs,omitempty" toml:"go_docs,omitempty" validate:"required"`

	// ModuleProxy configures the required trusted Go module proxy client.
	ModuleProxy *client.Config `yaml:"module_proxy,omitempty" json:"module_proxy,omitempty" toml:"module_proxy,omitempty" validate:"required"`
}

// IsEnabled reports whether shared review source configuration is present.
func (c *Config) IsEnabled() bool {
	return c != nil
}
