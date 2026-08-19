// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"errors"
	"sync"
)

// Invalidator drops cached state keyed by key -- in this package, always a
// tenant/domain id -- rather than reading, reconstructing, or otherwise
// trusting anything about what changed. Invalidate must be safe to call for
// a key that was never cached and safe to call twice running for the same
// change: at-least-once, unordered delivery means both happen routinely,
// and a cache delete is naturally idempotent -- deleting an absent key is a
// no-op, not an error.
type Invalidator interface {
	Invalidate(ctx context.Context, key string) error
}

// Registry fans an invalidation out to every Invalidator registered for a
// family. A family is just a name -- FamilyTranslation, FamilyAuthorizedSet,
// or anything a future consumer defines -- so unrelated caches can register
// against the same family without knowing about each other, and a consumer
// backing several concerns can register a different Invalidator value
// against each one, so a family fan-out only ever touches the cache it
// actually concerns (MG-08's readAuthorizer does exactly that: two caches,
// two families, one Invalidator value each).
type Registry struct {
	mu           sync.RWMutex
	invalidators map[string][]Invalidator
}

// NewRegistry returns an empty Registry, ready to Register against.
func NewRegistry() *Registry {
	return &Registry{invalidators: make(map[string][]Invalidator)}
}

// Register adds inv to family. Meant to run during service startup wiring,
// not on a request path.
func (r *Registry) Register(family string, inv Invalidator) {
	if inv == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.invalidators[family] = append(r.invalidators[family], inv)
}

// Invalidate calls every Invalidator registered for family with key. It
// keeps going after an individual invalidator errors -- one cache's problem
// invalidating an entry must not stop a sibling cache in the same family
// from invalidating its own -- and joins every error it saw into one.
func (r *Registry) Invalidate(ctx context.Context, family, key string) error {
	r.mu.RLock()
	invalidators := append([]Invalidator(nil), r.invalidators[family]...)
	r.mu.RUnlock()

	var errs []error
	for _, inv := range invalidators {
		if err := inv.Invalidate(ctx, key); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
