# MG-10 — Device Type API surface

| | |
|---|---|
| **Repo** | `absmach/magistrala` (Go) |
| **Priority** | P4 |
| **Depends on** | MG-02, MG-09 |
| **Blocks** | MG-11 |
| **Status** | **⏸ Phase 2 — deferred** |

> **⏸ Deferred to phase 2** ([spec §8 A14](../architecture.md#8-decision-record)).
> Surface for MG-02, deferred with it.
>
> The design below is unchanged and remains the target.

## Problem

MG-02 gives `pkg/atom` the ability to manage device types. Nothing external can
reach it — no HTTP route, no SDK method. Operators cannot define a watermeter
type, and the UI has nothing to render from.

## Scope

**In scope**

- `DeviceType` and `DeviceTypeVersion` in `pkg/sdk`, with the capability document
  as structured fields rather than raw JSON Schema.
- CRUD + versioning over HTTP and SDK.
- Binding a device to a type (extends the MG-09 device surface).
- Listing devices by type.

**Out of scope**

- CLI and OpenAPI — MG-11.
- Command dispatch. The type *declares* commands; routing is unspecified.
- Ingest-time payload validation against the type (see Design).

## Design

### Public shape

Operators should not hand-write JSON Schema. The SDK exposes the declaration;
`pkg/atom` (MG-02) renders it to `json_schema` + `ui_schema`:

```go
type DeviceType struct {
    ID, Name, Key, Description, DomainID string
    Version      int
    Measurements []Measurement
    Commands     []Command
    Status       string   // active | deprecated | disabled
}
```

The raw schema should remain readable for advanced use, but the structured form
is the documented path. If callers must drop to JSON Schema for ordinary work,
the abstraction has failed.

### Routes

```
POST   /{domainID}/device-types
GET    /{domainID}/device-types
GET    /{domainID}/device-types/{id}
PATCH  /{domainID}/device-types/{id}
POST   /{domainID}/device-types/{id}/versions
GET    /{domainID}/device-types/{id}/versions
GET    /{domainID}/devices?device_type_id={id}
```

Hyphenated to match existing multi-word resource conventions
(`resource:bootstrap-config` in PR #3555).

### Validation semantics — state these explicitly

Atom validates entity **attributes** against the bound schema on write
(`src/identity/repo.rs:641`). It does **not** validate message payloads at
ingest — that path never touches Atom.

So a device type constrains device *metadata*, not telemetry, unless ingest-time
validation is built separately. This is a genuinely surprising distinction and
must be documented on the API, or users will assume readings are validated when
they are not.

Whether telemetry validation is wanted at all is an open question — it puts a
schema lookup on the hot path. Out of scope here; do not imply it.

### Naming

Atom `Profile` and Bootstrap `Profile` (PR #3555) are unrelated. This surface is
**Device Type** everywhere. `profile` must not appear in routes, SDK names or
docs for this concept.

### Gateway types are device types

There is no separate gateway-type surface. A gateway is a device with
`is_gateway` ([spec §8 A12](../architecture.md#8-decision-record)), so its type is
an ordinary device type in the same namespace — and a device that both measures
and relays needs *one* type declaring both, not two.

Anything presenting gateway types as a distinct catalogue would be wrong, and
would break the concentrator-meter case the capability model exists to allow.

## Acceptance criteria

1. Create a device type with measurements and commands; read it back with the
   declaration intact.
2. Create a device bound to it; conforming attributes succeed.
3. Non-conforming attributes are rejected with an error naming the offending
   field.
4. Create version 2; devices on version 1 continue working unchanged.
5. Deprecate version 1; new bindings are refused, existing ones unaffected.
6. `GET /devices?device_type_id=` returns exactly the bound devices.
7. Global (tenant-less) types are listed alongside domain types, if exposed at all
   — per the MG-02 decision.
8. Error responses are actionable: field path, expected constraint.
9. No route, field or doc string calls this a "profile".

## Test plan

- Unit: SDK type ↔ capability document mapping; error translation from Atom's
  `jsonschema` output to a field-level API error.
- Integration: full lifecycle including versioning and deprecation.
- Criterion 3 with several violation shapes — wrong type, missing required,
  out-of-range — asserting each produces a usable message.
- API-shape test for criterion 9.

## Risks

- **Schema-error legibility.** Atom surfaces raw `jsonschema` crate output.
  Untranslated, an operator sees a JSON pointer and a validator name. Translation
  is core to this PRD, not polish.
- **Attribute-vs-telemetry validation confusion** is the most likely user
  misunderstanding of the whole feature. Documentation is a deliverable here.
- **Capability model expressiveness.** Measurements and commands cover the
  watermeter case. Multi-channel devices, nested structures and enumerated
  states may not fit. Validate the model against two or three additional real
  device types before freezing it — the API is hard to widen after 1.0.
