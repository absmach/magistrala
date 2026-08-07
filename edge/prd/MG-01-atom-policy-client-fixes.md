# MG-01 — Fix Atom policy client defects

| | |
|---|---|
| **Repo** | `absmach/magistrala` (Go) |
| **Priority** | P0 |
| **Depends on** | — |
| **Blocks** | MG-04, MG-08 |
| **Status** | Draft |

## Problem

Four defects in `pkg/atom` sit directly beneath the access-control work. Each is
latent today — mostly because `PolicyService` is not wired into any running
binary — and each becomes a live correctness or security bug the moment it is.

### 1. `objectType` is sent unnamespaced

`policy_service.go:139` sends `ObjectType: entityKind(KindClient)` → `"device"`.
The write path sends the namespaced form `"entity:device"`
(`policy_service.go:225`).

Atom requires the namespaced form and says so explicitly:

> `object_type must be the full namespaced value matching object_kind, e.g. 'entity:device'`
> — `src/identity/access_tokens.rs:275`; construction at `src/graphql/entities.rs:157-165`

The filter therefore never matches what the writer stored. Confirmed against
Atom source, not inferred.

### 2. `DeletePolicyFilter` silently truncates

`policy_service.go:78-105` lists a subject's direct policies with
`Limit: policyPageLimit` (100, `:13`) and deletes matches from that single page.
A subject with more than 100 policies keeps access to everything past the cap.
Revocation reporting success while leaving access in place is a security defect,
not a pagination nit.

### 3. `CapabilityID` cannot see past 100 actions

`client.go:272-287` linear-scans `actions(limit: 100)`. Action 101 is
unresolvable, and the failure is a confusing "not found" far from the cause.

### 4. No applicability registered for `objectKind: entity`

`bootstrap.go:29-70` registers applicability for `tenant`, `group`,
`resource:channel`, `resource:rule` and `resource:report` — but nothing for
`entity`. Meanwhile `fluxmq/api/http/publish.go:184` already checks `read` on an
entity, and every device-level grant this project introduces will target
`entity:device`.

## Scope

**In scope**

- Fix the `objectType` mismatch. Introduce a single helper that produces the
  namespaced object type and use it on **both** the read and write paths, so the
  two cannot drift again.
- Make `DeletePolicyFilter` paginate to exhaustion, deleting across all pages.
- Make `CapabilityID` resolve reliably — paginate, or look up by name if Atom
  supports it. Cache resolved IDs; they are immutable.
- Register applicability for `entity` / `entity:device`: `read`, `write`,
  `delete`, `manage`.
- Widen `isSupportedObjectList` (`policy_service.go:182-187`) beyond
  `user + client + view`, which is required by MG-08.

**Out of scope**

- Wiring `PolicyService` into services — MG-08 does that for readers.
- Group-scoped blocks — MG-04.
- Any behaviour change to `AddPolicy`'s block-per-call shape. Block reuse is a
  legitimate optimisation but belongs with MG-04, which changes that path anyway.

## Design notes

The `objectType` fix should not be a literal-string edit in two places. Add:

```go
// atomObjectType returns the namespaced object type Atom requires,
// e.g. "entity:device", "resource:channel".
func atomObjectType(objectKind, kind string) string
```

and route both `policyGrantObjectType` (`policy_service.go:220-241`) and the
`ListAllObjects` query (`:135-143`) through it. The bug exists because the two
paths independently construct the same string; the fix is to remove that
independence.

For deletion, the loop must be resilient to the page shifting underneath it as
items are removed — page by offset and re-query, or collect all IDs first and
then delete. Collecting first is simpler and correct; the sets are small.

## Acceptance criteria

1. Read and write paths produce byte-identical `object_type` values for the same
   logical object, enforced by a test that compares them directly.
2. `ListAllObjects` against a subject with an object-scoped `read` grant on a
   device returns that device's ID. (This returns nothing today.)
3. Revoking a permission on a subject holding 250 policies removes **all**
   matching policies. Verified by re-querying after deletion, not by return value.
4. `CapabilityID` resolves an action registered beyond the first 100.
5. `atom-bootstrap` registers `entity` applicability; re-running it is idempotent.
6. `isSupportedObjectList` admits the `read`-on-`entity:device` case MG-08 needs
   and still rejects genuinely unsupported combinations.

## Test plan

- Unit: object-type helper across every kind; `isSupportedObjectList` truth table.
- Unit: `DeletePolicyFilter` against a mock returning 250 policies across
  3 pages — assert every matching ID was passed to delete.
- Integration (Atom in Docker): grant `read` on a device, assert
  `ListAllObjects` returns it; revoke, assert it is gone.
- Bootstrap idempotency: run `atom-bootstrap` twice, assert no duplicates and no
  errors.

## Risks

- **Changing `objectType` invalidates existing stored blocks.** Any block written
  with the unnamespaced form still holds it.

  **Resolved — [spec §8 C1](../architecture.md#8-decision-record): no backwards compatibility.**
  Fix the read path via the shared helper and leave it there. No dual-form
  matching, no compatibility shim. Blocks holding the old value stop matching and
  must be rewritten; that belongs in release notes rather than being discovered
  in the field. With this ruling the PRD is purely additive plus fixes.
