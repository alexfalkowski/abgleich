package sources

import (
	"github.com/alexfalkowski/abgleich/internal/review/internal/sources/cucumber"
	"github.com/alexfalkowski/abgleich/internal/review/internal/sources/golang"
	"github.com/alexfalkowski/abgleich/internal/review/internal/sources/ruby"
	"github.com/alexfalkowski/go-signal/v2"
)

// Module constructs registered source strategies and their shared transports.
// Review configuration supplies provider.Config; omission disables construction.
var Module = signal.Module(
	golang.Module,
	ruby.Module,
	cucumber.Module,
	signal.Constructor(NewRegistry),
)
