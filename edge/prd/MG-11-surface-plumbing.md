# MG-11 — CLI, PAT scopes, permissions and OpenAPI

| | |
|---|---|
| **Repo** | `absmach/magistrala` (Go) |
| **Priority** | P4 |
| **Depends on** | MG-09 |
| **Blocks** | — |
| **Status** | Draft |

## Problem

MG-09 and MG-10 introduce Device, Gateway and Device Type. Four surfaces still
speak `client` and would leave the 1.0 API internally inconsistent: the CLI,
personal access token scopes, the permission matrix, and the published API docs.

Mechanical work, but two parts are easy to get wrong in ways that break
deployments.

## Scope

**In scope**

- CLI commands for devices and gateways. (Device types are phase 2, with MG-10.)
- PAT `EntityType` scopes.
- `docker/permission.yaml` entity blocks.
- OpenAPI specs.

**Out of scope**

- Model or SDK changes — MG-09/10.
- The missing CLI binary (see Risks).
- Channel role inconsistency (see Risks).

## Design

### CLI

Add `cli/devices.go`, `cli/gateways.go`, `cli/devicetypes.go`. This is **net-new,
not a rewrite** — `cli/clients.go` was deleted in `16ba29cf4` along with
`users.go`, `groups.go` and `domains.go`.

Follow the shape of the surviving `cli/channels.go:35`.

Gateway commands must include setting a device's gateways and listing a
gateway's declared devices — that is where the CLI earns its keep for field work.
`cli/gateways.go` is a thin view over devices filtered by `is_gateway`, not a
separate entity surface.

### PAT scopes

`auth/pat.go:73-99` declares `EntityType` as a positional `iota` enum. An earlier
draft of this PRD claimed renumbering would invalidate issued tokens.
**That was wrong — `EntityType` is never persisted or transmitted as a number:**

| Path | Representation | Evidence |
|---|---|---|
| Database | `entity_type VARCHAR(50)`, stores the name | `auth/postgres/init.go:97`; written `repo.go:426`, read `repo.go:554` |
| JSON | name | `auth/pat.go:155-164` |
| Text | name | `auth/pat.go:166-174` |
| gRPC | `string entity_type = 6` | `internal/proto/auth/v1/auth.proto:47` |

Per [spec §8 C2](../architecture.md#8-decision-record):

1. **Remove `ClientsType` outright**, along with `ClientsScopeStr`, its `String()`
   case (`:101-126`) and its `ParseEntityType` case (`:128-153`). Renumbering is
   safe.
2. Add `DevicesType`. (`DeviceTypesType` waits for MG-10, phase 2.) **No `GatewaysType`** — a gateway
   *is* a device ([spec §8 A12](../architecture.md#8-decision-record)), so a
   device-scoped PAT already covers it. A separate scope would imply a separate
   population and be wrong the moment one device is both.
3. **Pin explicit values** instead of `iota`, so ordering never becomes
   load-bearing by accident later.
4. Update `IsValidOperationForEntity` (`:176-181`), which enumerates
   `ClientsType` today.

**Remaining work:** existing `pat_scopes` rows with `entity_type = 'clients'`
will fail `ParseEntityType` once the constant is gone. Migrate them or drop them
— a clients-scoped PAT *should* stop working once clients no longer exist.

### Permissions

`docker/permission.yaml:4-32` — replace the `clients:` block with `devices:`.
(`device_types:` waits for MG-10, phase 2.) **No `gateways:` block:** a gateway is a device, so device
permissions already govern it. Adding one would fragment permissions across a
population that is not disjoint.

The reachability relation needs one new operation on devices:

```yaml
devices:
  operations:
    - set_gateways: update_permission     # or its own permission
```

Decide whether declaring a device's gateways is an ordinary update or warrants
its own permission. Its own is probably right — re-pointing a device's gateways
changes how its data reaches the platform, which is a different act from renaming
it, and least-privilege grants should be able to separate them.

`pkg/permissions/entities.go` is config-driven with string entity keys, so no
code change is needed for new entity types.

### OpenAPI

`apidocs/openapi/clients.yaml` → `devices.yaml`. (`device-types.yaml` is phase 2.)
`/gateways` routes are documented inside `devices.yaml`, since they return
devices filtered by `is_gateway` rather than a distinct resource. Update the
aggregate reference in
`apidocs/openapi/README.md`.

## Acceptance criteria

1. CLI can create, list, view, update, enable/disable and delete devices.
2. CLI can create a device with `is_gateway`, set a device's gateways, and list
   the devices declared on a gateway.
3. ⏸ *Phase 2* — CLI can manage device types and versions.
4. PATs scoped to `devices` authorize device operations and nothing else — and
   cover gateways, since a gateway is a device.
5. A PAT issued before the change, carrying a `bootstrap` or `domains` scope,
   still authorizes the same operations after the enum is renumbered — proving
   ordering is not load-bearing.
6. `permission.yaml` validates at startup (`pkg/permissions` rejects unknown
   entity types).
7. `set_gateways` is independently grantable from ordinary device update.
8. OpenAPI specs validate and match implemented routes.
9. No surface refers to `clients`.

## Test plan

- CLI: command-level tests following the existing `cli/` patterns.
- PAT: round-trip every `EntityType` through `String()` / `ParseEntityType` /
  JSON / text marshalling — the enum has four representations
  (`auth/pat.go:155-164`, `:257-320`) and they must agree.
- **Token-compatibility test**: decode a token fixture issued before the change
  and assert the documented behaviour.
- Permissions: service startup with the new file; assert an unknown entity type
  is rejected.
- OpenAPI: spec linting plus a route-coverage check against the router.

## Risks

- **Stale `pat_scopes` rows.** Not the enum ordering — that concern was
  unfounded — but rows holding `entity_type = 'clients'` will fail to parse.
  Migrate or drop them as part of this PR, not afterwards.
- **The CLI has no binary.** `cmd/cli/main.go` was deleted in `16ba29cf4`, so
  `cli` is an unimported library. These commands ship unreachable unless the
  binary is restored — out of scope here, but it makes acceptance criteria 1–3
  testable only at package level. Flag it; do not silently expand scope.
- **Channels have no roles** while devices, groups and domains do
  (`pkg/sdk/channels.go` has no role methods). Pre-existing and out of scope, but
  1.0 freezes it. Worth a decision before release.
