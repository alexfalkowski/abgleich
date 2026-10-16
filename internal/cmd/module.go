package cmd

import (
	"github.com/alexfalkowski/abgleich/internal/config"
	"github.com/alexfalkowski/abgleich/internal/health"
	"github.com/alexfalkowski/go-service/v2/di"
	"github.com/alexfalkowski/go-service/v2/module"
)

// Module for fx.
var Module = di.Module(
	module.Server,
	config.Module,
	health.Module,
)
