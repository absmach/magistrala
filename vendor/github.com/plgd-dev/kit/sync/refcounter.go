package sync

import (
	"context"
	"sync/atomic"
)

type ReleaseDataFunc = func(ctx context.Context, data interface{}) error

type RefCounter struct {
	count           int64
	data            interface{}
	releaseDataFunc ReleaseDataFunc
}

// Data returns data
func (r *RefCounter) Data() interface{} {
	v := atomic.LoadInt64(&r.count)
	if v <= 0 {
		panic("using RefCounter after data released")
	}
	return r.data
}

// Acquire increments counter
func (r *RefCounter) Acquire() {
	v := atomic.AddInt64(&r.count, 1)
	if v <= 1 {
		panic("using RefCounter after data released")
	}
}

// Count returns current counter value.
func (r *RefCounter) Count() int64 {
	return atomic.LoadInt64(&r.count)
}

// Release decrements counter, when counter reach 0, releaseDataFunc will be called
func (r *RefCounter) Release(ctx context.Context) error {
	v := atomic.AddInt64(&r.count, -1)
	if v < 0 {
		panic("using RefCounter after data released")
	}
	if v == 0 {
		if r.releaseDataFunc != nil {
			return r.releaseDataFunc(ctx, r.data)
		}
	}
	return nil
}

// NewRefCounter creates RefCounter
func NewRefCounter(data interface{}, releaseDataFunc ReleaseDataFunc) *RefCounter {
	return &RefCounter{
		data:            data,
		count:           1,
		releaseDataFunc: releaseDataFunc,
	}
}
