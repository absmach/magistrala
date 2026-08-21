# ATOM-06 — Entity `external_id`

| | |
|---|---|
| **Repo** | `absmach/atom` (Rust) |
| **Priority** | P0 |
| **Depends on** | — |
| **Blocks** | MG-08, MG-09 |
| **Status** | Draft |
| **Decision** | [spec §8 A8](../architecture.md#8-decision-record) |

> **Two decisions block this PRD** ([spec §7](../architecture.md#7-open-questions)).
> Decide before writing the migration — both are one-way doors, and a wrong
> choice silently merges two devices into one with no migration back:
>
> | | Question | Recommendation |
> |---|---|---|
> | **Case** | Are `ABC123` and `abc123` one device or two? | **Case-sensitive.** Tightenable later; the reverse is not |
> | **Whitespace** | Is `"ABC123 "` the same as `"ABC123"`? | Trim — but decide it, do not inherit it from whichever client writes first |

## Problem

Entities carry identifiers assigned outside Atom — serial numbers, MAC
addresses, employee numbers, SKUs. Atom has no field for them.

`alias` is the closest thing and cannot serve: it is slug-constrained to
`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$` (`001_initial.sql:105-113`) and
additionally forbidden from looking like a UUID (`:107-113`). A real meter serial
such as `WM-2024-ABC.123` fails the slug pattern (uppercase, `.`) and any attempt
to normalise it without losing information.

The alternative — storing it in `attributes` — gives no uniqueness guarantee and
no index, so lookup is a JSONB containment scan and two devices can silently
claim the same serial.

## Why this is generic

External identifiers are a property of any system that mirrors things it did not
create. The field carries no semantics: Atom stores, indexes and enforces
uniqueness on an opaque string, and never interprets it.

`alias` and `external_id` are deliberately different: `alias` is a
*human-friendly, URL-safe* name Atom constrains; `external_id` is a
*foreign key into someone else's namespace* that Atom must not constrain.

## Scope

**In scope**

- `external_id TEXT NULL` on `entities`.
- Unique per tenant, ignoring soft-deleted rows:
  ```sql
  CREATE UNIQUE INDEX idx_entities_external_id
    ON entities (tenant_id, external_id)
    WHERE external_id IS NOT NULL AND deleted_at IS NULL;
  ```
- `externalId` on `CreateEntityInput` and `UpdateEntityInput`.
- `externalId` as an exact-match filter on the `entities` query.
- ⏸ *Phase 2* — the same filter on `authorizedObjectIds`, landing with ATOM-02
  which touches that resolver anyway. Nothing in phase 1 consumes it.
- `external_id` on the `Entity` GraphQL type and in `entity.*` workspace events.

> **Phase 1 touches only the `entities` query.** The `authorizedObjectIds` half is
> deferred to phase 2 and lands with
> [ATOM-02](./ATOM-02-authorized-object-ids-filters.md), which edits that resolver
> anyway — so the diff conflict the two would otherwise have is avoided entirely.

**Out of scope**

- Any format validation. The value is opaque by design — that is the point.
  Consumers may impose their own: Magistrala rejects `/` because the value travels
  verbatim in a topic, but that is Magistrala's rule, not Atom's.
- Resources, groups, tenants. Add when a requirement appears; entities is what is
  needed.
- Cross-tenant uniqueness. Two tenants may legitimately hold the same serial.

## Design

### Nullable, and unique only when present

Most entities have no external identifier. A partial unique index gives
uniqueness where the value exists and costs nothing where it does not.

### Case sensitivity — decide explicitly

`tenants.alias` is uniquely indexed on `lower(alias)` (`001_initial.sql:40-42`).
Serial numbers are a different case: `abc123` and `ABC123` may be genuinely
different part numbers in some vendor schemes, and treating them as one would
merge two devices.

**Recommend case-sensitive** (index the raw column), which is the conservative
choice: it can be tightened to case-insensitive later, whereas relaxing a
case-insensitive index after devices have merged is not recoverable.

State the decision in the schema comment either way — this is exactly the kind of
thing that is discovered the hard way.

### Soft-delete interaction

The index excludes `deleted_at IS NOT NULL`, so a deleted device's serial is
reusable. That is almost certainly wanted — replacing a meter with the same
serial should work — but it means `restore_entity` can now fail on a uniqueness
conflict if the serial was reused meanwhile. Handle it with a clear error rather
than a constraint-violation surfacing raw.

### Not a primary key

Consumers may store `external_id` in preference to the UUID — Magistrala's
message pipeline does exactly that (spec §8 A8), keeping the string on every
row so the publish path needs no lookup. That is a consumer choice; Atom's
identity remains the UUID, and `external_id` is mutable.

## Acceptance criteria

1. An entity can be created with an arbitrary-string `external_id` — including
   uppercase, `/`, `.`, spaces and unicode — and read back byte-identical.
2. Two entities in one tenant cannot share an `external_id`; the conflict is a
   clear error, not a raw constraint violation.
3. Two entities in *different* tenants may share one.
4. Multiple entities may have `external_id` NULL.
5. `entities(externalId: "…")` returns the match, scoped to tenant, and uses the
   index — verified by query plan, not assumed.
6. ⏸ *Phase 2* — `authorizedObjectIds(externalId: "…")` narrows correctly and
   cannot widen access.
7. `external_id` can be changed, and cleared to NULL.
8. Soft-deleting an entity frees its `external_id` for reuse; restoring one whose
   identifier was reused fails with a comprehensible error.
9. `external_id` appears in `entity.create` / `entity.update` events.
10. Existing entities and queries are unaffected.

## Test plan

- Migration on seeded data; assert all existing rows survive with NULL.
- Uniqueness: same tenant (reject), different tenants (allow), NULLs (allow many).
- Round-trip of hostile strings — unicode, embedded quotes, 1KB length, leading
  and trailing whitespace. Decide and pin whether whitespace is trimmed.
- Case sensitivity, asserting the documented behaviour explicitly.
- Delete → reuse → restore conflict path (criterion 8).
- Query plan for the `externalId` filter.

## Risks

- **Case-sensitivity is a one-way door.** Choosing case-insensitive merges
  devices that differ only in case, and no migration un-merges them. Pin it with
  a test.
- **Whitespace and unicode normalisation.** `"ABC123 "` and `"ABC123"` will be
  two devices unless trimmed. Trimming is probably right, but it must be a
  decision written down rather than an accident of the client.
- **Length.** `TEXT` is unbounded; an index on a multi-kilobyte value is
  pathological. Consider a sanity cap (e.g. 255) even though the format is
  otherwise unconstrained.
- **Consumers treating it as immutable.** It is mutable, and Magistrala stores it
  denormalised on every message row. Changing a device's `external_id` orphans
  its historical data under the new value. Magistrala must either forbid the
  change or accept the break — flagged in MG-09.
