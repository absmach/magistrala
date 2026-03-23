# HTTP Adapter

The HTTP Adapter exposes HTTP endpoints for publishing messages and WebSocket capabilities for publishing and subscribing to messages from SuperMQ channels. It authenticates clients via tokens or Basic auth, resolves domains/channels over gRPC, and forwards payloads to the message broker.

For more on SuperMQ, see the [official documentation][doc].

## Configuration

Environment variables (unset values fall back to defaults):

| Variable                              | Description                                          | Default                        |
| ------------------------------------- | ---------------------------------------------------- | ------------------------------ |
| `MG_HTTP_ADAPTER_LOG_LEVEL`          | Log level (debug, info, warn, error)                 | debug                          |
| `MG_HTTP_ADAPTER_HOST`               | HTTP Adapter host                                    | http-adapter                   |
| `MG_HTTP_ADAPTER_PORT`               | HTTP Adapter port                                    | 8008                           |
| `MG_HTTP_ADAPTER_SERVER_CERT`        | Path to PEM-encoded server certificate (enables TLS) | ""                             |
| `MG_HTTP_ADAPTER_SERVER_KEY`         | Path to PEM-encoded server key                       | ""                             |
| `MG_HTTP_ADAPTER_SERVER_CA_CERTS`    | Trusted CA bundle for HTTPS server                   | ""                             |
| `MG_HTTP_ADAPTER_CLIENT_CA_CERTS`    | Client CA bundle to require mTLS on HTTPS server     | ""                             |
| `MG_HTTP_ADAPTER_CACHE_NUM_COUNTERS` | Cache counters for topic parsing                     | 200000                         |
| `MG_HTTP_ADAPTER_CACHE_MAX_COST`     | Maximum cache size (bytes)                           | 1048576                        |
| `MG_HTTP_ADAPTER_CACHE_BUFFER_ITEMS` | Cache buffer items                                   | 64                             |
| `MG_MESSAGE_BROKER_URL`              | Message broker URL (publishing target)               | nats://nats:4222               |
| `MG_ES_URL`                          | Event store URL (publishing middleware)              | nats://nats:4222               |
| `MG_JAEGER_URL`                      | Jaeger tracing endpoint                              | <http://jaeger:4318/v1/traces> |
| `MG_JAEGER_TRACE_RATIO`              | Trace sampling ratio                                 | 1.0                            |
| `MG_SEND_TELEMETRY`                  | Send telemetry to SuperMQ call-home server           | true                           |
| `MG_HTTP_ADAPTER_INSTANCE_ID`        | Service instance ID (auto-generated when empty)      | ""                             |
| `MG_CLIENTS_GRPC_URL`                | Clients service gRPC URL                             | clients:7006                   |
| `MG_CLIENTS_GRPC_TIMEOUT`            | Clients gRPC request timeout                         | 300s                           |
| `MG_CLIENTS_GRPC_CLIENT_CERT`        | Clients gRPC client certificate                      | ""                             |
| `MG_CLIENTS_GRPC_CLIENT_KEY`         | Clients gRPC client key                              | ""                             |
| `MG_CLIENTS_GRPC_SERVER_CA_CERTS`    | Clients gRPC trusted CA bundle                       | ""                             |
| `MG_CHANNELS_GRPC_URL`               | Channels service gRPC URL                            | channels:7005                  |
| `MG_CHANNELS_GRPC_TIMEOUT`           | Channels gRPC request timeout                        | 300s                           |
| `MG_CHANNELS_GRPC_CLIENT_CERT`       | Channels gRPC client certificate                     | ""                             |
| `MG_CHANNELS_GRPC_CLIENT_KEY`        | Channels gRPC client key                             | ""                             |
| `MG_CHANNELS_GRPC_SERVER_CA_CERTS`   | Channels gRPC trusted CA bundle                      | ""                             |
| `MG_DOMAINS_GRPC_URL`                | Domains service gRPC URL                             | domains:7003                   |
| `MG_DOMAINS_GRPC_TIMEOUT`            | Domains gRPC request timeout                         | 300s                           |
| `MG_DOMAINS_GRPC_CLIENT_CERT`        | Domains gRPC client certificate                      | ""                             |
| `MG_DOMAINS_GRPC_CLIENT_KEY`         | Domains gRPC client key                              | ""                             |
| `MG_DOMAINS_GRPC_SERVER_CA_CERTS`    | Domains gRPC trusted CA bundle                       | ""                             |
| `MG_AUTH_GRPC_URL`                   | Auth service gRPC URL                                | auth:7001                      |
| `MG_AUTH_GRPC_TIMEOUT`               | Auth service gRPC request timeout                    | 300s                           |
| `MG_AUTH_GRPC_CLIENT_CERT`           | Auth gRPC client certificate                         | ""                             |
| `MG_AUTH_GRPC_CLIENT_KEY`            | Auth gRPC client key                                 | ""                             |
| `MG_AUTH_GRPC_SERVER_CA_CERTS`       | Auth gRPC trusted CA bundle                          | ""                             |

## Deployment

The adapter is shipped as a Docker container. See the [`http-adapter` section](https://github.com/absmach/supermq/blob/main/docker/docker-compose.yaml#L1226-L1365) of `docker-compose.yaml` for deployment details.

To build and run locally:

```bash
# download the latest version of the service
git clone https://github.com/absmach/supermq
cd supermq

# compile the http adapter
make http

# copy binary to $GOBIN
make install

# set the environment variables and run the service
MG_HTTP_ADAPTER_LOG_LEVEL=debug \
MG_HTTP_ADAPTER_HOST=http-adapter \
MG_HTTP_ADAPTER_PORT=8008 \
MG_HTTP_ADAPTER_SERVER_CERT="" \
MG_HTTP_ADAPTER_SERVER_KEY="" \
MG_HTTP_ADAPTER_CACHE_NUM_COUNTERS=200000 \
MG_HTTP_ADAPTER_CACHE_MAX_COST=1048576 \
MG_HTTP_ADAPTER_CACHE_BUFFER_ITEMS=64 \
MG_MESSAGE_BROKER_URL=nats://nats:4222 \
MG_ES_URL=nats://nats:4222 \
MG_JAEGER_URL=<http://jaeger:4318/v1/traces> \
MG_JAEGER_TRACE_RATIO=1.0 \
MG_CLIENTS_GRPC_URL=clients:7006 \
MG_CLIENTS_GRPC_TIMEOUT=300s \
MG_CLIENTS_GRPC_CLIENT_CERT="" \
MG_CLIENTS_GRPC_CLIENT_KEY="" \
MG_CLIENTS_GRPC_SERVER_CA_CERTS="" \
MG_CHANNELS_GRPC_URL=channels:7005 \
MG_CHANNELS_GRPC_TIMEOUT=300s \
MG_CHANNELS_GRPC_CLIENT_CERT="" \
MG_CHANNELS_GRPC_CLIENT_KEY="" \
MG_CHANNELS_GRPC_SERVER_CA_CERTS="" \
MG_DOMAINS_GRPC_URL=domains:7003 \
MG_DOMAINS_GRPC_TIMEOUT=300s \
MG_DOMAINS_GRPC_CLIENT_CERT="" \
MG_DOMAINS_GRPC_CLIENT_KEY="" \
MG_DOMAINS_GRPC_SERVER_CA_CERTS="" \
MG_AUTH_GRPC_URL=auth:7001 \
MG_AUTH_GRPC_TIMEOUT=300s \
MG_AUTH_GRPC_CLIENT_CERT="" \
MG_AUTH_GRPC_CLIENT_KEY="" \
MG_AUTH_GRPC_SERVER_CA_CERTS="" \
MG_SEND_TELEMETRY=true \
MG_HTTP_ADAPTER_INSTANCE_ID="" \
$GOBIN/supermq-http
```

TLS is enabled by setting `MG_HTTP_ADAPTER_SERVER_CERT` and `MG_HTTP_ADAPTER_SERVER_KEY`. mTLS is enabled when `MG_HTTP_ADAPTER_CLIENT_CA_CERTS` is provided. gRPC client TLS/mTLS is enabled by setting the corresponding client cert/key/CA variables.

## Usage

Endpoints:

- `POST /m/{domain}/c/{channel}` (and wildcard `/m/{domain}/c/{channel}/*`): publish a message.
- `POST /hc/{domain}`: health-check message path (authenticated).
- `GET /health`: service health probe.
- `GET /metrics`: Prometheus metrics.

Authentication:

- Bearer token in `Authorization` header, or
- Basic auth where the password is the token (username ignored).

Supported content types: `application/json`, `application/senml+json`, `application/senml+cbor`.

Example publish:

```bash
curl -X POST http://localhost:8008/m/<domainID>/c/<channelID>/sub/topic \
  -H "Authorization: Bearer <client_token>" \
  -H "Content-Type: application/json" \
  -d '{ "temp": 22.5, "unit": "C" }'
```

## Implementation Details

- Publishes to the configured message broker (`MG_MESSAGE_BROKER_URL`) with optional event-store middleware (`MG_ES_URL`).
- Resolves domains and channels over gRPC to validate/route topics; authenticates via Auth gRPC; validates client identity via Clients gRPC.
- Topic parsing is cached (Ristretto) with configurable counters/cost/buffers to reduce resolver calls.
- Observability: Jaeger tracing, Prometheus metrics at `/metrics`, service health at `/health`.
- Optional call-home telemetry is enabled by default.

## Best Practices

- Use domain/channel routes consistently in publish paths; include subtopics to segment data.
- Keep cache defaults unless load patterns require tuning; monitor `/metrics` for cache hit ratios.
- Enable TLS/mTLS for production deployments (HTTP server and gRPC clients).
- Reuse a single broker URL across services (often NATS) to simplify operations.

## Versioning and Health Check

The adapter exposes `/health` with status and build metadata.

```bash
curl -X GET http://localhost:8008/health \
  -H "accept: application/health+json"
```

Example response:

```json
{
  "status": "pass",
  "version": "0.18.0",
  "commit": "7d6f4dc4f7f0c1fa3dc24eddfb18bb5073ff4f62",
  "description": "http adapter",
  "build_time": "1970-01-01_00:00:00"
}
```

For endpoint details, see the [HTTP Adapter API documentation](https://docs.api.supermq.absmach.eu/?urls.primaryName=http.yaml).

[doc]: https://docs.supermq.absmach.eu/
