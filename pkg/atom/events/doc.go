// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package events consumes the workspace events Atom publishes from its
// transactional outbox (absmach/atom migrations/004_event_outbox.sql,
// src/events/publisher.rs) and turns them into cache invalidation, nothing
// else (MG-14). Delivery is at-least-once with no ordering guarantee, so
// this package only ever invalidates -- it never reconstructs or patches
// cached state from an event's payload. The next cache lookup after an
// invalidation re-reads fresh from Atom, which is the only state a consumer
// here is allowed to trust.
//
// Two things live here: the wire-to-family mapping (which Atom event names
// matter, and which cache family each belongs to -- event.go) and a small
// Registry that families dispatch through (registry.go). The registry
// doesn't know or care what "translation" or "authorized_set" cache MG-08
// actually built -- see readers/api/http/readauthz.go for the concrete
// Invalidator implementation and how it registers.
//
// Event invalidation is an optimization layered on top of the existing
// cache TTLs, never a replacement for them: if the broker is unreachable, or
// this consumer is never wired up at all, the caches it would have
// invalidated keep working exactly as they did before -- bounded by their
// TTL alone, just slower to reflect a change.
package events
