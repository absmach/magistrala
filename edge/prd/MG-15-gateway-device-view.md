# MG-15 — Gateway device view

| | |
|---|---|
| **Repo** | `absmach/magistrala` (Go) |
| **Priority** | P4 — gated by MG-09, which is P4 |
| **Depends on** | MG-06, MG-08, MG-09, ATOM-01 |
| **Blocks** | — |
| **Status** | **⏸ Phase 2 — deferred** |

> **⏸ Deferred to phase 2** ([spec §8 A14](../architecture.md#8-decision-record)).
> The observed half needs `device_id` in storage. The **declared** half is UI composition over ATOM-01 and needs no PRD.
>
> The design below is unchanged and remains the target — this is a scope call,
> not a reversal.

## Problem

An operator clicks a gateway and needs to see the devices reachable through it —
both what was *commissioned* onto it and what has *actually* published through it.

Neither half exists today. `publisher` is already a reader filter
(`readers/messages.go:49`), so "all messages from gateway G" works, but there is
no distinct-device aggregation and nothing merges it with the declared relation.

**The merge is the point.** Either list alone is misleading:

| Declared | Observed | Status | Why it matters |
|---|---|---|---|
| ✓ | ✓ | **Healthy** | |
| ✓ | ✗ | **Silent** | Commissioned, never heard. For a wired link this is the fault condition — and observed-only cannot see it |
| ✗ | ✓ | **Undeclared** | Undocumented device, or a neighbour's broadcast — and declared-only cannot see it |

See [spec §2.4](../architecture.md#24-declared-and-observed) and
[§3.7](../architecture.md#37-the-gateway-view).

## Scope

**In scope**

- Distinct `device_id` values for a given publisher, with last-seen timestamp and
  message count — the **observed** half.
- The **merge** with the declared relation from MG-09, yielding a per-device
  status of healthy / silent / undeclared.
- The inverse: distinct `publisher` values for a given `device_id` — which
  gateways have relayed for this meter — alongside its declared counterpart.
- Time-bounded: "devices seen in the last 24h" is the operationally useful form.
- Exposed over HTTP, gRPC and SDK, subject to the same authorization as any other
  read.

**Out of scope**

- Enriching the roster with device entity data (name, type, status). The reader
  has no access to Atom; joining belongs in the UI backend or a composing layer,
  which already talks to both.
- Live presence. This is "what has been published", not "what is connected".
- Any stored gateway↔device relation. The whole point is that it is derived.

## Design

### Shape

```
GET /{workspaceID}/gateways/{id}/devices?from=…&to=…
  → [ { serial, status, last_seen, message_count, device_id? }, … ]
       status ∈ healthy | silent | undeclared

GET /{workspaceID}/devices/{id}/gateways
  → [ { gateway_id, declared, last_seen, message_count }, … ]
```

`device_id` is present only where a device record exists — an *undeclared* row
may be an orphan serial with no entity behind it.

Both are aggregations over the same table the readers already query, so they
belong in `readers/` beside `ReadAll` rather than in a new service.

### The query

```sql
SELECT device_id, MAX(time) AS last_seen, COUNT(*) AS message_count
FROM messages
WHERE channel = :channel AND publisher = :publisher
  AND time >= :from AND time < :to
GROUP BY device_id
```

MG-06 adds the `(channel, device_id, name, time DESC)` index; this query wants
`(channel, publisher, device_id)` ordering to avoid a full scan of the channel's
partition. **Measure before assuming the existing indexes cover it** — a
`GROUP BY` over a hypertable without a supporting index is the kind of query that
looks fine on test data and melts on a year of production.

### Authorization

Same boundary as any other read: channel-level check, then — for non-admin
callers — narrowed to the caller's authorized device set (MG-08). A customer
querying a gateway's roster must see only *their* devices on it, not the
gateway's full fleet.

This means MG-15 inherits MG-08's UUID → `external_id` translation. Building it
before MG-08 lands would ship a roster endpoint that leaks the full device list
of every gateway to anyone with channel read.

### Where the merge happens

The declared side comes from Atom (via MG-09's `GatewayDevices`); the observed
side from the message store. **The reader cannot reach Atom**, so the join
belongs in the composing layer — the UI backend, or a thin endpoint that already
holds both clients. Readers expose the observed aggregation; they do not learn
about entities.

### Orphan devices are included

A `device_id` with no device entity appears in the roster, because the roster is
built from traffic. That is a feature — it is how an operator discovers devices
worth registering — but it means the response contains identifiers the entity
store knows nothing about. Consumers must not assume every `device_id` resolves.

## Acceptance criteria

1. A gateway publishing for three devices yields exactly those three, with
   accurate `last_seen` and counts.
1a. A device declared on the gateway but never heard appears as **silent**; one
   heard but not declared appears as **undeclared**; one both declared and heard
   appears as **healthy**.
2. A device published by two gateways appears in both gateways' rosters, and the
   inverse query returns both publishers.
3. Time bounds narrow correctly; a device silent in the window is absent.
4. Orphan `device_id`s — no matching entity — appear.
5. A non-admin caller sees only devices they are authorized for, on both queries.
6. A caller without channel access is refused.
7. Consistent across HTTP, gRPC and SDK.
8. Query plan uses an index; runtime is bounded on a realistically sized table.

## Test plan

- Integration (`ory/dockertest`, both backends): write messages from two
  gateways with overlapping device sets, assert both directions.
- Authorization: customer with two of a gateway's five devices sees two
  (criterion 5) — this fails without MG-08 and is the reason for the dependency.
- Orphan inclusion.
- Performance: roster over a channel with 10k devices and a year of data, with
  the plan captured.

## Risks

- **Shipping before MG-08 leaks fleet composition.** The roster reveals every
  `device_id` a gateway serves. Without the authorized-set narrowing, any user
  with channel read learns the full device list — including other customers'
  meters. Sequence accordingly.
- **Aggregation cost.** `GROUP BY device_id` over a large hypertable partition is
  the expensive part. Consider a continuous aggregate if the live query does not
  hold up; that is a real possibility, not a remote one.
- **Cardinality.** A channel with 100k devices returns a 100k-row roster. Needs
  pagination from the start, and a default time bound so the unbounded form is
  not the easy one to call.
