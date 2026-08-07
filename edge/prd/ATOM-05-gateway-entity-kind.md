# ATOM-05 — Add `gateway` entity kind

| | |
|---|---|
| **Repo** | `absmach/atom` (Rust) |
| **Priority** | — |
| **Status** | **WITHDRAWN** |
| **Superseded by** | [spec §8 A12](../architecture.md#8-decision-record) |

---

## Withdrawn

This PRD added `gateway` to Atom's `entities.kind` enum. **Gateway is a
capability, not a type**, so no new kind is needed — gateways stay
`entity_kind: device` with an `is_gateway` attribute.

### Why

Types are exclusive; roles compose. The model defines a gateway as *a Device with
a proxy role* ([spec §2.1](../architecture.md#21-a-gateway-is-a-path-not-a-container)),
and a distinct kind contradicts that on a real and common device: a smart
electricity meter that also concentrates wM-Bus water meters produces its own
readings **and** relays others. Under exclusive kinds it must be one or the
other, and either answer is wrong.

Surveying comparable platforms found the same conclusion everywhere:

| Platform | Gateway is |
|---|---|
| ThingsBoard | Device + `Is gateway` **boolean** |
| AWS Greengrass v2 | Core device — an IoT `thing`, **same type as clients** |
| Azure IoT Edge | Device identity with edge capability |
| ChirpStack | Separate — but its gateways are dumb radio infrastructure producing no data |

### What happened to its parts

| Was | Now |
|---|---|
| `Gateway` variant on `EntityKind` + two CHECK relaxations | Not needed |
| `entity:gateway` object type | `attributesContains: {is_gateway: true}` — ATOM-01 |
| Separate gateway profile namespace | One namespace. A device that both measures and relays needs *one* type, which is more correct |
| Gateway-specific `ActionAssignmentRule`s | Not needed — the existing `{entity_kind: device, publish, resource:channel}` rule keeps applying |

### The trap it carried, now gone

`pkg/atom/bootstrap.go:72-87` installs `{entity_kind: device, publish,
resource:channel}` as the only publish guardrail. Introducing a `gateway` kind
would have **silently stripped every gateway's right to publish** until matching
rules were added — surfacing as a bare authorization denial with nothing pointing
at the cause. Withdrawing this PRD removes that failure mode entirely.

### If this needs revisiting

The trigger would be a requirement to grant over *all gateways* as a first-class
object type, in a deployment where attribute filtering does not scale. The
migration stays permissive and additive, so it can be done later at the same cost
— with the publish-guardrail trap as the thing to plan for.
