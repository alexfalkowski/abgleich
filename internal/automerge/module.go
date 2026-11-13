package automerge

import (
	"github.com/alexfalkowski/abgleich/internal/automerge/internal/strategy"
	"github.com/alexfalkowski/abgleich/internal/commands"
	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/abgleich/internal/scheduler"
	"github.com/alexfalkowski/go-signal/v2"
)

// Module provides Dependabot, label, and rebase strategies in precedence order,
// connects the coordinator, and registers its sweep and command before startup.
var Module = signal.Module(
	signal.Constructor(NewReconciler, NewCommand, NewCoordinator, strategies),
	signal.Register(registerCoordinator),
	signal.Register(registerSchedule),
	signal.Register(registerCommand),
)

func strategies(service github.Service, cfg *Config) []strategy.Strategy {
	return []strategy.Strategy{
		strategy.NewDependabot(service, cfg.GetDependabotExclusions()),
		strategy.NewLabel(service),
		strategy.NewRebase(service),
	}
}

func registerCoordinator(lifecycle signal.Lifecycle, coordinator *Coordinator) {
	lifecycle.Append(signal.Hook{
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
