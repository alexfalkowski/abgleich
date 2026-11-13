package commands

import (
	"github.com/alexfalkowski/go-service/v3/context"
	"github.com/alexfalkowski/go-service/v3/errors"
	"github.com/alexfalkowski/go-sync"
)

const maxQueuedCommands = 4

type queuedCommand struct {
	handler sync.Handler
	key     string
}

// NewQueue constructs a queue for one command's background work.
func NewQueue() *Queue {
	return &Queue{ready: make(chan struct{}, 1)}
}

// Queue holds up to four queued or running operations in memory. Callers supply
// keys for coalescing and schedule Process as one non-overlapping job.
type Queue struct {
	ready   chan struct{}
	pending []queuedCommand
	mutex   sync.Mutex
}

// Ready signals accepted work without blocking Enqueue. Signals coalesce; the
// consumer drains Process after a signal and must not close the channel.
func (q *Queue) Ready() <-chan struct{} {
	return q.ready
}

// Enqueue accepts work or coalesces it with an existing key. A full queue returns
// ErrCommandInProgress. Callers normalize keys when identity ignores case.
func (q *Queue) Enqueue(key string, handler sync.Handler) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	for _, command := range q.pending {
		if command.key == key {
			return nil
		}
	}

	if len(q.pending) == maxQueuedCommands {
		return ErrCommandInProgress
	}

	q.pending = append(q.pending, queuedCommand{key: key, handler: handler})
	q.notify()

	return nil
}

// Process runs accepted work serially, collecting failures and stopping on
// cancellation. Each attempted operation is removed, including failed work.
func (q *Queue) Process(ctx context.Context) error {
	// A job deadline may leave requests that have not been attempted. Preserve
	// their wakeup so a notified worker can give them a fresh processing budget.
	defer func() {
		q.mutex.Lock()
		defer q.mutex.Unlock()

		if len(q.pending) > 0 {
			q.notify()
		}
	}()

	var failures error
	for range maxQueuedCommands {
		if err := ctx.Err(); err != nil {
			return errors.Join(failures, err)
		}

		q.mutex.Lock()
		if len(q.pending) == 0 {
			q.mutex.Unlock()
			break
		}

		command := q.pending[0]
		q.mutex.Unlock()

		failures = errors.Join(failures, command.handler(ctx))

		q.mutex.Lock()
		q.pending[0] = queuedCommand{}
		q.pending = q.pending[1:]
		q.mutex.Unlock()

		if err := ctx.Err(); err != nil {
			return errors.Join(failures, err)
		}
	}

	return failures
}

func (q *Queue) notify() {
	select {
	case q.ready <- struct{}{}:
	default:
	}
}
