package commands

import (
	"fmt"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/time"
	"github.com/alexfalkowski/go-sync"
)

const (
	maxReceiptTimeout = time.Second

	// Leave most of the delivery deadline for authorization and the action.
	receiptTimeoutDivisor = 4
)

// NewDispatcher constructs synchronous PR comment dispatch with no commands.
// Feature modules register their handlers before the application starts.
func NewDispatcher(service github.Service) *Dispatcher {
	return &Dispatcher{github: service, handlers: make(map[string]Handler)}
}

// Dispatcher selects handlers by exact command name and supplies shared
// installation validation and reaction feedback. Concurrent commands are
// rejected while another is running; no work is queued or retried.
type Dispatcher struct {
	github   github.Service
	handlers map[string]Handler
	mutex    sync.Mutex
}

// Register adds a handler under a single command name without arguments.
// Register before serving deliveries. Duplicate names wrap ErrCommandRegistered
// and leave the existing handler intact.
func (d *Dispatcher) Register(name string, handler Handler) error {
	if _, ok := d.handlers[name]; ok {
		return fmt.Errorf("%w: %s", ErrCommandRegistered, name)
	}

	d.handlers[name] = handler
	return nil
}

// Handle dispatches a verified PR comment within the caller's deadline.
// Unknown commands are ignored without feedback. A recognized command returns
// ErrCommandInProgress when busy, or its handler's complete outcome otherwise.
func (d *Dispatcher) Handle(ctx context.Context, event *github.IssueCommentEvent) error {
	handler, ok := d.handlers[Parse(event.GetComment().GetBody())]
	if !ok {
		return nil
	}

	if !d.mutex.TryLock() {
		return ErrCommandInProgress
	}
	defer d.mutex.Unlock()

	if err := d.github.ValidateInstallation(ctx); err != nil {
		return err
	}

	d.acknowledgeReceipt(ctx, event)

	accepted, err := handler.Handle(ctx, event)
	d.reportOutcome(ctx, event, accepted)

	return err
}

func (d *Dispatcher) acknowledgeReceipt(ctx context.Context, event *github.IssueCommentEvent) {
	timeout := maxReceiptTimeout
	if deadline, ok := ctx.Deadline(); ok {
		timeout = min(timeout, time.Until(deadline)/receiptTimeoutDivisor)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	d.react(ctx, event, "eyes")
}

func (d *Dispatcher) reportOutcome(ctx context.Context, event *github.IssueCommentEvent, accepted bool) {
	var reaction string
	if accepted {
		reaction = "+1"
	} else {
		reaction = "confused"
	}

	d.react(ctx, event, reaction)
}

func (d *Dispatcher) react(ctx context.Context, event *github.IssueCommentEvent, content string) {
	if ctx.Err() != nil {
		return
	}

	// The service logs feedback failures; they do not change command outcomes.
	_ = d.github.AddCommentReaction(ctx, event.GetRepo(), event.GetComment(), content)
}
