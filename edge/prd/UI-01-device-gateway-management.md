# UI-01 — Device and gateway management

| | |
|---|---|
| **Repo** | `absmach/magistrala-ui` — `apps/mg` (frontend), `backend/` (BFF) |
| **Priority** | P5 — after MG-09/MG-11 |
| **Depends on** | MG-09, MG-11 |
| **Blocks** | UI-02, UI-03 |
| **Status** | Draft |

## Problem

The platform ships Device, `is_gateway` and `gateways[]` as attributes and a
query filter, and **deliberately nothing more**
([spec §2.2](../architecture.md#22-what-follows-from-that--normative)). Gateway-ness
is a UI concept by design. Without this PRD, "gateway" is specified and unbuilt.

## Scope

**In scope**

- Device list and detail: create, edit, enable/disable, delete.
- `serial` (Atom `external_id`) as a first-class, searchable field.
- **`is_gateway` toggle** on any device, at creation or later.
- **Setting a device's gateways** — the `gateways[]` list, 0..N.
- A `/gateways` view: devices where `is_gateway` is true.

**Out of scope**

- The gateway detail view — UI-02.
- Sharing and grants — UI-03.
- Device types — deferred with MG-02/MG-10.
- Anything requiring message data — phase 2.

## Design

### Gateway is a filter, not a section

A gateway is a device with a flag. The `/gateways` view is
`GET /devices?is_gateway=true`, and a gateway's detail page **is** a device detail
page with an extra panel. Do not build a parallel entity surface: a device may be
both a gateway and a reporting device, and a UI that treats them as disjoint
populations breaks that case.

### Setting gateways

The relation lives on the **device** and is replace-the-list, so the natural
control is a multi-select on the device form.

Operators will also want the inverse — "add these 40 meters to this gateway" —
from the gateway page. That is a convenience that fans out to N device writes;
the storage does not change.

> **Concurrency.** Atom has no optimistic concurrency
> (`src/identity/repo.rs:335-341` is last-write-wins), so two operators editing
> one device's gateways will silently lose an edit. Until that is addressed, the
> bulk path should write sequentially and re-read, and the UI should not present
> the result as atomic.

### Deleting a gateway

**Must not cascade.** Deleting a gateway leaves its devices intact, now with a
stale reference that is dropped on read
([spec §2.2](../architecture.md#22-what-follows-from-that--normative), consequence 2).
The confirmation dialog must say what happens — "N devices will no longer list
this gateway" — not imply the devices go with it.

## Acceptance criteria

1. Create a device with a serial containing uppercase, `.`, `-` and `/`; all are
   accepted — there is no format validation.
2. Toggle `is_gateway` on an existing device; it appears under `/gateways` and
   remains under `/devices`.
3. Set a device's gateways to 0, 1 and 3 entries; each round-trips.
4. A device that is both a gateway and a reporter renders correctly in both views.
5. Deleting a gateway leaves its devices intact, and the confirmation says so.
6. Two domains may each hold a device with the same serial.
7. Search by serial finds the device.

## Test plan

- Component tests for the device form, `is_gateway` toggle and gateway
  multi-select.
- E2E: the criteria above against a live stack.
- Explicitly test criterion 4 — the composite device is what the capability model
  exists to allow, and the easiest thing for a UI to get wrong.

## Risks

- **Rebuilding gateways as a separate entity.** The most likely design error, and
  it breaks the composite device. Criterion 4 is the guard.
- **Bulk assignment feels atomic and is not.** N sequential writes with no
  transaction; partial failure must be visible rather than silently partial.
