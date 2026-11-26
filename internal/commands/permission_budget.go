package commands

import (
	"github.com/alexfalkowski/go-service/v3/time"
	"github.com/alexfalkowski/go-sync"
)

const (
	permissionCheckWindow      = 10 * time.Second
	permissionChecksPerWindow  = 4
	concurrentPermissionChecks = 2
)

// permissionBudget bounds GitHub permission checks for busy-slot admission.
// Each check occupies a rate slot for the full window, even after it returns.
type permissionBudget struct {
	available [permissionChecksPerWindow]time.Time
	mutex     sync.Mutex
	next      int
	active    int
}

func (b *permissionBudget) tryStart() bool {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	now := time.Now()
	if b.active >= concurrentPermissionChecks || now.Before(b.available[b.next]) {
		return false
	}

	b.available[b.next] = now.Add(permissionCheckWindow.Duration())
	b.next = (b.next + 1) % len(b.available)
	b.active++
	return true
}

func (b *permissionBudget) finish() {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.active--
}
