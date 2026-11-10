package cucumber

import (
	_ "embed"
	"path"

	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/abgleich/internal/review/internal/sources/provider"
)

//go:embed prompt.txt
var guide string

// NewProvider contributes framework guidance to the review source DI group.
func NewProvider() provider.Result {
	return provider.Result{Strategy: Provider{}}
}

// Provider adds framework guidance without external source access or mutable state.
type Provider struct{}

// Matches selects Cucumber guidance for repositories with Gherkin features.
func (Provider) Matches(paths []string) bool {
	for _, name := range paths {
		if path.Ext(name) == ".feature" {
			return true
		}
	}

	return false
}

// Guide returns worker-owned framework review instructions.
func (Provider) Guide() string { return guide }

// Tools returns no tools because this provider only supplies framework guidance.
func (Provider) Tools(provider.Repository) []ai.Tool { return nil }
