# MG-07 — Gateway publish-on-behalf-of enforcement

| | |
|---|---|
| **Repo** | `absmach/magistrala` (Go) |
| **Priority** | — |
| **Status** | **WITHDRAWN** |
| **Superseded by** | [spec §8 A7](../architecture.md#8-decision-record) |

---

## Withdrawn

This PRD existed to enforce a `gateway_id` attachment on the publish path:

```
allow if device.gateway_id == publisher
```

**The attachment it enforced no longer exists.** One device can broadcast to
several gateways — a BLE meter heard by three gateways in range is a normal
deployment, not an error — so a scalar `gateway_id` cannot represent the
relationship at all.

Per [spec §8 A7](../architecture.md#8-decision-record), **the channel is the boundary**: a
gateway connected to a channel may publish any `device_id` on it. There is no
per-device authorization on the publish path and no device lookup, which is the
point — the hot path stays lookup-free.

## What happened to its parts

| Was | Now |
|---|---|
| `gateway_id == publisher` check | Removed. Channel authorization, which already exists, is the whole control. |
| Attachment cache | Removed. Nothing to cache. |
| Cache invalidation on re-homing | Removed. Nothing to invalidate. There is no re-homing — a device is simply heard by different gateways over time. |
| `provisioning_state` deny clause (B1) | Removed. Superseded by late binding — see [spec §8 A8](../architecture.md#8-decision-record); data is stored whether or not a device entity exists. |
| "Which devices does this gateway serve?" | Derived, not stored: `DISTINCT device_id WHERE publisher = <gateway>` over the message store (MG-06). Handles many-gateways-per-device for free. |

## The accepted risk

A compromised gateway can fabricate readings for any `device_id` on channels it
holds — including impersonating another customer's meter if they share a channel.

This was accepted deliberately in exchange for a lookup-free publish path. The
mitigation is deployment-level: **segregate channels per site or customer** where
cross-fabrication matters. That guidance belongs in operator documentation, and
is recorded in A7 rather than lost here.

## If this needs revisiting

The trigger would be a requirement for per-device publish authorization — for
example a multi-tenant gateway estate where channels cannot be segregated. The
shape it would take is a group-scoped `publish_on_behalf_of` grant
(devices in an object group, gateways granted over that group via
`group_direct_objects`), which composes with ATOM-04's many-to-many membership
and would let several gateways serve one device without a scalar attachment.

That was the alternative considered and not taken.
