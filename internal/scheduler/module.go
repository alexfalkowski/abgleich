package scheduler

import "github.com/alexfalkowski/go-signal/v2"

// Module provides the shared scheduler and connects it to startup and shutdown.
var Module = signal.Module(signal.Constructor(NewScheduler), signal.Register(register))

func register(lifecycle signal.Lifecycle, scheduler *Scheduler) {
	lifecycle.Append(signal.Hook{
		OnStart: scheduler.Start,
		OnStop:  scheduler.Stop,
	})
}
