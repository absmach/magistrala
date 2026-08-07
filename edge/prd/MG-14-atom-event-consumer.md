# MG-14 — Consume Atom domain events

| | |
|---|---|
| **Repo** | `absmach/magistrala` (Go) |
| **Priority** | P1 |
| **Depends on** | — |
| **Improves** | MG-07, MG-08 (both ship TTL-only without it) |
| **Status** | **⏸ Phase 2 — deferred** |

> **⏸ Deferred to phase 2** ([spec §8 A14](../architecture.md#8-decision-record)).
> Invalidates MG-08 part B's authorized-device cache. Phase 1 barely has that cache — invalidation for something not yet built.
>
> The design below is unchanged and remains the target.

## Problem

Two caches in this programme are correctness-sensitive and, without event-driven
invalidation, both rely on a TTL:

| Cache | Consequence of staleness |
|---|---|
| MG-08 authorized device set | A revoked customer keeps reading data until expiry — the TTL *is* the revocation SLA |
| MG-08 UUID → `external_id` translation | A newly-granted device stays invisible until expiry |

> An earlier draft listed a second consumer, MG-07's attachment cache. That cache
> no longer exists — [spec §8 A7](../architecture.md#8-decision-record) removed the attachment and
> withdrew MG-07. MG-08 is now the only consumer, which narrows this PRD's value
> but does not remove it: the revocation SLA is customer-visible.

Earlier drafts treated this as unavoidable, on the basis that no entity lifecycle
events exist: all Magistrala event consumers were deleted in
`16ba29cf4`, leaving only messaging events.

**That reasoning was incomplete. Atom already publishes exactly these events.**

## What Atom provides

A transactional outbox (`migrations/004_event_outbox.sql`) with an AMQP publisher
(`src/events/publisher.rs`), emitting ~40 domain events. The ones that matter
here:

| Event | Invalidates |
|---|---|
| `entity.update` | translation cache — a changed `external_id` or a newly-created device |
| `entity.create`, `entity.delete` | translation cache — a new or removed device changes the UUID ↔ serial map |
| `group_member.add`, `group_member.remove` | authorized-set cache — the sharing operation under MG-04 |
| `direct_policy.create`, `direct_policy.delete` | authorized-set cache — grant changes |
| `entity.parent_group.set`, `entity.parent_group.clear` | authorized-set cache |

The outbox design is deliberately append-only, with unconstrained
`actor_entity_id` / `tenant_id` so that failure events and post-purge history
survive — see the rationale comment at `004_event_outbox.sql:6-28`. Delivery is
at-least-once with retry and an `unparseable` flag distinguishing permanent
deserialize failures from transient broker outages.

**It is dark by default.** Publishing is a no-op unless `ATOM_EVENTS_AMQP_URL` is
set (`atom/docker-compose.yml:51`), and Magistrala's compose does not set it at
all — no `ATOM_EVENTS*` variable appears anywhere under `docker/`.

## Scope

**In scope**

- Enable Atom event publishing in Magistrala's deployment: `ATOM_EVENTS_AMQP_URL`
  and related settings in `docker/.env` and `docker/docker-compose.yaml`.
- A consumer in `pkg/atom/events` (or similar) subscribing to the Atom exchange.
- A cache-invalidation interface MG-08 registers against, and any later consumer
  can reuse.
- At-least-once handling: idempotent invalidation, which is trivially safe since
  invalidation is a delete.

**Out of scope**

- Magistrala-*emitted* entity lifecycle events. This consumes Atom's; it does not
  reintroduce the producers deleted in `16ba29cf4`.
- Replacing TTL. Event invalidation is an optimisation **on top of** a TTL, never
  a replacement — see Risks.
- Reacting to events beyond cache invalidation (audit, notifications, journal).

## Design

### Transport

Atom publishes AMQP 0.9.1. Magistrala already runs an AMQP broker — FluxMQ
exposes it (`docker/nginx/snippets/fluxmq-amqp-upstream.conf`,
`MG_NGINX_AMQP_PORT`) and services already consume from it via
`pkg/messaging/fluxmq`. Point Atom at that broker rather than introducing another.

The surviving generic event machinery in `pkg/events/` (`Subscriber`,
`SubscriberConfig`, backends for fluxmq/nats/redis) is the natural home for the
consumer, and is currently unused for anything but messaging.

### Invalidation, not synchronisation

The consumer must only **invalidate**, never populate. An event says "this fact
changed"; the next lookup re-reads from Atom. Populating caches from event
payloads reintroduces ordering and consistency problems that at-least-once
delivery does not solve.

Concretely: on `direct_policy.delete` for subject S, drop S's authorized set.
Do not attempt to remove just the affected device from the cached set by parsing
the payload — re-read it.

### Degradation

If the broker is unreachable or events stop flowing, behaviour must degrade to
today's TTL semantics — stale but bounded — not to indefinite staleness. This is
why the TTL stays.

## Acceptance criteria

1. Atom publishes events in the Magistrala compose stack; the exchange receives
   them.
2. Creating a device with a serial that already has stored data invalidates the
   translation cache, so its history becomes visible to grant-holders without
   waiting for the TTL.
3. Changing a device's `external_id` invalidates the translation cache.
4. Revoking a customer's group grant invalidates the authorized-set cache, and
   the next read reflects it.
5. Adding a device to a sharing group takes effect on the next read.
6. Duplicate delivery of the same event is harmless.
7. With the broker stopped, the caches still expire by TTL and the system stays
   correct — slower, not wrong.
8. Events for other tenants do not invalidate unrelated cache entries.

## Test plan

- Integration (Atom + broker in Docker): each event type above, asserting
  invalidation timing well inside the TTL.
- Duplicate delivery: replay the same event, assert no error and no incorrect
  state.
- Broker-down: stop the broker mid-test, assert TTL fallback still produces
  correct results (criterion 7).
- Ordering: apply `entity.update` events out of order, assert the final state is
  read from Atom rather than reconstructed from payloads.

## Risks

- **Treating events as authoritative.** At-least-once delivery with no ordering
  guarantee means payload-derived state will eventually be wrong. Invalidate
  only; the design note above is the guard.
- **Losing the TTL.** Once event invalidation works, the TTL looks redundant and
  someone will raise it to hours. Then a broker outage becomes a security
  problem — revoked customers keep reading. Keep the TTL as the correctness
  floor and document it as such.
- **Outbox lag.** `ATOM_EVENTS_OUTBOX_POLL_INTERVAL_SECS` defaults to 5s
  (`atom/docker-compose.yml:57`), so invalidation is not instantaneous. Fine
  here, but "immediately" in the acceptance criteria means seconds, not
  milliseconds.
- **New infrastructure dependency** between Atom and the broker in a deployment
  that currently has none. It is optional and degrades safely, but it is one more
  thing to configure and monitor.
