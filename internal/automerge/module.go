package automerge

import (
	"github.com/alexfalkowski/abgleich/internal/automerge/internal/strategy"
	"github.com/alexfalkowski/abgleich/internal/scheduler"
	"github.com/alexfalkowski/go-service/v2/di"
)

// Module constructs the reconciler and registers its scheduled job before startup.
var Module = di.Module(
	strategy.Module,
	di.Constructor(NewReconciler, NewCommands),
	di.Register(register),
)

func register(schedules *scheduler.Scheduler, cfg *scheduler.Config, reconciler *Reconciler) error {
	return schedules.Register("automerge", cfg, reconciler.Reconcile)
}
