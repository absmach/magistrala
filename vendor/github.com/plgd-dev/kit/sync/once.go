package sync

import (
	"sync"
	"sync/atomic"
)

// Once performs exactly one action.
// See sync.Once for more details.
type Once struct {
	m    sync.Mutex
	done uint32
}

// Done returns true after Try executes succesfully.
func (o *Once) Done() bool {
	return atomic.LoadUint32(&o.done) == 1
}

// Try executes the function f exactly once for this instance of Once.
// If the function f returns false, it enables further execution attempts.
func (o *Once) Try(f func() bool) {
	if atomic.LoadUint32(&o.done) == 1 {
		return
	}

	o.m.Lock()
	defer o.m.Unlock()
	if o.done == 1 {
		return
	}
	if f() {
		atomic.StoreUint32(&o.done, 1)
	}
}
