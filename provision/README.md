# Provision service

Provision service provides an HTTP API to create initial SuperMQ resources for gateways or edge deployments. It can create clients and channels based on a configurable layout, optionally create bootstrap configurations, whitelist clients, and issue X.509 certificates for mTLS.

For gateways to communicate with [SuperMQ][supermq], configuration is required (MQTT host, client, channels, certificates). A gateway can fetch bootstrap configuration from the [Bootstrap][bootstrap] service using its `<external_id>` and `<external_key>`. The [Agent][agent] service is typically used on gateways to retrieve that configuration.

You can create bootstrap configuration directly via [Bootstrap][bootstrap] or through Provision. [SuperMQ UI][mgxui] uses the Bootstrap service; Provision is intended to automate gateway setups where one physical gateway may require multiple clients and channels (for example, [Agent][agent] and [Export][export]). This setup is defined as a **provision layout**.

## Configuration

The service is configured using environment variables and/or a TOML config file. Defaults below are from `provision/config.go`. Docker add-on examples are in `docker/addons/provision/docker-compose.yaml` and `docker/.env`. The binary reads `SMQ_PROVISION_*` variables; the add-on compose file uses `MG_PROVISION_*`, so ensure the container receives the expected names.

### Core service

| Variable | Description | Default |
| --- | --- | --- |
| `SMQ_PROVISION_HTTP_PORT` | Provision service listening port | `9016` |
| `SMQ_PROVISION_LOG_LEVEL` | Service log level | `info` |
| `SMQ_PROVISION_ENV_CLIENTS_TLS` | SDK TLS verification | `false` |
| `SMQ_PROVISION_SERVER_CERT` | HTTPS server certificate | "" |
| `SMQ_PROVISION_SERVER_KEY` | HTTPS server key | "" |
| `SMQ_SEND_TELEMETRY` | Send telemetry to SuperMQ call-home server | `true` |
| `SMQ_MQTT_ADAPTER_INSTANCE_ID` | Instance ID used in health output | "" |

### SuperMQ endpoints and credentials

| Variable | Description | Default |
| --- | --- | --- |
| `SMQ_PROVISION_USERS_LOCATION` | Users service URL | `http://localhost` |
| `SMQ_PROVISION_CLIENTS_LOCATION` | Clients service URL | `http://localhost` |
| `SMQ_PROVISION_CERTS_LOCATION` | Certs service URL (certs SDK) | `http://localhost` |
| `SMQ_PROVISION_BS_SVC_URL` | Bootstrap service URL | `http://localhost:9000` |
| `SMQ_PROVISION_CERTS_SVC_URL` | Certs service URL (Magistrala SDK) | `http://localhost:9019` |
| `SMQ_PROVISION_USERNAME` | SuperMQ username | `user` |
| `SMQ_PROVISION_PASS` | SuperMQ password | `test` |
| `SMQ_PROVISION_API_KEY` | SuperMQ authentication token | "" |
| `SMQ_PROVISION_EMAIL` | SuperMQ user email | `test@example.com` |
| `SMQ_PROVISION_DOMAIN_ID` | Default domain ID (unused by HTTP API) | "" |

### Provisioning behavior

| Variable | Description | Default |
| --- | --- | --- |
| `SMQ_PROVISION_CONFIG_FILE` | Provision config file | `config.toml` |
| `SMQ_PROVISION_X509_PROVISIONING` | Issue client certificates during provisioning | `false` |
| `SMQ_PROVISION_BS_CONFIG_PROVISIONING` | Save client config in Bootstrap | `true` |
| `SMQ_PROVISION_BS_AUTO_WHITELIST` | Auto-whitelist client | `true` |
| `SMQ_PROVISION_BS_CONTENT` | Bootstrap config content (JSON string) | "" |
| `SMQ_PROVISION_CERTS_HOURS_VALID` | Client cert validity period | `2400h` |

## Features

- **Layout-driven provisioning**: Create clients and channels from a predefined layout.
- **Bootstrap integration**: Create bootstrap configs and optionally whitelist clients.
- **X.509 certificates**: Issue client certificates during provisioning when enabled.
- **Gateway metadata**: Enrich gateway clients with control/data/export channel IDs.
- **Observability**: `/metrics` and `/health` endpoints.

## Provision layout

Provision layout is configured in a TOML file (see `provision/configs/config.toml` or `docker/addons/provision/configs/config.toml`). If the file exists, it is loaded and any missing fields are filled with env values. The layout defines which clients and channels will be created when calling `/mapping`.

Default behavior (when no config file is loaded) creates one client and two channels: `control` and `data`.

Notes:

- At least one client must include `external_id` in metadata. This value is replaced with the `external_id` from the provisioning request and is used for bootstrap creation.
- Channel metadata `type` is reserved for `control`, `data`, and `export` and is used to enrich gateway metadata.
- Bootstrap content can be provided via `bootstrap.content` in the TOML file or as JSON through `SMQ_PROVISION_BS_CONTENT`.

Example layout:

```toml
[[clients]]
  name = "client"

  [clients.metadata]
    external_id = "xxxxxx"

[[channels]]
  name = "control-channel"

  [channels.metadata]
    type = "control"

[[channels]]
  name = "data-channel"

  [channels.metadata]
    type = "data"

[[channels]]
  name = "export-channel"

  [channels.metadata]
    type = "data"
```

## Authentication

Provision uses SuperMQ APIs and requires a valid token. There are three ways to provide it:

- `Authorization: Bearer <token>` on each request.
- `SMQ_PROVISION_API_KEY` in env or TOML (used when no header token is provided).
- `SMQ_PROVISION_USERNAME` and `SMQ_PROVISION_PASS` in env or TOML (used to create an access token when no header token is provided).

`POST /{domainID}/mapping` can create its own token using API key or username/password if no `Authorization` header is provided. The `Authorization` header takes precedence when present. `GET /{domainID}/mapping` always requires a bearer token.

## Architecture

### Runtime flow

1. The service loads configuration from env and optionally merges a config file.
2. `POST /{domainID}/mapping` validates the request and ensures a token exists.
3. Clients are created from the configured layout (external ID is injected into metadata).
4. Channels are created with names prefixed by the request `name`.
5. If enabled, bootstrap configs are created and clients are whitelisted (connected to channels).
6. If X.509 provisioning is enabled, certificates are issued and returned in the response.

## Running

Provision service can be run standalone or via Docker Compose.

Standalone:

```bash
make provision

SMQ_PROVISION_BS_SVC_URL=http://localhost:9013 \
SMQ_PROVISION_CLIENTS_LOCATION=http://localhost:9006 \
SMQ_PROVISION_USERS_LOCATION=http://localhost:9002 \
SMQ_PROVISION_CONFIG_FILE=provision/configs/config.toml \
./build/provision
```

Docker Compose (add-on):

```bash
docker compose -f docker/docker-compose.yaml -f docker/addons/provision/docker-compose.yaml up provision
```

## Usage

The Provision service exposes the following endpoints:

| Operation | Method & Path | Description |
| --- | --- | --- |
| `provision` | `POST /{domainID}/mapping` | Create clients, channels, bootstrap config, and optional certs |
| `mapping` | `GET /{domainID}/mapping` | Return bootstrap content from config |
| `health` | `GET /health` | Service health check |

### Example: Provision a gateway

When credentials are available via env/config, you can omit the `Authorization` header. `Content-Type` must be exactly `application/json`.

```bash
curl -s -S -X POST http://localhost:<SMQ_PROVISION_HTTP_PORT>/<domainID>/mapping \
  -H 'Content-Type: application/json' \
  -d '{"name": "gateway-a", "external_id": "33:52:77:99:43", "external_key": "223334fw2"}'
```

If you want to supply a token explicitly:

```bash
curl -s -S -X POST http://localhost:<SMQ_PROVISION_HTTP_PORT>/<domainID>/mapping \
  -H "Authorization: Bearer <token|api_key>" \
  -H 'Content-Type: application/json' \
  -d '{"name": "gateway-a", "external_id": "<external_id>", "external_key": "<external_key>"}'
```

Response contains created clients, channels, and optional certificate data:

```json
{
  "clients": [
    {
      "id": "c22b0c0f-8c03-40da-a06b-37ed3a72c8d1",
      "name": "client",
      "key": "007cce56-e0eb-40d6-b2b9-ed348a97d1eb",
      "metadata": {
        "external_id": "33:52:79:C3:43"
      }
    }
  ],
  "channels": [
    {
      "id": "064c680e-181b-4b58-975e-6983313a5170",
      "name": "control-channel",
      "metadata": {
        "type": "control"
      }
    },
    {
      "id": "579da92d-6078-4801-a18a-dd1cfa2aa44f",
      "name": "data-channel",
      "metadata": {
        "type": "data"
      }
    }
  ],
  "whitelisted": {
    "c22b0c0f-8c03-40da-a06b-37ed3a72c8d1": true
  }
}
```

### Example: Read bootstrap mapping

```bash
curl -s -S -X GET http://localhost:<SMQ_PROVISION_HTTP_PORT>/<domainID>/mapping \
  -H "Authorization: Bearer <token|api_key>" \
  -H 'Content-Type: application/json'
```

## Certificates

When `SMQ_PROVISION_X509_PROVISIONING=true`, the provisioning flow issues certificates for each client and returns them in the response as `client_cert`, `client_key`, and `ca_cert`. The certificate TTL is controlled by `SMQ_PROVISION_CERTS_HOURS_VALID`.

## Testing

```bash
go test ./provision/...
```

[supermq]: https://github.com/absmach/supermq
[bootstrap]: https://github.com/absmach/supermq/tree/main/bootstrap
[export]: https://github.com/absmach/export
[agent]: https://github.com/absmach/agent
[mgxui]: https://github.com/absmach/supermq/ui
