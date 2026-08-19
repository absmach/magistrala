# UI-03 — Sharing management

| | |
|---|---|
| **Repo** | `absmach/magistrala-ui` — `apps/mg`, `backend/` |
| **Priority** | P5 |
| **Depends on** | UI-01, MG-03, MG-04 · ATOM-03 (for "who can see this") |
| **Blocks** | — |
| **Status** | Draft |

## Problem

"A customer sees only their meters" was the requirement that started this design.
The platform delivers it as object groups plus group-scoped grants
([spec §3.5](../architecture.md#35-groups-and-sharing)) — but a group and a
permission block are not things an operator should ever see. Without a UI, the
feature exists and is unusable.

## Scope

**In scope**

- Create and manage **device groups** — named sets, with membership editing.
- Grant a **user** read access over a group; revoke it.
- From a device: which groups it belongs to, and who can see it.
- From a user: which devices they can see.

**Out of scope**

- **Per-device *data* sharing.** Phase 1 shares device *records*; filtering
  message data by device needs `device_id` and is phase 2. **The UI must not
  imply otherwise** — see Risks.
- Group hierarchy beyond a single level, unless MG-03 exposes it.
- Roles beyond read.

## Design

### Speak in the operator's language

The operator thinks "give this customer these meters". The platform does that as
one object group, one permission block scoped `group_direct_objects`, and one
direct policy per subject. **None of those words should appear in the UI.**

### Membership is the sharing operation

Adding a device to a customer's group grants access; removing it revokes. One
grant per customer, not one per device — so the primary interaction is editing
membership, not editing permissions.

### Removal is not revocation

With many-to-many membership (ATOM-04) a device can be in several groups, so
removing it from one **does not necessarily** end access — another group may
still grant it. A UI that says "access revoked" after one removal will be wrong.

This is what ATOM-03 is for: "who can still see this device" must be *asked*, not
inferred from the group just edited.

### Fleet groups do not exist

Groups mean sharing and nothing else. There is no gateway-fleet group to confuse
this with — the device↔gateway relation is an attribute (UI-01), deliberately
kept out of this namespace.

## Acceptance criteria

1. Create a group, add devices, grant a user read; that user sees exactly those
   devices.
2. Removing a device from the group ends that user's access to it.
3. A device in two groups granted to two different users is visible to both.
4. Removing it from one group leaves the other user's access intact, **and the UI
   reflects that** rather than reporting access revoked.
5. "Who can see this device" lists every subject with access, however granted.
6. A user with no grants sees no devices — not all of them.
7. Adding 200 devices to a group is one operator action.

## Test plan

- E2E for each criterion.
- Criteria 3–5 together: the multi-group case is where a naive UI reports
  revocation that did not happen.
- Criterion 6 explicitly — the empty-set inversion is the security-relevant
  failure and must be tested, not assumed.

## Risks

- **Implying data-level sharing.** In phase 1 a granted customer can see the
  device *record*, not its readings. If the UI presents this as "customer can see
  their meter data", it is lying until phase 2. Label it, and do not build a data
  view that silently shows everything on the channel.
- **Reporting revocation that did not occur** — criterion 4. Needs ATOM-03;
  without it the honest UI can only say "removed from this group", not "access
  revoked".
- **Groups drifting into other uses.** The namespace is single-purpose by design;
  anything that adds an organisational or topological group here re-creates the
  problem A10 removed.
