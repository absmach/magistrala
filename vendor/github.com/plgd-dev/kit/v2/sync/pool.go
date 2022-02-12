package sync

import (
	"context"
	"fmt"
	"sync"
)

// Pool is a synchronized key-value store with customizable factory for missing items.
type Pool struct {
	mtx    sync.Mutex
	store  map[string]interface{}
	create PoolFunc
}

// PoolFunc is triggered on a miss by GetOrCreate,
// so that it may add the missing item to the pool.
type PoolFunc func(ctx context.Context, key string) (interface{}, error)

// NewPool creates a pool.
func NewPool() *Pool {
	return &Pool{store: make(map[string]interface{})}
}

// SetFactory sets the pool factory for GetOrCreate.
func (p *Pool) SetFactory(f PoolFunc) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.create = f
}

// Put adds an item to the pool.
func (p *Pool) Put(key string, item interface{}) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.store[key] = item
}

// Get returns an item from the pool or false.
func (p *Pool) Get(key string) (_ interface{}, ok bool) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	item, ok := p.store[key]
	return item, ok
}

// Delete pops an item from the pool or false.
func (p *Pool) Delete(key string) (_ interface{}, ok bool) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	item, ok := p.store[key]
	delete(p.store, key)

	return item, ok
}

// GetOrCreate returns an item and calls the factory on a mis.
// Warning: The factory function is called under the lock,
// therefore it must not call any Pool's methods to avoid deadlocks.
func (p *Pool) GetOrCreate(ctx context.Context, key string) (interface{}, error) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	if item, ok := p.store[key]; ok {
		return item, nil
	}

	if p.create == nil {
		return nil, fmt.Errorf("missing pool factory")
	}
	item, err := p.create(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("could not create pool item %s: %w", key, err)
	}
	p.store[key] = item
	return item, nil
}
