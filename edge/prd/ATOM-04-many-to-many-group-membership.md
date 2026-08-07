# ATOM-04 — Many-to-many object group membership

| | |
|---|---|
| **Repo** | `absmach/atom` (Rust) |
| **Priority** | P0 |
| **Depends on** | — |
| **Blocks** | MG-03, MG-04 |
| **Status** | Draft |
| **Decision** | [spec A1](../architecture.md#8-decision-record) |

> **Cost correction.** When recommending this change I described it as
> "near-trivial — a `PRIMARY KEY` change". That was based on the migration tool's
> `ON CONFLICT` clauses. Having read Atom's repository layer, it is **moderate,
> not trivial**: single membership is assumed in the upsert semantics, the delete
> path, the public API shape, the list queries, and — most importantly — the
> authorization evaluation path. The decision may still be right; the estimate
> was wrong. See [Scale](#scale-of-the-change).

## Problem

`object_group_entities` has `PRIMARY KEY (entity_id)`
(`migrations/001_initial.sql:525-532`) — an entity belongs to **at most one**
object group. Same for resources, `PRIMARY KEY (resource_id)` (`:537-544`).

This forbids overlapping object sets:

```
"Customer A meters"  → granted to Customer A
"Building 5 meters"  → granted to a maintenance contractor
Meter 7 ∈ both, and neither set contains the other
```

A hierarchy cannot express it, because the sets intersect without nesting.

## Why this is generic

Group membership being many-to-many is the ordinary case for a grouping
primitive — tags, labels, collections, teams. Atom's own API already reads as
though it were: `entityGroups(entityId)` returns a **list**
(`src/graphql/groups.rs:104`), and the mutation is named `addGroupMember`, not
`setGroupParent`. No product-specific semantics are introduced; a constraint is
removed.

## Scale of the change

Single membership is assumed in five places, in increasing order of risk.

### 1. Schema — trivial

```sql
ALTER TABLE object_group_entities
  DROP CONSTRAINT object_group_entities_pkey,
  ADD PRIMARY KEY (group_id, entity_id);
```

Data-preserving. Same for `object_group_resources` if included (see Scope).

### 2. Upsert semantics — small

`src/identity/repo.rs:511-523` currently **moves** an entity between groups:

```sql
INSERT INTO object_group_entities (group_id, entity_id, tenant_id)
VALUES ($1, $2, $3)
ON CONFLICT (entity_id) DO UPDATE
SET group_id = EXCLUDED.group_id, ...
```

Becomes `ON CONFLICT (group_id, entity_id) DO NOTHING` — additive rather than
replacing. This also settles spec §8 **E3**: today's behaviour is a silent
move; after this it is a genuine add.

### 3. Removal — small, but an API change

`src/identity/repo.rs:556` deletes by entity alone:

```sql
DELETE FROM object_group_entities WHERE entity_id = $1
```

That now means "remove from *all* groups". `removeGroupMember` already takes a
`group_id` (`groups.rs:699`), so the mutation is fine — but
`clear_entity_parent_group_in_tx` and any caller expecting "clear the parent"
need explicit semantics: remove-from-one versus remove-from-all.

### 4. The `parent_group_id` attribute API — moderate, semantic

Membership is currently settable through an **attribute** on create/update —
`parent_group_id_from_attrs` (`src/authz/repo.rs:59,237-241`). A scalar attribute
cannot express a set.

Decide one:

- **A.** Keep `parent_group_id` as a convenience meaning "sole membership"
  (replaces all), with `addGroupMember` / `removeGroupMember` as the set API.
  Backwards compatible, but two mechanisms with different semantics on one
  relation is exactly the kind of thing that later confuses everyone.
- **B.** Deprecate the attribute path; membership is only mutated through the
  explicit mutations. Cleaner; breaks existing callers, including Magistrala's
  projection (`pkg/atom/mapping.go:46` writes `parent_group_id`).

**Recommend B**, consistent with the "avoid technical debt" ruling in
[spec §8 C1](../architecture.md#8-decision-record).

### 5. Queries joining membership — moderate, correctness-critical

This is the part that matters.

**Row multiplication in listings.** `src/authz/repo.rs:4839-4851`:

```sql
candidates AS (
  SELECT e.id, ..., gep.group_id AS parent_group_id
  FROM entities e
  LEFT JOIN group_entity_parents gep ON gep.entity_id = e.id
  ...
  AND ($8::uuid IS NULL OR gep.group_id IN (SELECT id FROM target_groups))
```

With M:N this yields one row **per (entity, group) pair** — duplicate entities in
every listing and an inflated `total`. Needs `EXISTS`-style filtering rather than
a join that projects the group, or explicit de-duplication.

**Missed grants in authorization.** `src/authz/repo.rs:5687-5692`:

```sql
SELECT e.id, ..., gep.group_id AS parent_group_id
FROM entities e
LEFT JOIN group_entity_parents gep ON gep.entity_id = e.id
WHERE e.id = $1 ...
```

`fetch_optional` over a now-multi-row result takes one arbitrary group. Since
this record feeds group-scoped policy evaluation, **an entity in two groups would
have grants through one of them silently ignored** — non-deterministically, since
which row wins is unspecified.

This is the single most important change in the PRD: `AuthzObjectRecord`'s
`parent_group_id: Option<Uuid>` must become a set, and every `group_direct_objects`
/ `group_descendant_objects` evaluation must consider all of them.

**Views.** `group_entity_parents` / `group_resource_parents`
(`001_initial.sql:549-555`) keep working but their names become misleading —
they are membership, not parentage.

## Scope

**In scope**

- Schema change for `object_group_entities`.
- Additive upsert; explicit removal semantics.
- De-duplicated list queries with correct `total`.
- Set-based membership in the authorization evaluation path.
- Decision and implementation of the `parent_group_id` attribute question.

**Open — decide before starting**

- **Include `object_group_resources`?** Symmetry argues yes; no requirement
  exists yet (channels in multiple groups). Splitting them leaves an asymmetry
  that will confuse; doing both roughly doubles the query work.

**Out of scope**

- Group hierarchy — `object_group_hierarchy PRIMARY KEY (child_id)`
  (`:446-454`) stays a tree. A group has one parent; only *membership* becomes
  many-to-many.
- Principal group membership, unless it shares the same tables.

## Acceptance criteria

1. An entity can be added to two groups and appears in both `groupMembers`
   listings.
2. `entityGroups` returns both.
3. `entities(parentGroupId:)` returns the entity **once**, with `total` counting
   it once, for either group.
4. A grant via `group_direct_objects` on **either** group authorizes the entity —
   verified for both, not just the first.
5. Removing from one group leaves the other membership and its grants intact.
6. `includeDescendants` traversal is correct when an entity is in two groups in
   different subtrees.
7. Adding an entity to a group it is already in is idempotent.
8. Cross-tenant membership is still rejected (`identity/repo.rs:505-509`).
9. Existing single-membership data behaves identically after migration.

## Test plan

- Migration against seeded data; assert every existing membership survives.
- **Criterion 4 is the security-relevant one** — entity in groups G1 and G2, grant
  only via G2, assert allowed. This fails today's evaluation path and is the
  reason for the change.
- Criterion 3 with an entity in three groups, asserting no duplicates and correct
  `total` under pagination.
- Descendant traversal across two subtrees.
- Regression: full existing group and authz suites.

## Risks

- **Silent grant loss** if the evaluation path is not fully converted to sets.
  Fails open or closed non-deterministically depending on row order — the worst
  possible failure mode. Criterion 4 is the guard.
- **Duplicate rows** in listings are cosmetic in the UI but corrupt `total` and
  therefore pagination.
- **Query plans** change once the join can fan out. Check plans for
  `authorized_object_ids` with and without a group filter.
- **Scope creep into resources.** Decide up front; discovering halfway that
  resources need it too doubles the work mid-flight.
