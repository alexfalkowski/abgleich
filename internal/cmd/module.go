package cmd

import (
	"github.com/alexfalkowski/abgleich/internal/ai"
	"github.com/alexfalkowski/abgleich/internal/automerge"
	"github.com/alexfalkowski/abgleich/internal/commands"
	"github.com/alexfalkowski/abgleich/internal/config"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/abgleich/internal/health"
	"github.com/alexfalkowski/abgleich/internal/scheduler"
	"github.com/alexfalkowski/abgleich/internal/status"
	"github.com/alexfalkowski/abgleich/internal/summarise"
	"github.com/alexfalkowski/abgleich/internal/transport/http"
	"github.com/alexfalkowski/abgleich/internal/transport/http/client"
	"github.com/alexfalkowski/go-service/v2/di"
	"github.com/alexfalkowski/go-service/v2/module"
)

// Module for fx.
var Module = di.Module(
	module.Server,
	config.Module,
	client.Module,
	github.Module,
	commands.Module,
	automerge.Module,
	status.Module,
	ai.Module,
	summarise.Module,
	http.Module,
	scheduler.Module,
	health.Module,
)
