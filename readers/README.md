# Readers

Readers expose HTTP and gRPC APIs for retrieving stored messages. They read normalized SenML records (and optional JSON payloads) from storage backends and apply authn/authz checks via the Auth, Clients, and Channels services. Magistrala provides two reader services: `postgres-reader` and `timescale-reader`.

## Configuration

Readers are optional services. Values shown are from [docker/.env](https://github.com/absmach/magistrala/blob/main/docker/.env) and the corresponding add-on compose files in `docker/addons/*-reader/docker-compose.yaml`.

### Postgres reader

#### Service endpoints

| Variable | Description | Default |
| --- | --- | --- |
| `MG_POSTGRES_READER_LOG_LEVEL` | Service log level | `debug` |
| `MG_POSTGRES_READER_HTTP_HOST` | HTTP host | `postgres-reader` |
| `MG_POSTGRES_READER_HTTP_PORT` | HTTP port | `9009` |
| `MG_POSTGRES_READER_HTTP_SERVER_CERT` | HTTPS server certificate path | "" |
| `MG_POSTGRES_READER_HTTP_SERVER_KEY` | HTTPS server key path | "" |
| `MG_POSTGRES_READER_GRPC_HOST` | gRPC host | `postgres-reader` |
| `MG_POSTGRES_READER_GRPC_PORT` | gRPC port | `7009` |
| `MG_POSTGRES_READER_GRPC_SERVER_CERT` | gRPC server cert path | `${GRPC_MTLS:+./ssl/certs/readers-grpc-server.crt}${GRPC_TLS:+./ssl/certs/readers-grpc-server.crt}` |
| `MG_POSTGRES_READER_GRPC_SERVER_KEY` | gRPC server key path | `${GRPC_MTLS:+./ssl/certs/readers-grpc-server.key}${GRPC_TLS:+./ssl/certs/readers-grpc-server.key}` |
| `MG_POSTGRES_READER_GRPC_SERVER_CA_CERTS` | gRPC server CA certs path | `${GRPC_MTLS:+./ssl/certs/ca.crt}${GRPC_TLS:+./ssl/certs/ca.crt}` |
| `MG_POSTGRES_READER_INSTANCE_ID` | Instance ID | "" |

#### Database

| Variable | Description | Default |
| --- | --- | --- |
| `MG_POSTGRES_HOST` | PostgreSQL host | `postgres` |
| `MG_POSTGRES_PORT` | PostgreSQL port | `5432` |
| `MG_POSTGRES_USER` | PostgreSQL user | `supermq` |
| `MG_POSTGRES_PASS` | PostgreSQL password | `supermq` |
| `MG_POSTGRES_NAME` | PostgreSQL database name | `messages` |
| `MG_POSTGRES_SSL_MODE` | PostgreSQL SSL mode | `disable` |
| `MG_POSTGRES_SSL_CERT` | PostgreSQL SSL client cert | "" |
| `MG_POSTGRES_SSL_KEY` | PostgreSQL SSL client key | "" |
| `MG_POSTGRES_SSL_ROOT_CERT` | PostgreSQL SSL root cert | "" |

#### Dependencies

| Variable | Description | Default |
| --- | --- | --- |
| `SMQ_AUTH_GRPC_URL` | Auth gRPC URL | `auth:7001` |
| `SMQ_AUTH_GRPC_TIMEOUT` | Auth gRPC timeout | `300s` |
| `SMQ_AUTH_GRPC_CLIENT_CERT` | Auth gRPC client cert | `${GRPC_MTLS:+./ssl/certs/auth-grpc-client.crt}` |
| `SMQ_AUTH_GRPC_CLIENT_KEY` | Auth gRPC client key | `${GRPC_MTLS:+./ssl/certs/auth-grpc-client.key}` |
| `SMQ_AUTH_GRPC_CLIENT_CA_CERTS` | Auth gRPC CA certs | `${GRPC_MTLS:+./ssl/certs/ca.crt}` |
| `SMQ_CLIENTS_GRPC_URL` | Clients gRPC URL | `clients:7006` |
| `SMQ_CLIENTS_GRPC_TIMEOUT` | Clients gRPC timeout | `300s` |
| `SMQ_CLIENTS_GRPC_CLIENT_CERT` | Clients gRPC client cert | `${GRPC_MTLS:+./ssl/certs/clients-grpc-client.crt}` |
| `SMQ_CLIENTS_GRPC_CLIENT_KEY` | Clients gRPC client key | `${GRPC_MTLS:+./ssl/certs/clients-grpc-client.key}` |
| `SMQ_CLIENTS_GRPC_CLIENT_CA_CERTS` | Clients gRPC CA certs | `${GRPC_MTLS:+./ssl/certs/ca.crt}` |
| `SMQ_CHANNELS_GRPC_URL` | Channels gRPC URL | `channels:7005` |
| `SMQ_CHANNELS_GRPC_TIMEOUT` | Channels gRPC timeout | `300s` |
| `SMQ_CHANNELS_GRPC_CLIENT_CERT` | Channels gRPC client cert | `${GRPC_MTLS:+./ssl/certs/channels-grpc-client.crt}` |
| `SMQ_CHANNELS_GRPC_CLIENT_KEY` | Channels gRPC client key | `${GRPC_MTLS:+./ssl/certs/channels-grpc-client.key}` |
| `SMQ_CHANNELS_GRPC_CLIENT_CA_CERTS` | Channels gRPC CA certs | `${GRPC_MTLS:+./ssl/certs/ca.crt}` |
| `SMQ_SEND_TELEMETRY` | Send telemetry to call-home server | `true` |

Note: When running the postgres reader binary directly, configuration is read from `SMQ_POSTGRES_*` and `SMQ_POSTGRES_READER_HTTP_*` prefixes (see `cmd/postgres-reader/main.go` and `readers/postgres/README.md`). The Docker add-on uses `MG_POSTGRES_*`/`MG_POSTGRES_READER_*` values.

### Timescale reader

#### Service endpoints

| Variable | Description | Default |
| --- | --- | --- |
| `MG_TIMESCALE_READER_LOG_LEVEL` | Service log level | `debug` |
| `MG_TIMESCALE_READER_HTTP_HOST` | HTTP host | `timescale-reader` |
| `MG_TIMESCALE_READER_HTTP_PORT` | HTTP port | `9011` |
| `MG_TIMESCALE_READER_HTTP_SERVER_CERT` | HTTPS server certificate path | "" |
| `MG_TIMESCALE_READER_HTTP_SERVER_KEY` | HTTPS server key path | "" |
| `MG_TIMESCALE_READER_GRPC_HOST` | gRPC host | `timescale-reader` |
| `MG_TIMESCALE_READER_GRPC_PORT` | gRPC port | `7011` |
| `MG_TIMESCALE_READER_GRPC_SERVER_CERT` | gRPC server cert path | `${GRPC_MTLS:+./ssl/certs/readers-grpc-server.crt}${GRPC_TLS:+./ssl/certs/readers-grpc-server.crt}` |
| `MG_TIMESCALE_READER_GRPC_SERVER_KEY` | gRPC server key path | `${GRPC_MTLS:+./ssl/certs/readers-grpc-server.key}${GRPC_TLS:+./ssl/certs/readers-grpc-server.key}` |
| `MG_TIMESCALE_READER_GRPC_SERVER_CA_CERTS` | gRPC server CA certs path | `${GRPC_MTLS:+./ssl/certs/ca.crt}${GRPC_TLS:+./ssl/certs/ca.crt}` |
| `MG_TIMESCALE_READER_INSTANCE_ID` | Instance ID | "" |

#### Database

| Variable | Description | Default |
| --- | --- | --- |
| `MG_TIMESCALE_HOST` | TimescaleDB host | `timescale` |
| `MG_TIMESCALE_PORT` | TimescaleDB port | `5432` |
| `MG_TIMESCALE_USER` | TimescaleDB user | `supermq` |
| `MG_TIMESCALE_PASS` | TimescaleDB password | `supermq` |
| `MG_TIMESCALE_NAME` | TimescaleDB database name | `supermq` |
| `MG_TIMESCALE_SSL_MODE` | TimescaleDB SSL mode | `disable` |
| `MG_TIMESCALE_SSL_CERT` | TimescaleDB SSL client cert | "" |
| `MG_TIMESCALE_SSL_KEY` | TimescaleDB SSL client key | "" |
| `MG_TIMESCALE_SSL_ROOT_CERT` | TimescaleDB SSL root cert | "" |

#### Dependencies

Timescale reader uses the same gRPC dependency variables listed for the Postgres reader (`SMQ_AUTH_GRPC_*`, `SMQ_CLIENTS_GRPC_*`, `SMQ_CHANNELS_GRPC_*`) and `SMQ_SEND_TELEMETRY`.

## Features

- **Message retrieval**: Read SenML messages by channel with paging and filters.
- **Flexible filters**: Subtopic, publisher, protocol, name, numeric/string/data/bool values, and time range.
- **Aggregation**: Timescale reader supports `aggregation` + `interval` (requires `from` and `to`).
- **Multiple formats**: Use `format=messages` for SenML or another table name for JSON payloads.
- **Authn/authz**: Supports user tokens or thing keys; enforces channel access.
- **Observability**: `/metrics` Prometheus endpoint and `/health` checks.

## Architecture

### Runtime flow

1. Client calls `GET /{domainID}/channels/{chanID}/messages` with a user token or thing key.
2. The service authenticates the caller and authorizes channel access via gRPC (Auth, Clients, Channels).
3. The reader repository builds a filtered SQL query and reads from storage.
4. Results are returned as a paged list of SenML messages or JSON payloads.

### Components

- **HTTP API**: `readers/api/http` exposes the messages endpoint, health, and metrics.
- **gRPC API**: `readers/api/grpc` exposes the readers service for internal use.
- **Repositories**: `readers/postgres` and `readers/timescale` implement storage access.
- **Middleware**: `readers/middleware` adds logging and metrics.

### Messages table (SenML)

Defined in `readers/postgres/init.go` and consumed by both readers:

| Column | Type | Description |
| --- | --- | --- |
| `id` | `UUID` | Message ID (primary key) |
| `channel` | `UUID` | Channel ID |
| `subtopic` | `VARCHAR(254)` | Subtopic |
| `publisher` | `UUID` | Publisher (client) ID |
| `protocol` | `TEXT` | Protocol name |
| `name` | `TEXT` | SenML name |
| `unit` | `TEXT` | SenML unit |
| `value` | `FLOAT` | Numeric value |
| `string_value` | `TEXT` | String value |
| `bool_value` | `BOOL` | Boolean value |
| `data_value` | `TEXT` | Data value (base64 or raw string) |
| `sum` | `FLOAT` | Sum value |
| `time` | `FLOAT` | Measurement time |
| `update_time` | `FLOAT` | Update time |

## Deployment

### Build and run locally

Postgres reader:

```bash
make postgres-reader

SMQ_POSTGRES_READER_LOG_LEVEL=info \
SMQ_POSTGRES_READER_HTTP_PORT=9009 \
SMQ_POSTGRES_HOST=localhost \
SMQ_POSTGRES_PORT=5432 \
SMQ_POSTGRES_USER=supermq \
SMQ_POSTGRES_PASS=supermq \
SMQ_POSTGRES_NAME=messages \
SMQ_AUTH_GRPC_URL=localhost:7001 \
SMQ_CLIENTS_GRPC_URL=localhost:7006 \
SMQ_CHANNELS_GRPC_URL=localhost:7005 \
./build/postgres-reader
```

Timescale reader:

```bash
make timescale-reader

MG_TIMESCALE_READER_LOG_LEVEL=info \
MG_TIMESCALE_READER_HTTP_PORT=9011 \
MG_TIMESCALE_HOST=localhost \
MG_TIMESCALE_PORT=5432 \
MG_TIMESCALE_USER=supermq \
MG_TIMESCALE_PASS=supermq \
MG_TIMESCALE_NAME=supermq \
SMQ_AUTH_GRPC_URL=localhost:7001 \
SMQ_CLIENTS_GRPC_URL=localhost:7006 \
SMQ_CHANNELS_GRPC_URL=localhost:7005 \
./build/timescale-reader
```

### Docker Compose

Postgres reader add-on:

```bash
docker compose -f docker/docker-compose.yaml -f docker/addons/postgres-reader/docker-compose.yaml up
```

Timescale reader add-on:

```bash
docker compose -f docker/docker-compose.yaml -f docker/addons/timescale-reader/docker-compose.yaml up
```

## Testing

```bash
go test ./readers/...
```

## Usage

The Readers service supports the following operations (see `apidocs/openapi/readers.yaml`):

| Operation | Method & Path | Description |
| --- | --- | --- |
| `getMessages` | `GET /{domainID}/channels/{chanID}/messages` | List messages with filters |
| `health` | `GET /health` | Service health check |

Supported query parameters include:

`limit`, `offset`, `format`, `subtopic`, `publisher`, `protocol`, `name`, `v`, `comparator`, `vs`, `vd`, `vb`, `from`, `to`, `aggregation`, `interval`, `order`, `dir`.

Comparator usage (for `vs`/`vd`):

| Comparator | Usage | Example |
| --- | --- | --- |
| `eq` | Equal to the query | `eq["active"] -> "active"` |
| `ge` | Substrings of the query | `ge["tiv"] -> "active", "tiv"` |
| `gt` | Substrings excluding exact match | `gt["tiv"] -> "active"` |
| `le` | Superstrings of the query | `le["active"] -> "tiv"` |
| `lt` | Superstrings excluding exact match | `lt["active"] -> "active", "tiv"` |

### Example: List messages

```bash
curl -X GET "http://localhost:9009/<domainID>/channels/<channelID>/messages?limit=10&offset=0&subtopic=s1&name=temp&v=21.5&comparator=ge" \
  -H "Authorization: Bearer <your_access_token>"
```

### Example: Aggregate messages (Timescale)

```bash
curl -X GET "http://localhost:9011/<domainID>/channels/<channelID>/messages?aggregation=avg&interval=10s&from=1709218556069&to=1709218757503" \
  -H "Authorization: Thing <thing_key>"
```

### Example: Health check

```bash
curl -X GET http://localhost:9009/health \
  -H "accept: application/health+json"
```

For an in-depth explanation of Readers Service, see the [official documentation][doc].

[doc]: https://docs.magistrala.absmach.eu/dev-guide/readers/
