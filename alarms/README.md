# Alarms

The Alarms service stores, manages and exposes alarms raised by rules and device activity. It consumes alarm events from the message broker, persists them to PostgreSQL, and provides an HTTP API for listing, viewing, updating, and deleting alarms with full authn/authz, metrics, and tracing support.

## Configuration

The service is configured using the following environment variables (values shown are from [docker/.env](https://github.com/absmach/magistrala/blob/main/docker/.env) as consumed by [docker/docker-compose.yaml](https://github.com/absmach/magistrala/blob/main/docker/docker-compose.yaml)):

| Variable | Description | Default |
| --- | --- | --- |
| `MG_ALARMS_LOG_LEVEL` | Log level for the service | `debug` |
| `MG_ALARMS_HTTP_HOST` | HTTP host to bind | `alarms` |
| `MG_ALARMS_HTTP_PORT` | HTTP port to bind | `8050` |
| `MG_ALARMS_HTTP_SERVER_CERT` | Path to PEM-encoded HTTPS server certificate | "" |
| `MG_ALARMS_HTTP_SERVER_KEY` | Path to PEM-encoded HTTPS server key | "" |
| `MG_ALARMS_DB_HOST` | PostgreSQL host | `alarms-db` |
| `MG_ALARMS_DB_PORT` | PostgreSQL port | `5432` |
| `MG_ALARMS_DB_USER` | PostgreSQL user | `magistrala` |
| `MG_ALARMS_DB_PASS` | PostgreSQL password | `magistrala` |
| `MG_ALARMS_DB_NAME` | PostgreSQL database name | `alarms` |
| `MG_ALARMS_DB_SSL_MODE` | PostgreSQL SSL mode | `disable` |
| `MG_ALARMS_DB_SSL_CERT` | PostgreSQL SSL client cert | "" |
| `MG_ALARMS_DB_SSL_KEY` | PostgreSQL SSL client key | "" |
| `MG_ALARMS_DB_SSL_ROOT_CERT` | PostgreSQL SSL root cert | "" |
| `MG_ALARMS_INSTANCE_ID` | Instance ID for tracing/health | "" |
| `SMQ_MESSAGE_BROKER_URL` | Message broker URL for alarm ingestion | `nats://nats:4222` |
| `SMQ_JAEGER_URL` | Jaeger collector endpoint | `http://jaeger:4318/v1/traces` |
| `SMQ_JAEGER_TRACE_RATIO` | Trace sampling ratio | `1.0` |
| `SMQ_AUTH_GRPC_URL` | Auth gRPC endpoint | `auth:7001` |
| `SMQ_AUTH_GRPC_TIMEOUT` | Auth gRPC timeout | `300s` |
| `SMQ_AUTH_GRPC_CLIENT_CERT` | Auth gRPC client cert path | `${GRPC_MTLS:+./ssl/certs/auth-grpc-client.crt}` |
| `SMQ_AUTH_GRPC_CLIENT_KEY` | Auth gRPC client key path | `${GRPC_MTLS:+./ssl/certs/auth-grpc-client.key}` |
| `SMQ_AUTH_GRPC_SERVER_CA_CERTS` | Auth gRPC server CA path | `${GRPC_MTLS:+./ssl/certs/ca.crt}` |
| `SMQ_DOMAINS_GRPC_URL` | Domains gRPC endpoint | `domains:7003` |
| `SMQ_DOMAINS_GRPC_TIMEOUT` | Domains gRPC timeout | `300s` |
| `SMQ_DOMAINS_GRPC_CLIENT_CERT` | Domains gRPC client cert path | `${GRPC_MTLS:+./ssl/certs/domains-grpc-client.crt}` |
| `SMQ_DOMAINS_GRPC_CLIENT_KEY` | Domains gRPC client key path | `${GRPC_MTLS:+./ssl/certs/domains-grpc-client.key}` |
| `SMQ_DOMAINS_GRPC_SERVER_CA_CERTS` | Domains gRPC server CA path | `${GRPC_MTLS:+./ssl/certs/ca.crt}` |
| `SMQ_ALLOW_UNVERIFIED_USER` | Allow unverified users to access | `true` |

## Features

- **Alarm ingestion**: Consumes alarms from the message broker and persists them to PostgreSQL.
- **Stateful updates**: Updates assignee, acknowledgment, resolution, and metadata fields.
- **Filtering and paging**: Lists alarms by domain, rule, channel, client, subtopic, status, severity, and time range.
- **Observability**: `/metrics` Prometheus endpoint and Jaeger tracing support.
- **Auth and authorization**: Authn/authz enforced via gRPC auth and domains services.

## Architecture

### Runtime flow

1. The message broker publishes alarm events under the `alarms.>` subject.
2. The Alarms consumer decodes the event payload, enriches it with message metadata, validates it, and calls `CreateAlarm`.
3. The repository writes to PostgreSQL while deduplicating repeated active alarms with the same severity.
4. The HTTP API exposes list/view/update/delete operations with authn/authz, metrics, and tracing middleware.

### Components

- **HTTP API**: `alarms/api` exposes REST endpoints and health/metrics handlers.
- **Service layer**: `alarms/service.go` validates requests and coordinates repository operations.
- **Repository**: `alarms/postgres/alarms.go` implements persistence and filtering.
- **Consumer**: `alarms/consumer` processes broker messages and creates alarms.
- **Message broker**: `alarms/brokers` uses NATS JetStream with stream `alarms` and subject `alarms.>`.
- **Migrations**: `alarms/postgres/init.go` defines the alarms schema and indexes.

### Alarms table

Defined in `alarms/postgres/init.go`:

| Column | Type | Description |
| --- | --- | --- |
| `id` | `VARCHAR(36)` | Alarm UUID (primary key) |
| `rule_id` | `VARCHAR(36)` | Rule ID that triggered the alarm |
| `domain_id` | `VARCHAR(36)` | Domain ID |
| `channel_id` | `VARCHAR(36)` | Channel ID |
| `subtopic` | `TEXT` | Subtopic associated with the alarm |
| `client_id` | `VARCHAR(36)` | Client ID |
| `measurement` | `TEXT` | Measurement name |
| `value` | `TEXT` | Measured value |
| `unit` | `TEXT` | Measurement unit |
| `threshold` | `TEXT` | Threshold value |
| `cause` | `TEXT` | Cause/description |
| `status` | `SMALLINT` | 0 = active, 1 = cleared |
| `severity` | `SMALLINT` | Severity (0-100) |
| `assignee_id` | `VARCHAR(36)` | Assignee ID |
| `created_at` | `TIMESTAMPTZ` | Creation timestamp |
| `updated_at` | `TIMESTAMPTZ` | Last update timestamp |
| `updated_by` | `VARCHAR(36)` | User who updated |
| `assigned_at` | `TIMESTAMPTZ` | When assigned |
| `assigned_by` | `VARCHAR(36)` | Who assigned |
| `acknowledged_at` | `TIMESTAMPTZ` | When acknowledged |
| `acknowledged_by` | `VARCHAR(36)` | Who acknowledged |
| `resolved_at` | `TIMESTAMPTZ` | When resolved |
| `resolved_by` | `VARCHAR(36)` | Who resolved |
| `metadata` | `JSONB` | Custom metadata |

Index: `idx_alarms_state (domain_id, rule_id, channel_id, subtopic, client_id, measurement, created_at DESC)`

## Deployment

### Build and run locally

```bash
make alarms

MG_ALARMS_LOG_LEVEL=debug \
MG_ALARMS_HTTP_PORT=8050 \
MG_ALARMS_DB_HOST=localhost \
MG_ALARMS_DB_PORT=5432 \
MG_ALARMS_DB_USER=magistrala \
MG_ALARMS_DB_PASS=magistrala \
MG_ALARMS_DB_NAME=alarms \
SMQ_MESSAGE_BROKER_URL=nats://localhost:4222 \
SMQ_AUTH_GRPC_URL=localhost:7001 \
SMQ_AUTH_GRPC_TIMEOUT=300s \
SMQ_DOMAINS_GRPC_URL=localhost:7003 \
SMQ_DOMAINS_GRPC_TIMEOUT=300s \
./build/alarms
```

### Docker Compose

The service is available as a Docker container. Refer to [docker/docker-compose.yaml](https://github.com/absmach/magistrala/blob/main/docker/docker-compose.yaml) for the `alarms` and `alarms-db` services and their environment variables. For a full local stack, make sure the auth, domains, and message broker services are also running.

```bash
docker compose -f docker/docker-compose.yaml up alarms alarms-db
```

### Health check

```bash
curl -X GET http://localhost:8050/health \
  -H "accept: application/health+json"
```

## Testing

```bash
go test ./alarms/...
```

## Usage

The Alarms service supports the following operations:

| Operation | Method & Path | Description |
| --- | --- | --- |
| `listAlarms` | `GET /{domainID}/alarms` | List alarms with filters |
| `viewAlarm` | `GET /{domainID}/alarms/{alarmID}` | Retrieve a single alarm |
| `updateAlarm` | `PUT /{domainID}/alarms/{alarmID}` | Update alarm status/assignee/metadata |
| `deleteAlarm` | `DELETE /{domainID}/alarms/{alarmID}` | Delete an alarm |
| `health` | `GET /health` | Service health check |

Alarm creation is driven by message broker events and is not exposed as an HTTP endpoint.

### Example: List alarms

```bash
curl -X GET "http://localhost:8050/<domainID>/alarms?limit=10&offset=0&status=active&severity=50" \
  -H "Authorization: Bearer <your_access_token>"
```

### Example: View an alarm

```bash
curl -X GET http://localhost:8050/<domainID>/alarms/<alarmID> \
  -H "Authorization: Bearer <your_access_token>"
```

### Example: Update an alarm

```bash
curl -X PUT http://localhost:8050/<domainID>/alarms/<alarmID> \
  -H "Authorization: Bearer <your_access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "cleared",
    "assignee_id": "<userID>",
    "severity": 40,
    "metadata": { "note": "cleared after inspection" }
  }'
```

### Example: Delete an alarm

```bash
curl -X DELETE http://localhost:8050/<domainID>/alarms/<alarmID> \
  -H "Authorization: Bearer <your_access_token>"
```

### Example: Health check

```bash
curl -X GET http://localhost:8050/health \
  -H "accept: application/health+json"
```
