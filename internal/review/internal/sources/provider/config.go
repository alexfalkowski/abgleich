package provider

import "github.com/alexfalkowski/go-service/v3/net/url"

// Config owns trusted, credential-free endpoints for shared review source access.
// Go and Ruby dependencies are selected from reviewed manifests and verified
// before any archive source is shared with the provider.
type Config struct {
	// GemRegistry selects the required trusted RubyGems registry or mirror.
	GemRegistry *url.SecureURL `yaml:"gem_registry,omitempty" json:"gem_registry,omitempty" toml:"gem_registry,omitempty" validate:"required"`

	// GoDocs selects the required trusted Go documentation base URL.
	GoDocs *url.SecureURL `yaml:"go_docs,omitempty" json:"go_docs,omitempty" toml:"go_docs,omitempty" validate:"required"`

	// ModuleProxy selects the required trusted Go module proxy.
	ModuleProxy *url.SecureURL `yaml:"module_proxy,omitempty" json:"module_proxy,omitempty" toml:"module_proxy,omitempty" validate:"required"`
}

// IsEnabled reports whether shared review source configuration is present.
func (c *Config) IsEnabled() bool {
	return c != nil
}
