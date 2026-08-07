# ATOM-02 — Expose scoping filters on `authorizedObjectIds`

| | |
|---|---|
| **Repo** | `absmach/atom` (Rust) |
| **Priority** | P0 |
| **Depends on** | — |
| **Blocks** | MG-08 (reader authorization) |
| **Status** | **⏸ Phase 2 — deferred** |

> **⏸ Deferred to phase 2** ([spec §8 A14](../architecture.md#8-decision-record)).
> Justified by MG-08 **part B** device-set scale, which is phase 2. Part A checks a handful of publishers, for which per-publisher `authzCheck` is sufficient.
>
> The design below is unchanged and remains the target.

## Problem

`authorizedObjectIds` answers "which objects may this subject act on". Its query
struct and repository implementation support attribute, profile, status, group
and descendant filtering — but the GraphQL resolver hardcodes all of them to
`None`/`false`.

Callers must therefore page the subject's **entire** authorized set into memory
and filter client-side. For a consumer that wants "the meters this customer may
read, within this group" that means materialising every authorized ID first,
which does not scale and makes `total` meaningless for the caller's real query.

## Why this is generic

The listing queries (`entities()`, `resources()`) already accept these filters.
This makes the authorization query accept the same ones, so "what can I see"
and "what can I see, narrowed" are the same question with the same vocabulary.
Nothing caller-specific is introduced.

## Current state

`AuthorizedObjectIdsQuery` (`src/models/access.rs`) carries:

```rust
subject_id, action, object_kind, object_type, tenant_id, q,
attributes_contains, profile_id, entity_status, group_type,
parent_group_id, include_descendants, limit, offset
```

The resolver (`src/graphql/authz.rs:47-61`) passes only the first six and pins
the rest:

```rust
attributes_contains: None,
profile_id: None,
entity_status: None,
group_type: None,
parent_group_id: None,
include_descendants: false,
```

The repository implements all of them (`src/authz/repo.rs:155,189,221`).

## Scope

**In scope**

- Extend `AuthorizedObjectIdsInput` with: `attributesContains`, `profileId`,
  `entityStatus`, `parentGroupId`, `includeDescendants`.
- Thread them through the resolver (`src/graphql/authz.rs:47-61`).
- Preserve the existing capability check
  (`access::require_authz_check_access`, `authz.rs:40`) unchanged — filters
  narrow a result set, they must never widen it.

> **Coordinate with [ATOM-06](./ATOM-06-entity-external-id.md).** Both add
> parameters to the *same* `authorizedObjectIds` resolver
> (`src/graphql/authz.rs:47-61`) — this one adds the existing repository filters,
> ATOM-06 adds `externalId`. Independent in design, guaranteed to conflict in the
> diff. Sequence them or land them together.

**Out of scope**

- `group_type` — no consumer yet. Leave pinned to `None` rather than exposing an
  unused knob.
- Changes to the scoped-token ceiling semantics described at `authz.rs:43-46`.
  Filters apply **after** ceiling filtering; adding them must not alter that
  order.
- Repository or SQL changes.

## Design

Direct plumbing. Parameter naming and types mirror `entities()` after ATOM-01 so
the two queries read the same way.

The critical invariant: **filters are conjunctive with the authorization result,
never disjunctive.** A caller supplying `parentGroupId` for a group they cannot
read must receive an empty set, not an error and not the unfiltered set.

## Acceptance criteria

1. `authorizedObjectIds(input: {subjectId, action: "read", objectKind: "entity",
   objectType: "entity:device", parentGroupId: <group>})` returns only device IDs
   that are both authorized for the subject and in that group.
2. `attributesContains` narrows the same way.
3. `includeDescendants: true` walks the group tree; `false` (and omitted) does
   not.
4. `total` reflects the filtered count.
5. A subject with no grants receives an empty list for every filter combination —
   filters cannot grant access.
6. Scoped-token ceiling filtering still applies, and applies first.
7. Omitting all new arguments produces results identical to today.

## Test plan

- Unit: each new parameter reaches `AuthorizedObjectIdsQuery` unchanged;
  omitted parameters retain today's defaults.
- Integration:
  - subject with a `group_direct_objects` grant + `parentGroupId` filter →
    intersection only;
  - subject with **no** grant + any filter → empty;
  - scoped token whose ceiling excludes an object the direct policy allows →
    object absent regardless of filters;
  - pagination: `limit`/`offset` over a filtered set returns each ID exactly once
    across pages.
- Regression: existing `authorizedObjectIds` tests pass untouched.

## Risks

- **Widening by accident** is the failure mode that matters. A filter applied as
  an `OR` branch, or applied before the ceiling filter, becomes privilege
  escalation. Acceptance criteria 5 and 6 are the guards; both need explicit
  tests, not incidental coverage.
- Query-plan regression on the larger `WHERE` clause — check the plan for the
  common case (`objectKind: entity` + `parentGroupId`) before merging.
