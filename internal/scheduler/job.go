package scheduler

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/time"
	"github.com/alexfalkowski/go-sync"
)

// NewJob constructs a job whose handler observes cancellation and returns its
// complete run outcome. Register through Scheduler.Register to add outcome logging.
func NewJob(name string, cfg *Config, handler sync.Handler) *Job {
	return &Job{name: name, config: cfg, handler: handler}
}

// Job runs a scheduled operation with a timeout and overlap protection.
// Construct it with NewJob and connect Start and Stop to the service lifecycle.
type Job struct {
	handler sync.Handler
	config  *Config
	cancel  context.CancelFunc
	stopped chan struct{}
	name    string
	runs    sync.WaitGroup
	running sync.Mutex
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
	defer j.runs.Wait()

	ticker := time.NewTicker(j.config.GetInterval())
	defer ticker.Stop()

	j.run(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			j.run(ctx)
		}
	}
}

func (j *Job) run(ctx context.Context) {
	if !j.running.TryLock() {
		return
	}

	j.runs.Go(func() {
		defer j.running.Unlock()

		ctx, cancel := context.WithTimeout(ctx, j.config.GetTimeout())
		defer cancel()

		_ = j.handler(ctx)
	})
}
