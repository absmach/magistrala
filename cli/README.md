# Magistrala CLI

The Magistrala CLI is backed by Atom GraphQL. It talks to the Atom GraphQL endpoint directly instead of the removed legacy HTTP management APIs.

## Build

From the project root:

```bash
make cli
```

The binary is written to `build/cli`.

## Configuration

The CLI defaults to `http://localhost:8080/graphql`.

You can configure the endpoint and token with flags:

```bash
./build/cli --graphql-url http://localhost:8080/graphql --token "$MG_ATOM_TOKEN" domains list
```

Or with environment variables:

```bash
export MG_ATOM_GRAPHQL_URL=http://localhost:8080/graphql
export MG_ATOM_TOKEN=<token>
```

`MAGISTRALA_TOKEN` is also accepted as a token fallback.

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

Clients map to Atom entities. The default entity kind is `device`.

```bash
./build/cli clients create <domain_id> "Thermostat 1"
./build/cli clients create <domain_id> "Gateway" --kind application --attributes '{"site":"lab"}'
./build/cli clients list <domain_id>
./build/cli clients get <client_id>
```

## Channels

Channels map to Atom resources with `kind="channel"`.

```bash
./build/cli channels create <domain_id> "Measurements"
./build/cli channels create <domain_id> "Alerts" --attributes '{"retention":"7d"}'
./build/cli channels list <domain_id>
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

All management commands except `login` require a bearer token through `--token`, `MG_ATOM_TOKEN`, or `MAGISTRALA_TOKEN`.
