# ATOM-03 — Reverse policy lookup: `directPolicies(objectId:)`

| | |
|---|---|
| **Repo** | `absmach/atom` (Rust) |
| **Priority** | P3 |
| **Depends on** | — |
| **Blocks** | Sharing UI; correct revocation |
| **Status** | Draft |

## Problem

`directPolicies` filters by subject only — `tenantId`, `subjectKind`,
`subjectId`, `permissionBlockId` (`src/graphql/policies.rs:285-294`). There is no
way to ask **"who has access to this object?"**

Two consequences:

1. A sharing UI cannot show who a resource is shared with without enumerating
   every subject in the tenant and querying each.
2. Revocation is unsafe. Magistrala's `DeletePolicyFilter`
   (`pkg/atom/policy_service.go:78-105`) works around the gap by listing a
   subject's policies and matching client-side, capped at 100 — so revoking
   access on a widely-shared object silently misses policies past that cap.

## Why this is generic

"Who can access X" is the inverse of "what can this subject access", which Atom
already answers. Every system with a sharing model needs both directions. No
domain semantics are introduced.

## Scope

**In scope**

- Add `object_id: Option<ID>` to the `directPolicies` query, returning every
  direct policy whose permission block targets that object.
- Object matching must cover the scope modes that can reference a specific
  object:
  - `object` — `permission_blocks.object_id = $1`
  - `group_direct_objects` / `group_descendant_objects` — blocks whose
    `group_id` contains the object, direct or transitive respectively
- Add `object_kind` / `object_type` as optional co-filters, since an ID alone is
  ambiguous across kinds.

**Out of scope**

- Tenant-, platform-, `object_kind`- and `object_type`-scoped blocks. These grant
  access to an object without naming it; including them would make the result a
  full effective-access computation rather than a policy lookup. If effective
  access is wanted, that is a separate `effectiveAccess(objectId:)` query with
  different semantics — do not conflate them.
- Any change to `authzCheck` or the evaluation engine.

## Design

Extend `ListDirectPolicies` (`src/authz/repo.rs`) with the optional object
filter, joining `permission_blocks` and — for the group scope modes — the
membership tables `object_group_entities` / `object_group_resources`
(`migrations/001_initial.sql:525-544`) and, for descendants,
`object_group_hierarchy` (`:446-454`).

Authorization: reuse `require_policy_read` (`policies.rs:298`) unchanged. Reading
who can access an object is a policy-read operation on the tenant, as today.

**Result semantics must be documented on the field**, because the out-of-scope
exclusion above is surprising if undocumented: this returns *direct policies
naming this object*, not *everyone who can reach it*.

## Acceptance criteria

1. `directPolicies(objectId: <device>)` returns policies whose block is
   `scope_mode: "object"` with that `object_id`.
2. It also returns policies whose block is `group_direct_objects` over a group
   the object is a direct member of.
3. `group_descendant_objects` matches transitively through
   `object_group_hierarchy`; `group_direct_objects` does not.
4. Tenant-, platform- and kind-scoped blocks are **not** returned, and the field
   documentation says so.
5. `objectKind` / `objectType` co-filters narrow correctly.
6. Combining `objectId` with `subjectId` intersects.
7. Callers without policy-read on the tenant are refused.
8. Omitting `objectId` produces results identical to today.

## Test plan

- Integration: build one device reachable four ways — direct object block, direct
  group membership, descendant group membership, and a tenant-scoped block —
  then assert exactly the first three are returned.
- Hierarchy depth: 3-level group tree, confirm direct vs descendant behaviour at
  each level.
- Authorization: caller without policy-read is refused.
- Pagination over a widely-shared object.

## Risks

- **Recursive hierarchy traversal cost.** Descendant matching needs a recursive
  CTE. Bound it by tenant and check the plan on a deep tree.
- **Misreading the result as effective access.** The exclusion in Scope is
  deliberate and load-bearing; if a consumer treats this as "everyone who can
  see X", they will under-report. Documentation on the field is part of the
  deliverable, not a nicety.
