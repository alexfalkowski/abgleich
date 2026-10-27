package automerge

import (
	"github.com/alexfalkowski/abgleich/internal/scheduler"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/alexfalkowski/go-service/v2/telemetry/logger"
	"github.com/alexfalkowski/go-service/v2/time"
	"github.com/alexfalkowski/go-sync"
)

type reconcileRequest struct {
	context context.Context
	result  chan error
}

// NewCoordinator constructs the shared execution path for scheduled sweeps and
// webhook checks. Pending repositories are held only in memory.
func NewCoordinator(reconciler *Reconciler, cfg *scheduler.Config, log *logger.Logger) *Coordinator {
	return &Coordinator{
		reconciler: reconciler, config: cfg, logger: log,
		repositories: make(map[string]bool), wake: make(chan struct{}, 1), reconcileRequests: make(chan reconcileRequest),
	}
}

// Coordinator serializes all auto-merge checks within one process. Pending
// repositories coalesce; events during a check request one subsequent pass.
// Start and Stop must be connected to the service lifecycle before scheduling.
type Coordinator struct {
	reconciler        *Reconciler
	config            *scheduler.Config
	logger            *logger.Logger
	repositories      map[string]bool // A false entry is running; true entries have a pending pass.
	wake              chan struct{}
	reconcileRequests chan reconcileRequest
	cancel            context.CancelFunc
	stopped           chan struct{}
	pending           []string
	mutex             sync.Mutex
}

// Start begins the processing loop. Call once before the scheduler starts.
func (c *Coordinator) Start(context.Context) error {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.stopped = make(chan struct{})

	go c.loop(ctx)

	return nil
}

// Stop cancels active webhook work and waits for the loop to exit. Stop the
// scheduler first so accepted sweeps have already returned their full outcome.
func (c *Coordinator) Stop(ctx context.Context) error {
	c.cancel()

	select {
	case <-c.stopped:
		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}

// Reconcile waits for exclusive execution and returns the complete sweep
// outcome. The caller's deadline includes waiting behind webhook work.
func (c *Coordinator) Reconcile(ctx context.Context) error {
	request := reconcileRequest{context: ctx, result: make(chan error, 1)}
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case c.reconcileRequests <- request:
		return <-request.result
	}
}

// Enqueue accepts a repository name without waiting for GitHub. Names ignore
// case, and repeated notifications for a pending repository coalesce.
func (c *Coordinator) Enqueue(repository string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	repository = strings.ToLower(repository)

	if c.repositories[repository] {
		return
	}

	c.repositories[repository] = true
	c.pending = append(c.pending, repository)
	c.notify()
}

func (c *Coordinator) loop(ctx context.Context) {
	defer close(c.stopped)

	for {
		if ctx.Err() != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case request := <-c.reconcileRequests:
			request.result <- c.reconciler.Reconcile(request.context)
		case <-c.wake:
			c.process(ctx)
		}
	}
}

func (c *Coordinator) process(ctx context.Context) {
	c.mutex.Lock()

	if len(c.pending) == 0 {
		c.mutex.Unlock()
		return
	}

	repository := c.pending[0]
	c.pending[0] = ""
	c.pending = c.pending[1:]
	c.repositories[repository] = false
	c.mutex.Unlock()

	c.reconcileRepository(ctx, repository)

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.repositories[repository] {
		delete(c.repositories, repository)
	}

	if len(c.pending) != 0 {
		c.notify()
	}
}

func (c *Coordinator) reconcileRepository(ctx context.Context, repository string) {
	ctx, cancel := context.WithTimeout(ctx, c.config.GetTimeout())
	defer cancel()

	started := time.Now()

	err := c.reconciler.ReconcileRepository(ctx, repository)
	if err == nil {
		err = context.Cause(ctx)
	}

	message := logger.NewMessage("automerge event finished", err)
	c.logger.LogAttrs(ctx, message.Level(), message,
		logger.String("repository", repository), logger.String("duration", time.Since(started).String()))
}

func (c *Coordinator) notify() {
	select {
	case c.wake <- struct{}{}:
	default:
	}
}
