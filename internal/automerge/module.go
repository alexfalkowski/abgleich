package automerge

import (
	"github.com/alexfalkowski/abgleich/internal/scheduler"
	"github.com/alexfalkowski/go-service/v2/di"
)

// Module constructs the rebaser and registers its scheduled job before startup.
var Module = di.Module(di.Constructor(NewRebaser), di.Register(register))

func register(schedules *scheduler.Scheduler, cfg *scheduler.Config, rebaser *Rebaser) error {
	return schedules.Register("automerge", cfg, rebaser.Rebase)
}
