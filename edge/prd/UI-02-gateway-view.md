# UI-02 — Gateway view

| | |
|---|---|
| **Repo** | `absmach/magistrala-ui` — `apps/mg`, `backend/` |
| **Priority** | P5 |
| **Depends on** | UI-01 |
| **Blocks** | — |
| **Status** | Draft |

## Problem

Click a gateway, see what it is and what it serves. This is the concrete test of
the model: if the three panels cannot be assembled from platform primitives, the
model is wrong.

## Scope

**In scope**

- **Panel 1 — gateway information.** One entity read: name, serial, status,
  credentials and certificate metadata.
- **Panel 2 — declared devices.** `GET /devices?gateways=<id>`, backed by
  `attributesContains` (ATOM-01).
- Navigation from a device to the gateways that reach it, and back.

**Out of scope**

- **Panel 3 — bootstrap configuration.** Needs the `gateway_id` reference on the
  Bootstrap projection (MG-12), which is phase 2.
- **Observed devices** and the healthy/silent/undeclared status. Needs
  `device_id` on messages — phase 2, and the half that makes the view
  *diagnostic* rather than a list.
- Fault correlation.

## Design

Phase 1 shows **declared only**. State that in the UI: a device appears because
someone commissioned it here, not because it has been heard from. An operator who
reads it as liveness will misdiagnose.

The panel is the reverse of a relation stored on the device, so it is one filtered
query — not a stored fleet the gateway owns.

⏸ *Phase 2* adds the observed half and merges the two into a per-device status,
which is where the view earns its keep: **declared but never heard** is a fault
for a wired link, and no other view surfaces it.

## Acceptance criteria

1. A gateway page shows its own attributes and its declared devices.
2. A gateway with no declared devices renders an empty state, not an error.
3. A device declared on three gateways appears on all three pages.
4. From a device, its gateways are listed and navigable.
5. The list is explicitly labelled as declared, not observed.
6. A non-admin sees only devices they may read — the list is authorization-filtered.

## Test plan

- E2E against a live stack for each criterion.
- Criterion 3 with a genuinely multi-gateway device — the case a containment
  query gets right and a naive implementation gets wrong.
- Criterion 6 with a scoped user.

## Risks

- **Reading declared as observed.** An operator seeing a device listed assumes it
  is alive. Until phase 2 there is no liveness here at all; labelling is the only
  mitigation and it is a real one.
- **Fleet size.** A gateway with thousands of devices needs pagination from the
  start.
