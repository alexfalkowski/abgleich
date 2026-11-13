package config

import (
	"github.com/alexfalkowski/go-service/v3/config"
	"github.com/alexfalkowski/go-signal/v2"
)

// Module decodes worker configuration and supplies package-owned settings.
var Module = signal.Module(
	signal.Constructor(config.NewConfig[Config]),
	signal.Decorate(decorateConfig),
	signal.Constructor(
		githubConfig, aiConfig, automergeConfig, cleanupConfig,
		summariseConfig, reviewConfig, schedulerConfig, healthConfig,
	),
)
