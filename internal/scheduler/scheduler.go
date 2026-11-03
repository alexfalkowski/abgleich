package scheduler

import (
	"fmt"

	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/telemetry/logger"
	"github.com/alexfalkowski/go-service/v2/time"
	"github.com/alexfalkowski/go-sync"
)

// NewScheduler constructs a scheduler with no jobs. Feature packages register
// their jobs before the application starts.
func NewScheduler(log *logger.Logger) *Scheduler {
	return &Scheduler{logger: log}
}

// Scheduler runs registered jobs independently and coordinates their lifecycle.
// Registration must finish before Start; call Start and Stop once through the
// service lifecycle.
type Scheduler struct {
	logger *logger.Logger
	jobs   []*Job
}

// Register adds a job with its own schedule and handler. Names must be unique;
// a duplicate wraps [ErrJobRegistered] with the name and leaves the existing job intact. Register
// before Start. The handler must observe cancellation and return its full outcome.
// Outcomes and overruns are logged after each run.
// A nil configuration uses the default interval and timeout.
func (s *Scheduler) Register(name string, cfg Schedule, handler sync.Handler) error {
	return s.RegisterTriggered(name, cfg, handler, nil, nil)
}

// RegisterTriggered adds one job with separate scheduled and notified handlers.
// Notifications coalesce while either handler runs and request a later notified
// pass. Both handlers share the job deadline, overlap protection, and shutdown.
// ready must remain open for the scheduler's lifetime and requires a non-nil
// triggered handler. Pass nil for both to disable notifications.
func (s *Scheduler) RegisterTriggered(name string, cfg Schedule, handler, triggered sync.Handler, ready <-chan struct{}) error {
	for _, job := range s.jobs {
		if job.name == name {
			return fmt.Errorf("%w: %s", ErrJobRegistered, name)
		}
	}

	job := NewJob(name, cfg, handler)
	job.handler = s.logHandler(job, handler, "schedule")
	job.ready = ready

	if triggered != nil {
		job.triggered = s.logHandler(job, triggered, "queue")
	}

	s.jobs = append(s.jobs, job)

	return nil
}

// Start begins each job immediately and schedules subsequent runs. Jobs do not
// block one another, and each job skips starts while its previous run is active.
func (s *Scheduler) Start(ctx context.Context) error {
	for _, job := range s.jobs {
		if err := job.Start(ctx); err != nil {
			return fmt.Errorf("scheduler: start job %q: %w", job.name, err)
		}
	}

	return nil
}

// Stop cancels every job before waiting concurrently for active work. It waits
// until all jobs finish or ctx expires, returning joined failures identified by job name.
// Start must precede Stop.
func (s *Scheduler) Stop(ctx context.Context) error {
	// Cancel all jobs first so a slow handler cannot delay another job's cancellation.
	for _, job := range s.jobs {
		job.cancel()
	}

	group := sync.ErrorsGroup{}

	for _, job := range s.jobs {
		group.Go(func() error {
			if err := job.Stop(ctx); err != nil {
				return fmt.Errorf("scheduler: stop job %q: %w", job.name, err)
			}

			return nil
		})
	}

	return group.Wait()
}

func (s *Scheduler) logHandler(job *Job, handler sync.Handler, trigger string) sync.Handler {
	return func(ctx context.Context) error {
		started := time.Now()
		err := handler(ctx)
		if err == nil {
			err = context.Cause(ctx)
		}

		duration := time.Since(started)
		interval := job.config.GetInterval()
		overrun := duration > interval

		message := logger.NewMessage("job finished", err)
		level := message.Level()
		if err == nil && overrun {
			level = logger.LevelWarn
		}

		s.logger.LogAttrs(ctx, level, message,
			logger.String("job", job.name), logger.String("trigger", trigger), logger.String("duration", duration.String()),
			logger.String("interval", interval.String()), logger.Bool("overrun", overrun))

		return err
	}
}
