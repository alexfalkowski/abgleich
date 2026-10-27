package automerge

import (
	"github.com/alexfalkowski/abgleich/internal/automerge/internal/strategy"
	"github.com/alexfalkowski/abgleich/internal/commands"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/abgleich/internal/scheduler"
	"github.com/alexfalkowski/go-service/v2/di"
)

// Module provides Dependabot, label, and rebase strategies in precedence order,
// connects the coordinator, and registers its sweep and command before startup.
var Module = di.Module(
	di.Constructor(NewReconciler, NewCommand, NewCoordinator, strategies),
	di.Register(registerCoordinator),
	di.Register(registerSchedule),
	di.Register(registerCommand),
)

func strategies(service github.Service, cfg *Config) []strategy.Strategy {
	return []strategy.Strategy{
		strategy.NewDependabot(service, cfg.GetDependabotExclusions()),
		strategy.NewLabel(service),
		strategy.NewRebase(service),
	}
}

func registerCoordinator(lifecycle di.Lifecycle, coordinator *Coordinator) {
	lifecycle.Append(di.Hook{
		OnStart: coordinator.Start,
		OnStop:  coordinator.Stop,
	})
}

func registerSchedule(schedules *scheduler.Scheduler, cfg *scheduler.Config, coordinator *Coordinator) error {
	return schedules.Register("automerge", cfg, coordinator.Reconcile)
}

func registerCommand(dispatcher *commands.Dispatcher, command *Command) error {
	return dispatcher.Register(autoMergeCommand, command)
}
