# Magistrala CLI

The Magistrala CLI is backed by Atom GraphQL. It talks to the Atom GraphQL endpoint directly instead of the removed legacy HTTP management APIs.

## Build

From the project root:

```bash
make cli
```

The binary is written to `build/cli`.

## Configuration

The CLI reuses the Atom endpoint and token variables the rest of the deployment
already sets, so a shell that has sourced `docker/.env` is configured for local
GraphQL access:

| Variable              | Purpose                                             |
| --------------------- | --------------------------------------------------- |
| `ATOM_URL`            | Atom base URL; the endpoint is this plus `/graphql` |
| `ATOM_GRAPHQL_URL`    | Full GraphQL endpoint, overrides `ATOM_URL`         |
| `ATOM_SERVICE_TOKEN`  | Bearer token, falling back to `ATOM_ADMIN_TOKEN`    |
| `MG_CLI_ATOM_TIMEOUT` | CLI request timeout; defaults to `90s` when unset   |

Without `ATOM_URL` the CLI defaults to `http://localhost:8080/graphql`. The CLI
does not use the backend service `ATOM_TIMEOUT`; set `MG_CLI_ATOM_TIMEOUT` only
when an interactive command needs a different request timeout.

Both settings also have flags, which win over the environment:

```bash
./build/cli --graphql-url http://localhost:8080/graphql --token "$ATOM_ADMIN_TOKEN" workspaces list
```

`--graphql-url` also configures the `pkg/atom` client behind `devices`,
`gateways` and `devicetypes`: the Atom base URL is the endpoint minus its
trailing `/graphql`.

These persistent flags apply to every listing command:

| Flag | Purpose |
| --- | --- |
| `-l`, `--limit` | page size, `10` by default |
| `-o`, `--offset` | records to skip |
| `-n`, `--name` | name search, used by `devices all get` |
| `-r`, `--raw` | raw output for easier parsing |

## Login

```bash
./build/cli login admin 12345678
```

Tenant-scoped credentials can pass either a tenant ID or alias:

```bash
./build/cli login user@example.com secret --tenant-id <workspace_id>
./build/cli login user@example.com secret --tenant-alias <workspace_alias>
```

## Password

Authenticated users must provide their current password when changing it:

```bash
./build/cli --token "$ATOM_USER_TOKEN" password change <current_password> <new_password>
```

## Workspaces

Workspaces map to Atom tenants.

```bash
./build/cli workspaces create "Demo Workspace" --alias demo
./build/cli workspaces list --limit 20 --offset 0
./build/cli workspaces get <workspace_id>
```

## Channels

Channels map to Atom resources with `kind="channel"`.

```bash
./build/cli channels create <workspace_id> "Measurements"
./build/cli channels create <workspace_id> "Alerts" --attributes '{"retention":"7d"}'
./build/cli channels list <workspace_id>
./build/cli channels list <workspace_id> --kind ""
./build/cli channels get <channel_id>
```

## Groups

Atom exposes one mutation per group type, so `--type` selects between
`createObjectGroup` and `createPrincipalGroup`. It defaults to `object`, and
any other value is rejected before the request is sent.

```bash
./build/cli groups create <workspace_id> "Factory Floor" --description "Plant devices"
./build/cli groups create <workspace_id> "Operators" --type principal
./build/cli groups list <workspace_id>
./build/cli groups get <group_id>
```

## Authorization

```bash
./build/cli authz check <subject_id> read --object-kind resource --object-id <resource_id>
./build/cli authz check <subject_id> publish --resource-id <channel_id>
```

## Devices, gateways and device types

These commands go through `pkg/atom`'s typed client. Devices are Atom entities
of kind `device`.

```bash
./build/cli devices create <JSON_device> <workspace_id>
./build/cli devices all get <workspace_id>
./build/cli devices <device_id> get
./build/cli devices <device_id> update <JSON_string>
./build/cli devices <device_id> enable
./build/cli devices <device_id> disable
./build/cli devices <device_id> delete
```

A gateway is a device with `attributes.is_gateway` set; `gateways` manages the
reachability relation between devices and gateways.

```bash
./build/cli gateways set <device_id> <gateway_id1,gateway_id2,...>
./build/cli gateways <gateway_id> devices <workspace_id>
```

In raw mode, successful mutating commands that otherwise print `ok` return a
JSON object:

```json
{"status":"ok"}
```

```bash
./build/cli devicetypes create <JSON_device_type> <workspace_id>
./build/cli devicetypes all get <workspace_id>
./build/cli devicetypes <device_type_id> update <JSON_string>
./build/cli devicetypes <device_type_id> versions
./build/cli devicetypes <device_type_id> create-version <JSON_version>
./build/cli devicetypes <device_type_id> active-version
./build/cli devicetypes <device_type_id> bind <device_id> [version_id]
```

## Health

`health` is served by the per-service HTTP APIs rather than by Atom. It covers
the FluxMQ HTTP adapter and reads `MG_HTTP_ADAPTER_URL`, defaulting to the local
Compose nginx route at `http://localhost/http`.

```bash
./build/cli health fluxmq
```

All management commands except `login` and `health` require a bearer token
through `--token`, `ATOM_SERVICE_TOKEN`, or `ATOM_ADMIN_TOKEN`.
