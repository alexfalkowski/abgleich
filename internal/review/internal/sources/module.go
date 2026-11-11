package sources

import (
	"github.com/alexfalkowski/abgleich/internal/review/internal/sources/cucumber"
	"github.com/alexfalkowski/abgleich/internal/review/internal/sources/golang"
	"github.com/alexfalkowski/abgleich/internal/review/internal/sources/ruby"
	"github.com/alexfalkowski/go-service/v2/di"
)

// Module constructs registered source strategies and their shared transports.
// Review configuration supplies provider.Config; omission disables construction.
var Module = di.Module(
	golang.Module,
	ruby.Module,
	cucumber.Module,
	di.Constructor(NewRegistry),
)
