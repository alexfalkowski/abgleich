package scheduler

import (
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/time"
	"github.com/alexfalkowski/go-sync"
)

// NewJob constructs a job whose handler observes cancellation and returns its
// complete run outcome. Register through Scheduler.Register to add outcome logging.
// A nil schedule uses Config's shared interval and timeout defaults.
func NewJob(name string, cfg Schedule, handler sync.Handler) *Job {
	if cfg == nil {
		cfg = &Config{}
	}

	return &Job{name: name, config: cfg, handler: handler, worker: sync.NewWorker(1)}
}

// Job runs a scheduled operation with a timeout and overlap protection.
// Construct it with NewJob and connect Start and Stop to the service lifecycle.
type Job struct {
	handler   sync.Handler
	triggered sync.Handler
	ready     <-chan struct{}

	worker  *sync.Worker
	cancel  context.CancelFunc
	stopped chan struct{}

	config Schedule
	name   string
}

// Start begins the job immediately and schedules subsequent runs without
// overlapping. Call Start once through the service lifecycle; Stop cancels the
// schedule and active work.
func (j *Job) Start(context.Context) error {
	ctx, cancel := context.WithCancel(context.Background())
	j.cancel = cancel
	j.stopped = make(chan struct{})

	go j.loop(ctx)

	return nil
}

// Stop cancels scheduling and active work, then waits for completion until ctx
// expires. Start must precede Stop, and the handler must observe cancellation.
func (j *Job) Stop(ctx context.Context) error {
	j.cancel()

	select {
	case <-j.stopped:
		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}

func (j *Job) loop(ctx context.Context) {
	defer close(j.stopped)
	defer func() { _ = j.worker.Wait(context.Background()) }()

	ticker := time.NewTicker(j.config.GetInterval())
	defer ticker.Stop()

	_ = j.worker.TrySchedule(ctx, j.hook(j.handler))

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = j.worker.TrySchedule(ctx, j.hook(j.handler))
		case <-j.ready:
			_ = j.worker.Schedule(ctx, j.hook(j.triggered))

			// Scheduled starts missed while waiting for the worker stay skipped.
			select {
			case <-ticker.C:
			default:
			}
		}
	}
}

func (j *Job) hook(handler sync.Handler) sync.Hook {
	return sync.Hook{OnRun: func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, j.config.GetTimeout())
		defer cancel()

		return handler(ctx)
	}}
}
