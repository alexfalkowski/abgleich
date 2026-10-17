package scheduler

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/di"
)

// Module connects the scheduled job to application startup and shutdown.
var Module = di.Module(di.Register(register))

func register(lifecycle di.Lifecycle, job *Job) {
	lifecycle.Append(di.Hook{
		OnStart: func(context.Context) error {
			job.Start()
			return nil
		},
		OnStop: job.Stop,
	})
}
