# MG-09 — Device and Gateway model, SDK and protos

| | |
|---|---|
| **Repo** | `absmach/magistrala` (Go) |
| **Priority** | P4 |
| **Depends on** | MG-03, ATOM-01, ATOM-06 |
| **Blocks** | MG-11, UI-01 · MG-10, MG-12, MG-15 (phase 2) |
| **Status** | Draft |

## Problem

`Client` fuses connectivity identity with data-producing identity. Everything in
P1–P3 works around that fusion internally; this PRD makes the split visible in
the API.

**This is not a rename for readability.** `Device` differs from `Client` in
substance: credentials are optional, it carries a type and a serial, and it can
be attached to a gateway. If the only change were the label, it would not be
worth the churn.

Widest blast radius in the programme — this freezes the public API shape.

## Scope

**In scope**

- `Device` replacing `Client` in `pkg/sdk`, with the new fields.
- `Gateway` as a first-class API surface over the same underlying entity.
- The device→gateways reachability relation.
- Proto rename and reshaping.
- Clean break: no `Client` type, no `/clients` route, no alias.

**Out of scope**

- Device types — MG-10.
- CLI, PAT scopes, permissions, OpenAPI — MG-11.
- Provisioning flows — MG-12/13 (phase 2).
- Channel/connection model, which is unchanged.

## Design

### Model

`Device` gains, over today's `Client` (`pkg/sdk/clients.go:28`):

| Field | Meaning |
|---|---|
| `Serial` | **The device identifier.** Arbitrary string, Atom `external_id`, unique per workspace (ATOM-06). Appears verbatim in publish topics. Independent of Bootstrap `ExternalID` (C3) |
| ~~`DeviceTypeID`~~ | ⏸ phase 2 — device types are deferred (MG-02) |
| `ProvisioningState` | `pending` / `provisioned` / `rejected` — an **attribute**, per spec §8 A2. Lifecycle only; nothing on the publish path reads it |

**No scalar `GatewayID`.** One device can be reached through several gateways, so
the link is a **list** — `Gateways []string`, 0..N
([spec §8 A10](../architecture.md#8-decision-record)). A gateway's device list is
the reverse of it, one `attributesContains` query (ATOM-01).

`Credentials` becomes **optional** — the single property that makes one model
serve both a BLE meter and an NB-IoT meter.

### Gateway

A Gateway is **a Device with `IsGateway` set** — a capability, not a type
([spec §8 A12](../architecture.md#8-decision-record)). Every device, gateway or
not, is Atom `entity_kind: device`.

```go
type Device struct {
    ...
    IsGateway bool     `json:"is_gateway,omitempty"`
    Gateways  []string `json:"gateways,omitempty"`   // phase 1: gateway IDs
}

// ⏸ Phase 2 promotes each entry to carry an opaque bus address
// (spec §8 A13) — present only for bus-addressed protocols such as
// Modbus, and stored, rendered and compared but never parsed:
//
//   type GatewayLink struct {
//       ID      string
//       Address map[string]any
//   }
```

**The capability composes.** A device may be a gateway *and* report its own
measurements — a concentrator-meter is one device with one type doing both jobs.
Nothing may treat gateways as a disjoint population.

`/gateways` remains a first-class API surface; it lists devices where
`is_gateway` is true, via `attributesContains` (ATOM-01).

Because gateways stay `entity_kind: device`, the existing
`{entity_kind: device, publish, resource:channel}` guardrail
(`pkg/atom/bootstrap.go:72-87`) keeps applying — no new assignment rules, and
none of the silent publish-permission breakage a separate kind would have caused.

### The reachability relation

A device declares which gateways it is reachable through
([spec §2.3](../architecture.md#23-the-relation-is-a-relation--not-a-group)).
The relation is a property of the **device**, 0..N, and is *not* containment —
see [spec §2.1](../architecture.md#21-a-gateway-is-a-path-not-a-container).

```go
type Device struct {
    ...
    Gateways []string `json:"gateways,omitempty"`   // 0..N gateway IDs
}

SetDeviceGateways(ctx, deviceID, gatewayIDs []string, workspaceID, token) error
DeviceGateways(ctx, deviceID, workspaceID, token) ([]Gateway, error)
GatewayDevices(ctx, gatewayID, pm, workspaceID, token) (DevicesPage, error)
```

`SetDeviceGateways` replaces the whole list rather than offering attach/detach —
the list is short, and replace-semantics make the full-list write explicit.

**It needs optimistic concurrency, and Atom does not provide it.**
`update_entity` (`src/identity/repo.rs:335-341`) is `COALESCE($n, col)`: last
write wins, no version check. Two operators commissioning the same device
concurrently will silently lose one edit. Until Atom offers an `If-Match`, this
PRD must either serialise the update Magistrala-side or document last-write-wins
explicitly. **Do not leave it unstated** — the failure is silent.

`GatewayDevices` is the reverse lookup: devices whose `gateways` array contains
this gateway. Needs **ATOM-01** array containment; without it the only fallback
is fetching every device in the workspace, which paginates incorrectly.

This is the **declared** relation, and in phase 1 it is the whole of the gateway
view: `GatewayDevices` plus an entity read is everything the UI needs. The
*observed* half — what a gateway actually relayed for — needs `device_id` in
storage and is deferred with [MG-15](./MG-15-gateway-device-view.md).

**Why it exists beyond the UI.** The declared relation is the authoritative
source for **generating gateway config**. Self-identifying protocols need no
mapping — a wM-Bus telegram carries its serial, which flows through to the topic
and to `external_id` untouched. Bus-addressed protocols do: a Modbus unit ID is
an address, not a serial, and the gateway cannot derive one from the other. That
mapping lives in agent config, and generating it needs the authoritative serial
list for a gateway — which is exactly this query.

### ⏸ Phase 2 — Address conflicts

*Deferred with the address itself. Retained because the reasoning holds.*

#### Address conflicts — decide where this lives

Two devices declaring the **same bus address on the same gateway** is a
commissioning error: the agent would poll one unit and attribute it to two
devices. It is detectable without parsing the blob — byte-equality via containment
(`spec §3.3`) — so the "never interpret" rule survives either way.

Unassigned as yet:

- **Reject at write time**, here. Catches it at the point of the mistake, but
  needs a containment query on every `SetDeviceGateways`.
- **Surface as a warning** in the gateway view (MG-15). Cheaper, and tolerates the
  transient state during a bulk re-commission.

Recommend rejecting here, on the grounds that a silent mis-attribution is worse
than a rejected write. Decide before build; do not leave it to whoever notices.

### Deletion must not cascade

Per [spec §2.2](../architecture.md#22-what-follows-from-that--normative),
consequence 2: **deleting a gateway never deletes devices.** It is a path, not a
container. Deletion leaves stale IDs in the devices that named it; they are
resolved and dropped on read, and optionally swept.

Equally, deleting a *device* must not touch its gateways.

### Serial validation — none

`Serial` is an arbitrary string. **Magistrala imposes no format constraint**
([spec §8 A14](../architecture.md#8-decision-record)): no `/` rejection, no
length rule, no normalisation. Atom stores it unconstrained (ATOM-06) and
enforces per-tenant uniqueness; that is the whole of it.

An earlier draft rejected `/` to protect the topic grammar. Phase 1 introduces no
topic grammar, so there is nothing to protect — an application putting a serial
in a topic owns its own encoding.

Still open before this API freezes: case sensitivity and whitespace trimming for
uniqueness (ATOM-06), and whether `Serial` may change after creation. Mutability
is cheap in phase 1 and becomes expensive in phase 2, when the serial is
denormalised onto every message row.

### `provisioning_state`

Atom's `entities.status` is constrained to `active/inactive/suspended`
(`001_initial.sql:96`), so this lives in attributes — confirmed by
[spec §8 A2](../architecture.md#8-decision-record).

It is **not** Bootstrap enrollment state and must not be presented as a synonym:
sensors never enroll, and MG-07 reads this on the publish hot path where a
Bootstrap call does not belong. Atom's `status` means something different and
stays untouched.

### Surfaces

- SDK: `pkg/sdk/clients.go` → `devices.go`; new `gateways.go`. Interface block at
  `pkg/sdk/sdk.go:712-840`.
- Protos: `internal/proto/devices/v1/clients.proto` → `devices/v1/devices.proto`;
  `DevicesService` → `DevicesService`. Regenerate into `api/grpc/devices/`.
- `internal/proto/common/v1/common.proto:52` `Connection` — assess whether it
  needs a device field or whether channel connections stay gateway-level.
- `pkg/atom/mapping.go:6-12`: `KindDevice` → `KindDevice`. No gateway kind.
- `is_gateway` and `gateways` carried as entity attributes.

## Acceptance criteria

1. Create a device with no credentials; it persists and is retrievable.
2. Create a device with credentials; it can authenticate and publish.
3. Create a gateway; it holds credentials and is listed under `/gateways`. It is
   stored as Atom `entity_kind: device` with `is_gateway` set.
3a. A gateway can publish and subscribe on a connected channel, with **no new
   assignment rules** — the existing device guardrail covers it.
3b. **A device can be both**: one device with `is_gateway` set that also reports
   its own measurements appears in `/gateways` *and* `/devices`, and both its own
   readings and its relayed traffic are attributed correctly.
4. A device can declare 0, 1 and 3 gateways in turn; `DeviceGateways` and
   `GatewayDevices` agree in both directions each time.
4a. **Deleting a gateway leaves its devices intact**, with the stale reference
   dropped on read. Deleting a device leaves its gateways intact.
5. A device's data published by two different gateways is attributed to that one
   device — the model admits many gateways per device.
6. Group membership and grants are unaffected by which gateway published a
   device's data, and by changes to the reachability relation.
6a. Access to a gateway grants **no** access to the devices reachable through it,
   and vice versa (spec §2.2, consequence 3).
7. `Serial` is queryable and unique **within a workspace**; two workspaces may each
   hold the same serial, and a lookup in one never returns the other's device.
7a. A serial containing `/`, spaces or unicode is **accepted** — no format
   validation exists.
8. No `Client` type or `/clients` route remains anywhere in the tree.
9. Existing channel connections and message flow are unaffected.

## Test plan

- Unit: mapping between the SDK type and Atom entity attributes, both directions.
- Integration: full lifecycle for a credential-less device, a credentialed
  device, and a gateway; setting 0, 1 and many gateways; and one device that is
  both a gateway and a reporter.
- Criterion 6 explicitly — re-homing is where topology and sharing are most
  likely to get accidentally coupled.
- Grep-based check for criterion 8, as a test, so the break stays clean.
- Regression: messaging suite unaffected.

## Risks

- ~~**`Serial` versus Bootstrap `ExternalID`.**~~ **Resolved** —
  [spec §8 C3](../architecture.md#8-decision-record): independent, may coincide, nothing enforced.
  This PRD is no longer blocked on MG-12. Operator documentation should recommend
  using the same value; the code does not require it.
- **Clean break is unrecoverable once released.** Every SDK (TS, JS, Rust), the
  UI, and the docs move together. Sequence the cutover across repos rather than
  merging here and discovering the fan-out.
- **`Serial` uniqueness is per tenant** ([spec §8 C4](../architecture.md#8-decision-record)) —
  two workspaces may each hold `ABC123`; within one workspace it identifies exactly one
  device. Every serial lookup must be tenant-scoped; one that is not
  cross-attributes data between customers and fails silently.

  **Mechanism still open:** Atom's `alias` (`001_initial.sql:104-113`) gives
  uniqueness for free but constrains values to
  `^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`. Uppercase alphanumeric serials
  normalise fine; serials containing `/`, `.` or spaces do not. **Collect real
  meter serial formats before choosing** — otherwise it is a unique index on the
  attribute instead.
