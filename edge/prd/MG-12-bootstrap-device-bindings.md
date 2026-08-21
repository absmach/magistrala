# MG-12 — Bootstrap device/gateway bindings and fleet rendering

| | |
|---|---|
| **Repo** | `absmach/magistrala` (Go) |
| **Priority** | P5 |
| **Depends on** | PR #3555 merged and rebased, MG-09, ATOM-01 |
| **Blocks** | MG-13 |
| **Status** | **⏸ Phase 2 — deferred** |

> **⏸ Deferred to phase 2** ([spec §8 A14](../architecture.md#8-decision-record)).
> No bus `address` to deliver — A13 is deferred with the messaging track.
>
> The design below is unchanged and remains the target — this is a scope call,
> not a reversal.

## Problem

A gateway running the Magistrala agent needs a rendered configuration: its
credentials, its channels, and — the part that does not exist — **which sensors
it fronts and how to reach them locally**. Without the fleet list the agent
cannot poll a BLE or Modbus meter, because it does not know the meter exists.

[PR #3555](https://github.com/absmach/magistrala/pull/3555) reintroduces the
Bootstrap service with profiles, binding slots, templated rendering and an
authenticated device-facing protocol. It predates the device model, so its
binding slots speak `client` and its render context knows only one device.

## Prerequisite

PR #3555 branches from `c09020a29`, where the Atom package was `internal/atom`.
It has since moved to `pkg/atom` and the PR reports `mergeable: false`. **Rebase
first.** Nothing here is actionable until it lands.

## Scope

**In scope**

- `BindingSlot.Type`: `client` → `device`, add `gateway`.
- Render context: gateway identity plus its attached-device fleet.

**In scope, added by [spec §8 A13](../architecture.md#8-decision-record)**

- **Render the serial → bus-address map** into gateway config, generated from the
  declared relation. For each device where `gateways` contains this gateway, emit
  its serial and the `address` blob on *that* edge.
- The blob is passed through **verbatim**. Magistrala does not parse it, and this
  PRD must not introduce Modbus/BLE/M-Bus awareness to render it.

**Out of scope**
- **Reconciling `ExternalID` with `Serial`.** [spec §8 C3](../architecture.md#8-decision-record):
  they are independent, may coincide, and nothing enforces it.
- Gateway-announced discovery — MG-13.
- The device-facing crypto protocol, which is complete and unchanged
  (`bootstrap/device_bootstrap.go`).
- Bootstrap's own service architecture.

## Design

### Binding slots

`bootstrap/bindings.go` currently declares `Type ∈ "client","channel","cert"`.
Rename to `device` and add `gateway`. Since PR #3555 is unreleased, this is a
clean edit with no compatibility burden — provided it lands before release.

### Fleet in the render context

`RenderContext.Device` (`bootstrap/bindings.go`) becomes the gateway, plus:

```go
type RenderContext struct {
    Gateway  GatewayContext
    Devices  []DeviceContext   // the attached fleet
    Vars     map[string]any
    Bindings map[string]BindingContext
}

type DeviceContext struct {
    ID         string
    Serial     string
    DeviceType string
    Address    map[string]any   // opaque; the edge's address for *this* gateway
}
```

So a profile template can render the roster:

```
{{ range .Devices }}
  - serial:  {{ .Serial }}
    type:    {{ .DeviceType }}
    {{- with .Address }}
    address: {{ toJSON . }}     {{/* opaque — rendered, never parsed */}}
    {{- end }}
{{ end }}
```

Devices on self-identifying buses (wM-Bus, BLE) have no `address` and the block is
omitted.

### What the fleet list is for

Per [spec §8 A13](../architecture.md#8-decision-record), the roster tells the
agent **which meters are its own, and — for bus-addressed protocols — where to
find them.** Self-identifying buses (wM-Bus, BLE) need only the serial; the agent
matches its own scan results against the list. Modbus and M-Bus primary
addressing additionally carry the `address` blob, because the agent cannot derive
a serial from a unit ID.

So the contract is a clean split:

| | Knows |
|---|---|
| Cloud | which devices exist, their serials, their types, which gateway fronts them |
| Gateway | how to physically reach each serial on its local bus |

This keeps BLE/Modbus/M-Bus/LoRa addressing entirely out of Magistrala. The cost
is that a replacement gateway must rescan rather than inheriting a map, and
"which Modbus unit is meter ABC123 on?" is answerable only on the gateway.

**The agent's side of this contract is not designed** — see
[spec §8 E5](../architecture.md#8-decision-record). The gateway must persist the roster, its own
scan results, and the join between them, across restarts and cloud outages. That
is `absmach/agent` work outside this repo, but the *contract* — what this PRD
renders and what MG-13's announce accepts — must be settled here, or the cloud
side ships something the agent cannot consume.

One sub-question lands directly on this PRD: **does the gateway publish by serial
or by device ID?** MG-05's `d` segment accepts either. If serial resolves as a
route, the rendered roster need not carry cloud-assigned device IDs at all.

### Snapshot versus live

`BindingResolver` snapshots resources at bind time so the render path never calls
external services (`bootstrap/bindings.go`). The fleet is different: it changes
whenever a device is attached, detached or re-homed.

Options:

| | Behaviour | Cost |
|---|---|---|
| **Snapshot** | Consistent with existing design; fleet stale until refresh | Needs a refresh trigger on every attachment change |
| **Resolve at render** | Always current | Breaks the "render never calls out" invariant |

**Recommend snapshot**, reusing the existing `RefreshBootstrapBindings` path
(`pkg/sdk/bootstrap.go:452`), with attachment changes marking the config stale.
It preserves the architecture; the cost is an explicit refresh, which the agent
already has a mechanism to trigger.

### Identity: two independent values

Bootstrap `Config.ExternalID` ("a device MAC address is a good choice",
`bootstrap/README.md`) and Device `Serial` look like the same concept but answer
different questions: `Serial` identifies a physical device within a workspace;
`ExternalID` identifies an enrollment used to fetch a config.

Per [spec §8 C3](../architecture.md#8-decision-record) they are **independent**. They may
coincide, and operator documentation should recommend using the same value, but
nothing enforces it and neither is resolvable from the other.

## Acceptance criteria

1. A profile template renders a gateway config including its channels and
   credentials.
2. The rendered config lists all declared devices with serial and device type, and
   the `address` blob for those that have one **on this gateway's edge** — not the
   address from some other gateway's edge.
2a. A device with no address renders without the field; nothing in the render path
   inspects the blob's contents.
3. Attaching a device and refreshing bindings updates the rendered fleet.
4. Detaching removes it from the fleet.
5. A gateway whose `ExternalID` differs from its Device `Serial` bootstraps
   successfully — the two are not coupled.
6. `gateway` and `device` binding slot types resolve and validate.
7. The device-facing challenge/response flow is unchanged, verified by the
   existing PR #3555 tests passing untouched.
8. Secrets remain in Bootstrap's PostgreSQL; only non-secret metadata is
   projected to Atom (`bootstrap/atom.go`).

## Test plan

- Unit: render context construction; template rendering with 0, 1 and many
  attached devices.
- Integration: create gateway + enrollment, attach devices, bootstrap, assert the
  fleet in the decrypted config.
- Staleness: attach without refresh → old fleet; refresh → new fleet. Assert both
  halves so the documented behaviour is pinned.
- Regression: full PR #3555 suite.

## Risks

- **Fleet size in rendered config.** A gateway with 500 meters produces a large
  encrypted payload over a possibly constrained link. Serial + type + an optional
  small address blob per device — measure before assuming it fits (spec §8 D1).
- **Staleness window.** Between attaching a device and refreshing, the agent does
  not know it exists. Must be documented behaviour rather than a surprise, and it
  interacts directly with MG-13's discovery flow.
- **Stale addresses.** The cloud now holds the wiring map (A13), so "which Modbus
  unit is meter ABC123 on?" is answerable centrally and a replacement gateway
  inherits it. The new risk is the inverse: if someone re-wires the bus without
  updating the edge, the rendered config is confidently wrong. Agent-reported
  discovery (A13) is the mitigation — the gateway can contradict the record.
