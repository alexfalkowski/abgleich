package automerge

import (
	"github.com/alexfalkowski/abgleich/internal/automerge/internal/strategy"
	"github.com/alexfalkowski/abgleich/internal/commands"
	"github.com/alexfalkowski/abgleich/internal/scheduler"
	"github.com/alexfalkowski/go-service/v2/di"
)

// Module constructs and registers the scheduled reconciler and comment command
// before startup.
var Module = di.Module(
	strategy.Module,
	di.Constructor(NewReconciler, NewCommand),
	di.Register(registerSchedule),
	di.Register(registerCommand),
)

func registerSchedule(schedules *scheduler.Scheduler, cfg *scheduler.Config, reconciler *Reconciler) error {
	return schedules.Register("automerge", cfg, reconciler.Reconcile)
}

func registerCommand(dispatcher *commands.Dispatcher, command *Command) error {
	return dispatcher.Register(autoMergeCommand, command)
}
