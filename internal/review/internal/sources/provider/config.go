package provider

import "github.com/alexfalkowski/go-service/v3/strings"

// Config owns trusted, credential-free endpoints for shared review source access.
// Go and Ruby dependencies are selected from reviewed manifests and verified
// before any archive source is shared with the provider.
type Config struct {
	// GemRegistry selects a trusted RubyGems registry mirror; empty uses https://rubygems.org.
	GemRegistry string `yaml:"gem_registry,omitempty" json:"gem_registry,omitempty" toml:"gem_registry,omitempty" validate:"omitempty,http_url"`

	// GoDocs selects the trusted Go documentation base URL; empty uses https://go.dev/doc.
	GoDocs string `yaml:"go_docs,omitempty" json:"go_docs,omitempty" toml:"go_docs,omitempty" validate:"omitempty,http_url"`

	// ModuleProxy selects the trusted credential-free module proxy; empty uses https://proxy.golang.org.
	ModuleProxy string `yaml:"module_proxy,omitempty" json:"module_proxy,omitempty" toml:"module_proxy,omitempty" validate:"omitempty,http_url"`
}

// IsEnabled reports whether shared review source configuration is present.
func (c *Config) IsEnabled() bool {
	return c != nil
}

// GetGoDocs selects official Go release notes, or an operator-trusted mirror.
func (c *Config) GetGoDocs() string {
	if !c.IsEnabled() || strings.IsEmpty(c.GoDocs) {
		return "https://go.dev/doc"
	}

	return c.GoDocs
}

// GetModuleProxy returns the configured trusted proxy or the public Go proxy.
func (c *Config) GetModuleProxy() string {
	if !c.IsEnabled() || strings.IsEmpty(c.ModuleProxy) {
		return "https://proxy.golang.org"
	}

	return c.ModuleProxy
}

// GetGemRegistry returns the trusted public gem endpoint, without credentials.
func (c *Config) GetGemRegistry() string {
	if !c.IsEnabled() || strings.IsEmpty(c.GemRegistry) {
		return "https://rubygems.org"
	}

	return c.GemRegistry
}
