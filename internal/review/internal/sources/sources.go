package sources

import (
	"cmp"

	"github.com/alexfalkowski/abgleich/internal/review/internal/sources/provider"
	"github.com/alexfalkowski/go-service/v2/di"
	"github.com/alexfalkowski/go-service/v2/slices"
)

// RegistryParams collects registered strategies independently of language implementations.
type RegistryParams struct {
	// In marks the provider group as injected dependencies.
	di.In

	// Providers includes optional contributions from source modules; DI group order is unspecified.
	Providers []provider.Strategy `group:"review_sources"`
}

// NewRegistry removes disabled contributions and orders guides consistently across restarts.
func NewRegistry(params RegistryParams) *Registry {
	providers := slices.DeleteFunc(params.Providers, func(strategy provider.Strategy) bool { return strategy == nil })
	slices.SortFunc(providers, func(a, b provider.Strategy) int { return cmp.Compare(a.Guide(), b.Guide()) })

	return &Registry{providers: providers}
}

// Registry selects applicable providers without importing their implementations.
type Registry struct {
	providers []provider.Strategy
}

// Select includes providers matching reviewed paths, including deleted paths.
// Selection does not fetch external source or run any tools.
func (r *Registry) Select(paths []string) []provider.Strategy {
	var selected []provider.Strategy
	for _, strategy := range r.providers {
		if strategy.Matches(paths) {
			selected = append(selected, strategy)
		}
	}

	return selected
}
