package workflow

import "github.com/alexfalkowski/go-service/v2/strings"

// Config owns shared review source access. ModuleProxy is a trusted public Go
// module proxy; it receives module names only, without GitHub or AI credentials.
// Omission uses proxy.golang.org. Dependencies come from the reviewed go.mod,
// with a checksum from the reviewed go.sum required before source is shared.
type Config struct {
	GoDocs      string `yaml:"go_docs,omitempty" json:"go_docs,omitempty" toml:"go_docs,omitempty" validate:"omitempty,http_url"`
	ModuleProxy string `yaml:"module_proxy,omitempty" json:"module_proxy,omitempty" toml:"module_proxy,omitempty" validate:"omitempty,http_url"`
}

// GetGoDocs selects official Go release notes, or an operator-trusted mirror.
func (c *Config) GetGoDocs() string {
	if c == nil || strings.IsEmpty(c.GoDocs) {
		return "https://go.dev/doc"
	}

	return c.GoDocs
}

// GetModuleProxy returns the configured trusted proxy or the public Go proxy.
func (c *Config) GetModuleProxy() string {
	if c == nil || strings.IsEmpty(c.ModuleProxy) {
		return "https://proxy.golang.org"
	}

	return c.ModuleProxy
}
