package scheduler

import "github.com/alexfalkowski/go-service/v2/di"

// Module provides the shared scheduler and connects it to startup and shutdown.
var Module = di.Module(di.Constructor(NewScheduler), di.Register(register))

func register(lifecycle di.Lifecycle, scheduler *Scheduler) {
	lifecycle.Append(di.Hook{
		OnStart: scheduler.Start,
		OnStop:  scheduler.Stop,
	})
}
