package commands

import (
	"fmt"

	"github.com/alexfalkowski/abgleich/internal/github"
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/time"
	"github.com/alexfalkowski/go-sync"
)

const (
	maxReceiptTimeout = time.Second
	// Concurrent command names share two slots for webhook processing.
	concurrentCommandSlots = 2

	// Leave most of the delivery deadline for authorization and the action.
	receiptTimeoutDivisor = 4
)

type registeredCommand struct {
	handler Handler
	slots   *sync.Semaphore
}

// NewDispatcher constructs synchronous PR comment dispatch with no commands.
// Feature modules register their handlers before the application starts.
func NewDispatcher(service github.Service) *Dispatcher {
	return &Dispatcher{
		github:     service,
		commands:   make(map[string]registeredCommand),
		concurrent: sync.NewSemaphore(concurrentCommandSlots),
	}
}

// Dispatcher selects handlers by exact command name and supplies shared
// installation validation and reaction feedback. Each serial command has
// its own execution slot. Concurrent commands share a bounded set of slots.
type Dispatcher struct {
	github     github.Service
	commands   map[string]registeredCommand
	concurrent *sync.Semaphore
}

// RegisterSerial adds a handler that admits one delivery of its command name at a time.
// Register before serving deliveries. Duplicate names wrap [ErrCommandRegistered]
// and leave the existing registration intact.
func (d *Dispatcher) RegisterSerial(name string, handler Handler) error {
	return d.register(name, registeredCommand{handler: handler, slots: sync.NewSemaphore(1)})
}

// RegisterConcurrent adds a handler that draws from the shared concurrent slots.
// The handler must support overlapping deliveries of the same name. The naming
// and startup rules from [Dispatcher.RegisterSerial] apply.
func (d *Dispatcher) RegisterConcurrent(name string, handler Handler) error {
	return d.register(name, registeredCommand{handler: handler, slots: d.concurrent})
}

// Handle dispatches a verified PR comment within the caller's deadline.
// Unknown commands are ignored without feedback. When all applicable slots are
// occupied, it returns [ErrCommandInProgress] before making any GitHub calls.
func (d *Dispatcher) Handle(ctx context.Context, event *github.IssueCommentEvent) error {
	command, ok := d.commands[Parse(event.GetComment().GetBody())]
	if !ok {
		return nil
	}

	if !command.slots.TryAcquire(1) {
		return ErrCommandInProgress
	}
	defer command.slots.Release(1)

	if err := d.github.ValidateInstallation(ctx); err != nil {
		return err
	}

	d.acknowledgeReceipt(ctx, event)

	accepted, err := command.handler.Handle(ctx, event)
	d.reportOutcome(ctx, event, accepted)

	return err
}

func (d *Dispatcher) register(name string, command registeredCommand) error {
	if _, ok := d.commands[name]; ok {
		return fmt.Errorf("%w: %s", ErrCommandRegistered, name)
	}

	d.commands[name] = command

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
