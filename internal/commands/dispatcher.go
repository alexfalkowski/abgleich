package commands

import (
	"fmt"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/time"
	"github.com/alexfalkowski/go-sync"
)

const (
	maxReceiptTimeout    = time.Second
	maxProcessingReserve = time.Second

	// Leave most of the delivery deadline for authorization and the action.
	receiptTimeoutDivisor    = 4
	processingReserveDivisor = 4
)

// NewDispatcher constructs synchronous PR comment dispatch with no commands.
// Feature modules register their handlers before the application starts.
func NewDispatcher(service github.Service) *Dispatcher {
	return &Dispatcher{github: service, handlers: make(map[string]Handler), slot: sync.NewSemaphore(1)}
}

// Dispatcher selects handlers by exact command name and supplies shared
// installation validation and reaction feedback. When the command slot is
// busy, only deliveries with current write access can wait within their deadlines.
type Dispatcher struct {
	github      github.Service
	handlers    map[string]Handler
	slot        *sync.Semaphore
	permissions permissionBudget
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
// its handler's complete outcome or the context error if its wait expires.
func (d *Dispatcher) Handle(ctx context.Context, event *github.IssueCommentEvent) error {
	handler, ok := d.handlers[Parse(event.GetComment().GetBody())]
	if !ok {
		return nil
	}

	if acquired, err := d.acquire(ctx, event); err != nil || !acquired {
		return err
	}

	defer d.slot.Release(1)

	if err := d.github.ValidateInstallation(ctx); err != nil {
		return err
	}

	d.acknowledgeReceipt(ctx, event)

	accepted, err := handler.Handle(ctx, event)
	d.reportOutcome(ctx, event, accepted)

	return err
}

func (d *Dispatcher) acquire(ctx context.Context, event *github.IssueCommentEvent) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if d.slot.TryAcquire(1) {
		return true, nil
	}

	comment := event.GetComment()
	author := comment.GetUser()
	if author.GetType() != github.UserHuman || !github.HasCommentMetadata(comment) {
		return false, nil
	}

	// Check admission outside the slot. The handler still authorizes the live
	// comment and actor before acting, including after a wait.
	waitCtx, cancel := commandWaitContext(ctx)
	defer cancel()
	if err := d.checkWaitPermission(github.WithLocalBudget(waitCtx), event.GetRepo(), author); err != nil {
		return false, err
	}
	if err := d.slot.Acquire(waitCtx, 1); err != nil {
		return false, err
	}

	return true, nil
}

func commandWaitContext(ctx context.Context) (context.Context, context.CancelFunc) {
	deadline, ok := ctx.Deadline()
	if !ok {
		return context.WithCancel(ctx)
	}
	remaining := time.Until(deadline)
	reserve := min(maxProcessingReserve, remaining/processingReserveDivisor)
	return context.WithDeadline(ctx, deadline.Add(-reserve.Duration()))
}

func (d *Dispatcher) checkWaitPermission(ctx context.Context, repo *github.Repository, author *github.User) error {
	if !d.permissions.tryStart() {
		return ErrCommandInProgress
	}
	defer d.permissions.finish()

	allowed, err := d.github.HasWritePermission(ctx, repo, author)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrWriteAccessRequired
	}
	return nil
}

func (d *Dispatcher) acknowledgeReceipt(ctx context.Context, event *github.IssueCommentEvent) {
	timeout := maxReceiptTimeout
	if deadline, ok := ctx.Deadline(); ok {
		timeout = min(timeout, time.Until(deadline)/receiptTimeoutDivisor)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	d.react(ctx, event, github.ReactionEyes)
}

func (d *Dispatcher) reportOutcome(ctx context.Context, event *github.IssueCommentEvent, accepted bool) {
	var reaction github.Reaction
	if accepted {
		reaction = github.ReactionThumbsUp
	} else {
		reaction = github.ReactionConfused
	}

	d.react(ctx, event, reaction)
}

func (d *Dispatcher) react(ctx context.Context, event *github.IssueCommentEvent, content github.Reaction) {
	if ctx.Err() != nil {
		return
	}

	// The service logs feedback failures; they do not change command outcomes.
	_ = d.github.AddCommentReaction(ctx, event.GetRepo(), event.GetComment(), content)
}
