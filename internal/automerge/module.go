package automerge

import (
	"github.com/alexfalkowski/abgleich/internal/scheduler"
	"github.com/alexfalkowski/go-service/v2/di"
)

// Module constructs the reconciler and registers its scheduled job before startup.
var Module = di.Module(di.Constructor(NewReconciler), di.Register(register))

func register(schedules *scheduler.Scheduler, cfg *scheduler.Config, reconciler *Reconciler) error {
	return schedules.Register("automerge", cfg, reconciler.Reconcile)
}
