# MG-02 — Device Type (Atom Profile) API

| | |
|---|---|
| **Repo** | `absmach/magistrala` (Go) |
| **Priority** | P1 |
| **Depends on** | — |
| **Blocks** | MG-10 |
| **Status** | **⏸ Phase 2 — deferred** |

> **⏸ Deferred to phase 2** ([spec §8 A14](../architecture.md#8-decision-record)).
> Device types are a separate feature from the device/gateway relation and are not in the phase 1 cut.
>
> The design below is unchanged and remains the target.

## Problem

Magistrala has no concept of a device type. A watermeter and a temperature probe
are both an opaque `Client` with a free-form metadata map, so the UI cannot
render per-type views, nothing validates payload shape, and there is no way to
express "this model reports volume and battery, and accepts `set_interval`".

Atom already implements exactly this and Magistrala does not call it.

## What Atom already provides

| Capability | Location |
|---|---|
| Named type, tenant-scoped or global, unique on `(tenant, object_kind, kind, key)` | `migrations/001_initial.sql:48-76` |
| Versioned `json_schema` + `ui_schema` with `draft/active/deprecated/disabled` | `:77-90` |
| Binding on the entity: `profile_id`, `profile_version_id` | `:98-99` |
| **Schema enforcement on entity write** | `src/identity/repo.rs:641` |
| CRUD: `profiles`, `profile`, `profileVersions`, `createProfile`, `createProfileVersion`, `updateProfile` | `src/graphql/profiles.rs:24-209` |
| List entities of a type: `entities(profileId:)` | `src/graphql/entities.rs:79` |

No Atom change is required.

## Scope

**In scope**

- `DeviceType` and `DeviceTypeVersion` types in `pkg/atom`, mapping to Atom
  Profile / ProfileVersion with `object_kind = "entity"`, `kind = "device"`.
- Atom client methods: create, get, list, update, create-version, list-versions.
- Entity create/update carrying `profile_id` and `profile_version_id`
  (`src/graphql/entities.rs:205-206,275-276`).
- `ListEntities` gaining the `profileId` filter.
- Capability document helpers: build a JSON Schema from a measurement/command
  declaration, and read it back, so callers do not hand-write JSON Schema.

**Out of scope**

- HTTP/SDK/CLI surface — MG-10.
- Command dispatch. The type *declares* commands; routing them is unspecified
  (see [architecture.md §7](../architecture.md#7-open-questions)).
- Migrating existing devices onto types.

## Design

### Naming

Atom `Profile` and Bootstrap `Profile` (PR #3555) are unrelated concepts sharing
a word. In `pkg/atom` and every Magistrala-facing surface this is **Device Type**.
Never expose "profile" for this concept — the collision is a real hazard once
both are in play.

### Capability document

The declaration Magistrala cares about, expressed as JSON Schema so Atom enforces
it for free:

```go
type Measurement struct {
    Name   string // "volume"
    Unit   string // "m3"
    Access string // "r" | "rw"
}

type Command struct {
    Name   string            // "set_interval"
    Params map[string]string // {"seconds": "int"}
}
```

Rendered to `json_schema` for validation and `ui_schema` for rendering hints.
Keep the mapping in one place and round-trip it — a helper that generates schema
but cannot parse it back leaves the UI hand-parsing JSON Schema.

### Versioning

Device types are versioned and entities bind to a specific version. Changing a
type must not retroactively invalidate deployed devices, so:

- `createDeviceTypeVersion` creates a new version; it never mutates an existing one.
- Devices stay on their bound version until explicitly moved.
- Version status governs whether *new* bindings are allowed, not whether existing
  ones keep working.

## Acceptance criteria

1. Create a device type with a capability document; read it back with the
   declaration intact through the round-trip.
2. Create a device bound to that type. Attributes satisfying the schema succeed.
3. Attributes **violating** the schema are rejected by Atom, and the client
   surfaces a usable error naming the offending field — not a bare GraphQL error.
4. `ListDeviceTypes` returns both tenant-scoped and global types.
5. `ListEntities(profileID:)` returns exactly the devices bound to that type.
6. Adding version 2 leaves devices bound to version 1 working unchanged.
7. Deprecating a version blocks new bindings and leaves existing ones intact.

## Test plan

- Unit: capability document → JSON Schema → capability document round-trip,
  including edge cases (no commands, no units, `rw` access).
- Integration (Atom in Docker): full lifecycle — create type, version, bind
  device, valid write, invalid write, add version, deprecate.
- Error mapping: assert a schema violation produces a typed error carrying the
  field path.

## Risks

- **JSON Schema error legibility.** Atom returns whatever the `jsonschema` crate
  produces. If that surfaces raw to an operator creating a device, it will be
  unusable. Error translation is part of this PRD's work, not a follow-up.
- **Global vs tenant-scoped types.** Uniqueness differs between the two
  (`001_initial.sql:66-72`). Decide whether Magistrala exposes global types at
  all; if not, always send `tenant_id` and say so explicitly.
