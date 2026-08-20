# Magistrala CLI

The Magistrala CLI is backed by Atom GraphQL. It talks to the Atom GraphQL endpoint directly instead of the removed legacy HTTP management APIs.

## Build

From the project root:

```bash
make cli
```

The binary is written to `build/cli`.

## Configuration

The CLI reuses the Atom variables the rest of the deployment already sets, so a
shell that has sourced `docker/.env` is configured:

| Variable             | Purpose                                             |
| -------------------- | --------------------------------------------------- |
| `ATOM_URL`           | Atom base URL; the endpoint is this plus `/graphql` |
| `ATOM_GRAPHQL_URL`   | Full GraphQL endpoint, overrides `ATOM_URL`         |
| `ATOM_SERVICE_TOKEN` | Bearer token, falling back to `ATOM_ADMIN_TOKEN`    |
| `ATOM_TIMEOUT`       | Request timeout, `5s` by default                    |

Without `ATOM_URL` the CLI defaults to `http://localhost:8080/graphql`.

Both settings also have flags, which win over the environment:

```bash
./build/cli --graphql-url http://localhost:8080/graphql --token "$ATOM_ADMIN_TOKEN" domains list
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
./build/cli login user@example.com secret --tenant-id <domain_id>
./build/cli login user@example.com secret --tenant-alias <domain_alias>
```

## Domains

Domains map to Atom tenants.

```bash
./build/cli domains create "Demo Domain" --alias demo
./build/cli domains list --limit 20 --offset 0
./build/cli domains get <domain_id>
```

## Clients

Clients map to Atom entities. The default entity kind is `device`; the other
`EntityKind` values are `human`, `service`, `workload` and `application`. Pass
an empty `--kind` to list every kind.

```bash
./build/cli clients create <domain_id> "Thermostat 1"
./build/cli clients create <domain_id> "Gateway" --kind application --attributes '{"site":"lab"}'
./build/cli clients list <domain_id>
./build/cli clients list <domain_id> --kind ""
./build/cli clients get <client_id>
```

## Channels

Channels map to Atom resources with `kind="channel"`.

```bash
./build/cli channels create <domain_id> "Measurements"
./build/cli channels create <domain_id> "Alerts" --attributes '{"retention":"7d"}'
./build/cli channels list <domain_id>
./build/cli channels list <domain_id> --kind ""
./build/cli channels get <channel_id>
```

## Groups

```bash
./build/cli groups create <domain_id> "Factory Floor" --type object --description "Plant devices"
./build/cli groups list <domain_id>
./build/cli groups get <group_id>
```

## Authorization

```bash
./build/cli authz check <subject_id> read --object-kind resource --object-id <resource_id>
./build/cli authz check <subject_id> publish --resource-id <channel_id>
```

## Devices, gateways and device types

These commands go through `pkg/atom`'s typed client rather than the raw
GraphQL path the commands above use. Devices are Atom entities of kind
`device`, so `devices all get <domain_id>` and `clients list <domain_id>`
return the same objects.

```bash
./build/cli devices create <JSON_device> <domain_id>
./build/cli devices all get <domain_id>
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
./build/cli gateways <gateway_id> devices <domain_id>
```

```bash
./build/cli devicetypes create <JSON_device_type> <domain_id>
./build/cli devicetypes all get <domain_id>
./build/cli devicetypes <device_type_id> update <JSON_string>
./build/cli devicetypes <device_type_id> versions
./build/cli devicetypes <device_type_id> create-version <JSON_version>
./build/cli devicetypes <device_type_id> active-version
./build/cli devicetypes <device_type_id> bind <device_id> [version_id]
```

## Health

`health` is the one command still served by the per-service HTTP APIs rather
than by Atom. It covers `certs` and `fluxmq`, and reads `MG_CERTS_URL` and
`MG_HTTP_ADAPTER_URL`.

```bash
./build/cli health certs
```

All management commands except `login` and `health` require a bearer token
through `--token`, `ATOM_SERVICE_TOKEN`, or `ATOM_ADMIN_TOKEN`.
