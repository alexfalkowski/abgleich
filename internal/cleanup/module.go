package cleanup

import (
	"github.com/alexfalkowski/abgleich/internal/scheduler"
	"github.com/alexfalkowski/go-signal/v2"
)

// Module registers an independent cleanup job only when configured.
var Module = signal.Module(signal.Constructor(NewReconciler), signal.Register(registerSchedule))

func registerSchedule(schedules *scheduler.Scheduler, reconciler *Reconciler) error {
	if reconciler == nil {
		return nil
	}

	return schedules.Register("cleanup", reconciler.config, reconciler.Reconcile)
}
