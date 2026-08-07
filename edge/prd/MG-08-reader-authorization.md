# MG-08 — Reader authorization: enforce per-device access

| | |
|---|---|
| **Repo** | `absmach/magistrala` (Go) |
| **Priority** | P1 (phase 1 half) |
| **Depends on** | MG-01 (phase 1) · MG-06, ATOM-02, ATOM-06 (phase 2) |
| **Blocks** | — |
| **Status** | Draft |

> **This PRD splits across phases** ([spec §8 A14](../architecture.md#8-decision-record)).
>
> | | Scope | Phase |
> |---|---|---|
> | **A** | Enforce the *existing* `publishers` filter against authorization | **1 — now** |
> | **B** | Add device-level filtering and the UUID → `external_id` translation | 2 |
>
> **Part A is a live defect on `main` and needs none of the device work.** Today
> the `publishers` query filter is applied with no authorization check at all, so
> any user who can read a channel reads every publisher on it by changing a
> parameter. Fixing that requires only MG-01.
>
> Part B waits on MG-06. Acceptance criteria below are marked accordingly.
>
> Part A needs **only MG-01**. ATOM-02 was its scale story, and that belongs to
> part B — a handful of publishers is fine with a per-publisher `authzCheck`.

## Problem

Reader authorization is not a security boundary.

`readers/api/http/transport.go:251-266` authorizes `subscribe` on the **channel**,
then applies the caller-supplied `publisher` / `publishers` query filters without
validating them (`readers/messages.go:49-50`, applied at `transport.go:175`).

**Any user who can read a channel can read every publisher's messages on it by
changing a query parameter.** The filter is a convenience, not a control.

This is a live issue today, independent of the device work — the `publisher`
filter has always behaved this way. MG-06 adds `device_ids` with the same
property. This PRD makes both boundaries.

Without it, the customer requirement — *"customer A sees meters 1 and 3, not
meter 2"* — cannot be satisfied, because meter 2's data is one query parameter
away.

## Scope

**In scope**

- After the existing channel check, resolve the caller's authorized device set
  and intersect it with the requested filter.
- Same treatment for `publishers`.
- Wire `PolicyService` into the reader binaries — it is `nil` everywhere today
  (`cmd/auth/main.go:182`; readers construct only `channels` at
  `cmd/timescale-reader/main.go:109`, `cmd/postgres-reader/main.go:109`).
- Bypass for domain admins, who legitimately read the whole channel.
- Cache the resolved set per `(subject, domain)` with a short TTL.

**Out of scope**

- Changing the channel-level check. It stays; device scoping narrows within it.
- Per-message ACLs.
- Subscribe-side (live MQTT) device scoping — different path, different PRD.

## Design

### Resolution

Use `authorizedObjectIds` via `PolicyService.ListAllObjects`
(`pkg/atom/policy_service.go:128-156`):

```
subjectID:  <caller>
action:     "read"
objectKind: "entity"
objectType: "entity:device"      // namespaced — see MG-01
tenantID:   <domain>
```

Requires MG-01 (the `objectType` fix and the `isSupportedObjectList` widening) or
this returns nothing.

### The translation step — easy to miss, and it fails silently

`authorizedObjectIds` returns Atom entity **UUIDs**. Messages carry `device_id`
as the device's **external serial string** ([spec §8 A8](../architecture.md#8-decision-record)),
because the publish path performs no lookup. The authorized set must therefore be
mapped UUID → `external_id` before it can filter anything.

Skipping this produces a filter that matches no rows and presents as a
permissions bug — the caller sees an empty result and concludes they have no
access. Requires ATOM-06.

The mapping is one indexed query over a small set, cacheable alongside the
authorized set itself.

### Orphan data

Rows whose `device_id` has no device entity cannot be granted to anyone — there
is no object to grant. They stay readable through channel-level access only,
which is the correct default: unregistered data is visible to operators and never
to customers. No special handling is needed; it falls out of the intersection.

### Intersection rules

| Caller supplied | Behaviour |
|---|---|
| Nothing | Filter by the full authorized set |
| A subset of the authorized set | Use it as given |
| IDs outside the authorized set | Silently drop them — return the intersection, not an error |
| Only unauthorized IDs | Empty result |

Dropping rather than erroring avoids leaking which device IDs exist.

**The empty set must mean "no rows", never "no filter".** This is the sharpest
failure mode in the PRD: a subject with zero authorized devices must get zero
rows. MG-06 flags that `omitempty` erases an empty slice from the query — so
either use a pointer type or short-circuit before building the query. Confirm
which, and test it directly.

### Scale

Materialising every authorized ID into `= ANY(...)` degrades on large fleets.
Two mitigations, in order:

1. **Prefer server-side narrowing.** ATOM-02 exposes `parentGroupId`,
   `includeDescendants` and `attributesContains` on `authorizedObjectIds`, so the
   set can be narrowed in Atom rather than materialised in Go.
2. Cache per `(subject, domain)` with a short TTL. Group-scoped grants (MG-04)
   keep the practical set small.

If fleets outgrow both, the filter has to move into Atom entirely. Worth knowing
before it bites rather than after.

### Admin bypass

Per [spec §8 B3](../architecture.md#8-decision-record): determine "may read all devices in this
domain" from an **explicit tenant-scoped capability check**.

Not by string-matching a role name, and specifically **not** by treating an empty
authorized set as "unrestricted" — two opposite situations produce an identical
empty list:

| Caller | Per-device grants | Should see |
|---|---|---|
| Domain admin | none — holds a *tenant-wide* grant | everything |
| User with no access at all | none | nothing |

Reading empty as unrestricted gives both everything, so the caller with the
fewest permissions receives the most data. Asking the capability question
directly makes empty unambiguously mean "no access".

## Acceptance criteria

**Phase 1 (part A)**

A1. A user without a grant covering publisher X receives no rows for X, whatever
    `publishers` they request. This fails on today's `main` — it is the defect.
A2. A user requesting `publishers` they *are* entitled to gets exactly those.
A3. A domain admin is unaffected.
A4. A user with no publisher grants receives empty, **not everything** — the
    empty-set inversion is the failure mode that matters.

**Phase 2 (part B)**

1. A user granted `read` on meters 1 and 3, querying a channel carrying all
   three, receives data for 1 and 3 only.
2. The same user requesting `device_ids=2` receives **empty**, not meter 2's data.
3. Requesting `device_ids=1,2` returns meter 1 only.
4. A user with **no** device grants receives empty, not everything.
5. A domain admin receives all three.
6. Equivalent behaviour for `publishers`.
7. Unauthorized IDs are dropped silently — the response does not reveal whether
   they exist.
8. Behaviour is identical across HTTP, gRPC and SDK.
9. The channel-level check still rejects users without `subscribe`.
10. A customer granted a device sees its data, proving the UUID → `external_id`
    translation works end to end. This is the criterion that catches a missing
    translation, which otherwise looks like an authorization failure.
11. Orphan data — `device_id` with no entity — is visible to a channel-level
    reader and invisible to a device-scoped customer.
12. **Revoking a customer's grant ends their access within the stated TTL**, and
    immediately if MG-14 is present. The TTL *is* the revocation SLA, so it needs
    a criterion rather than living only in the test plan.

## Test plan

- **Regression test first.** Criteria 1–4 written against current `main` must
  fail. If they pass, the test is wrong.
- Integration (`ory/dockertest` + Atom): the full matrix, both backends.
- Cache: grant, query, revoke, query again — access ends within the stated TTL.
- Admin bypass: explicitly assert it is capability-driven by testing a
  non-admin with a large grant set and an admin with none.
- Performance: query latency with an authorized set of 1, 100 and 10 000 devices.

## Risks

- **Empty-set inversion** — treating "no authorized devices" as "no filter" turns
  this control into a full disclosure. Criterion 4 is the guard and must be an
  explicit test, not incidental coverage.
- **Stale cache after revocation** leaves a window where a revoked customer still
  reads data. Atom publishes `direct_policy.delete`, `group_member.remove` and
  `entity.parent_group.clear` (`src/events/publisher.rs`), so
  [MG-14](./MG-14-atom-event-consumer.md) makes invalidation deterministic rather
  than TTL-bounded. Keep the TTL as the correctness floor: with the broker down,
  the revocation SLA falls back to it, so it must stay short enough to be
  defensible on its own.
- **Existing deployments may rely on the current permissive behaviour.** A user
  who reads a channel today and sees everything will see less. That is the point,
  but it is a behavioural break and belongs in release notes.
- Depends on ATOM-02 for the scalable path. Without it, ship the materialised
  version and accept the ceiling — but do not silently skip enforcement.
