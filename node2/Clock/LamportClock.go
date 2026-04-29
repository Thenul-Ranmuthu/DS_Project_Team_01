package clock

import (
	"fmt"
	"sync"
)

type LamportClock struct {
	mu    sync.Mutex
	value uint64
}

// Create and return a new LamportClock starting at 0.
func New() *LamportClock {
	return &LamportClock{}
}

// Increment the clock by 1 for a local event (ex: an upload request).
func (lc *LamportClock) Tick() uint64 {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.value++

	return lc.value // Returns the new clock value.
}

// Update the clock when a message is received from another node.
// Rule: value = max(local, received) + 1
func (lc *LamportClock) Sync(received uint64) uint64 {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	if received > lc.value {
		lc.value = received
	}
	lc.value++
	return lc.value // Returns the new clock value.
}

// Sync the clock with a received value without incrementing.
// Used for replication where we sync to the same event, not a new event.
// Rule: value = max(local, received) [no increment]
func (lc *LamportClock) SyncValue(received uint64) uint64 {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	if received > lc.value {
		lc.value = received
	}
	return lc.value // Returns the current clock value without incrementing.
}

func (lc *LamportClock) Value() uint64 {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	return lc.value // Returns the current clock value without modifying it.
}

func (lc *LamportClock) String() string {
	return fmt.Sprintf("LamportClock(%d)", lc.Value()) // Returns a human-readable representation of the clock.
}
