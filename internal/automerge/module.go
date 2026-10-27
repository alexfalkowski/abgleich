package automerge

import (
	"github.com/alexfalkowski/abgleich/internal/automerge/internal/strategy"
	"github.com/alexfalkowski/abgleich/internal/commands"
	"github.com/alexfalkowski/abgleich/internal/scheduler"
	"github.com/alexfalkowski/go-service/v2/di"
)

// Module connects the shared coordinator before the scheduler starts and registers
// the scheduled sweep and comment command before startup.
var Module = di.Module(
	strategy.Module,
	di.Constructor(NewReconciler, NewCommand, NewCoordinator),
	di.Register(registerCoordinator),
	di.Register(registerSchedule),
	di.Register(registerCommand),
)

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
