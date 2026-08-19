# MG-13 — Gateway-announced device discovery

| | |
|---|---|
| **Repo** | `absmach/magistrala` (Go) |
| **Priority** | P5 |
| **Depends on** | MG-12 (MG-07 withdrawn — see banner) |
| **Blocks** | — |
| **Status** | **⏸ Phase 2 — deferred** |

> **Predates decisions A7 and A8 and is not currently in scope.** It still assumes
> a `gateway_id` attachment and a pending-state ingest check, both of which the
> model has since dropped. Retained because the *scenarios* it captures remain
> valid. Revise before picking it up.

> **⏸ Deferred to phase 2** ([spec §8 A14](../architecture.md#8-decision-record)).
> Predates A7/A8 and needs revising; discovery is phase 2 at the earliest.
>
> The design below is unchanged and remains the target — this is a scope call,
> not a reversal.

## Problem

Cloud-first provisioning requires an operator to create every meter record before
installation — serial typed in by hand, per device. For a utility rolling out
thousands of meters that is the dominant cost of deployment and the main source
of data-entry error.

The gateway already discovers meters on its local bus. It knows their serials. It
should be able to say so.

Target flow:

```
1. Installer powers the meter on site
2. Gateway discovers it on BLE/Modbus — serial ABC123
3. Gateway announces it
4. Device appears as pending, attached to that gateway
5. Operator approves: assigns device type and customer group
6. Data flows, attributed to the meter
```

The installer needs no console and no credentials.

## Scope

**In scope**

- An authenticated announce endpoint for gateways.
- Pending device lifecycle: `pending` → `provisioned` / `rejected`.
- Operator approval, including bulk.
- Defined handling of data arriving from a pending device.

**Out of scope**

- Local discovery itself — agent-side.
- Auto-approval. Every announced device requires an explicit decision (see
  Design).
- Auto device-type inference from serial patterns. Tempting, deferred.

## Design

### Authentication

The gateway already has an authenticated channel: PR #3555's challenge/response
gives proof of possession of the enrollment key without a device clock
(`bootstrap/device_bootstrap.go`). Reuse it. Announce is a Bootstrap-side
endpoint authenticated the same way.

The alternative — announcing over MQTT on a reserved topic — avoids a second
protocol but puts entity creation on the message path, where there is no
request/response and no useful error reporting. **Recommend the Bootstrap
endpoint.**

### Announce

```
POST /devices/announce/{externalID}
{ "devices": [ { "serial": "ABC123",
                 "hints": { "manufacturer": "…", "model": "…" } } ] }
```

**Revised by [spec §8 A13](../architecture.md#8-decision-record):** the announce
payload *should* carry the discovered address, since the cloud now holds it. Where
the agent can read a serial from a holding register — the device type says which
one — announce becomes the preferred way to populate the edge, turning "operator
types 500 mappings" into "operator confirms 500 discovered mappings".

Semantics:

- Unknown serial → create Device, `provisioning_state: pending`
  (a device attribute, per [spec §8 A2](../architecture.md#8-decision-record)),
  `gateway_id` = announcing gateway.
- Known serial, same gateway → update hints, no state change.
- Known serial, **different** gateway → this is a re-homing claim, not a
  discovery. Do not silently re-home; flag for operator confirmation. Silent
  re-homing would let any gateway steal any meter by announcing its serial.
- Idempotent: re-announcing an existing set is a no-op.

### Pending devices and their data

A pending device exists but is not approved. Its data must not silently enter the
system as though provisioned — that would make approval meaningless.

MG-07 denies publishes for devices not attached to the publisher. A pending
device **is** attached, so it would otherwise pass and approval would mean
nothing.

**DECIDED ([spec §8 B1](../architecture.md#8-decision-record)): reject.** MG-07 gains a clause
denying any device whose `provisioning_state` is not `provisioned`.

Consequences that belong to this PRD:

- **Readings between installation and approval are lost.** The approval window is
  therefore an operational parameter, not a UX detail — every unapproved minute
  costs data.
- Bulk approval (below) matters more under this decision than it would under
  quarantine.
- Approval must invalidate MG-07's attachment cache, or an approved device stays
  unable to publish until the TTL expires.
- Operators need visibility into how long devices have been pending, since that
  duration is data loss.

Revisit only if the window turns out to be days rather than hours; quarantine
would then need a retention and access-control policy for data belonging to
devices that may ultimately be rejected.

### Approval

```
POST /{domainID}/devices/{id}/approve   { device_type_id, group_id }
POST /{domainID}/devices/{id}/reject
POST /{domainID}/devices/approve        (bulk)
```

Approval assigns the device type and optionally the sharing group — the two
things that make the device useful. Bulk matters: a gateway announcing 200 meters
should not require 200 operator actions.

Rejection should be sticky, so a rejected serial is not re-created on the next
announce cycle.

### Rate limiting

A misbehaving or compromised gateway can announce unbounded serials, each
creating an entity. Bound announcements per gateway per interval, and cap pending
devices per gateway. Without this, announce is an entity-creation amplifier
reachable with one gateway credential.

## Acceptance criteria

1. A gateway announcing an unknown serial creates a pending device attached to it.
2. Re-announcing the same serial is idempotent.
3. Announcing a serial owned by another gateway does **not** re-home it silently;
   it is flagged.
4. A pending device's data is handled per the decided rule, with a test asserting
   exactly that behaviour.
5. Approval assigns type and group, sets `provisioned`, and data flows.
6. Rejection marks the device; the next announce does not recreate it.
7. Bulk approval handles 200 devices in one call.
8. An unauthenticated announce is refused.
9. Exceeding the announce rate limit or pending cap is refused with a clear error.
10. Cloud-first provisioning still works unchanged — both paths coexist.

## Test plan

- Integration: full flow — announce, list pending, approve, publish, read back
  attributed to the meter.
- Criterion 3 explicitly: two gateways, one serial. This is the security-relevant
  case.
- Idempotency: announce the same set five times, assert one device.
- Rate limiting and pending cap.
- Rejection stickiness across announce cycles.
- Interaction with MG-07 for criterion 4.

## Risks

- **Serial uniqueness is per tenant** ([spec §8 C4](../architecture.md#8-decision-record)) — two
  domains may each hold a meter with serial `ABC123`. The announce path must
  resolve serials **within the announcing gateway's tenant**; a lookup that
  forgets the tenant filter cross-attributes data between customers and fails
  silently. Needs an explicit test, not care.
- **Announce as an amplification vector** — one gateway credential creating
  unbounded entities. Rate limiting is a requirement, not a hardening extra.
- **Operator burden at scale.** 200 pending devices with no filtering or grouping
  in the UI is unusable, and the feature's value evaporates. Bulk approval is
  necessary but probably not sufficient; the UI needs a view designed for this.
- **The pending-data decision couples to MG-07.** Deciding it here without
  changing MG-07 leaves the two inconsistent.
