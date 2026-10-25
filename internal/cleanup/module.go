package cleanup

import (
	"github.com/alexfalkowski/abgleich/internal/scheduler"
	"github.com/alexfalkowski/go-service/v2/di"
)

// Module registers an independent cleanup job only when configured.
var Module = di.Module(di.Constructor(NewReconciler), di.Register(registerSchedule))

func registerSchedule(schedules *scheduler.Scheduler, reconciler *Reconciler) error {
	if reconciler.config == nil {
		return nil
	}

	return schedules.Register("cleanup", reconciler.config, reconciler.Reconcile)
}
