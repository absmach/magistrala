# Alarms

The Alarms service stores, manages and exposes alarms raised by rules and device activity. It consumes alarm events from the message broker, stores alarms as Atom resources, and provides an HTTP API for listing, viewing, updating, and deleting alarms with full authn/authz, metrics, and tracing support.

## Configuration

The service is configured using the following environment variables (values shown are from [docker/.env](https://github.com/absmach/magistrala/blob/main/docker/.env) as consumed by [docker/docker-compose.yaml](https://github.com/absmach/magistrala/blob/main/docker/docker-compose.yaml)):

| Variable | Description | Default |
| --- | --- | --- |
| `MG_ALARMS_LOG_LEVEL` | Log level for the service | `debug` |
| `MG_ALARMS_HTTP_HOST` | HTTP host to bind | `alarms` |
| `MG_ALARMS_HTTP_PORT` | HTTP port to bind | `8050` |
| `MG_ALARMS_HTTP_SERVER_CERT` | Path to PEM-encoded HTTPS server certificate | "" |
| `MG_ALARMS_HTTP_SERVER_KEY` | Path to PEM-encoded HTTPS server key | "" |
| `MG_ALARMS_INSTANCE_ID` | Instance ID for tracing/health | "" |
| `MG_MESSAGE_BROKER_URL` | Message broker URL for alarm ingestion | `nats://nats:4222` |
| `MG_JAEGER_URL` | Jaeger collector endpoint | `http://jaeger:4318/v1/traces` |
| `MG_JAEGER_TRACE_RATIO` | Trace sampling ratio | `1.0` |
| `ATOM_URL` | Atom HTTP endpoint | `http://atom:8080` |
| `ATOM_SERVICE_TOKEN` | Atom service token used by the alarms service. In Docker Compose this is populated from `MG_ATOM_TOKEN_ALARMS`. | "" |
| `ATOM_JWKS_URL` | Atom JWKS endpoint for JWT verification | `http://atom:8080/.well-known/jwks.json` |
| `ATOM_TIMEOUT` | Atom request timeout | `5s` |
| `MG_ALLOW_UNVERIFIED_USER` | Allow unverified users to access | `true` |

## Features

- **Alarm ingestion**: Consumes alarms from the message broker and persists them as Atom resources with `kind=alarm`.
- **Stateful updates**: Updates assignee, acknowledgment, resolution, and metadata fields.
- **Filtering and paging**: Lists alarms by domain, rule, channel, client, subtopic, status, severity, and time range.
- **Observability**: `/metrics` Prometheus endpoint and Jaeger tracing support.
- **Auth and authorization**: Authn/authz enforced through Atom JWT verification and PDP checks.

## Architecture

### Runtime flow

1. The message broker publishes alarm events under the `alarms.>` subject.
2. The Alarms consumer decodes the event payload, enriches it with message metadata, validates it, and calls `CreateAlarm`.
3. The repository writes to Atom resources while deduplicating repeated active alarms with the same severity.
4. The HTTP API exposes list/view/update/delete operations with authn/authz, metrics, and tracing middleware.

### Components

- **HTTP API**: `alarms/api` exposes REST endpoints and health/metrics handlers.
- **Service layer**: `alarms/service.go` validates requests and coordinates repository operations.
- **Repository**: `alarms/atom_repository.go` stores alarms as Atom resources and applies alarm filtering.
- **Consumer**: `alarms/consumer` processes broker messages and creates alarms.
- **Message broker**: `alarms/brokers` uses NATS JetStream with stream `alarms` and subject `alarms.>`.
- **Atom**: stores alarms as `resources` rows with `kind=alarm`.

### Atom Resource Mapping

The service stores each alarm as an Atom resource:

| Alarm field | Atom resource |
| --- | --- |
| `id` | `resources.id` and `resources.name` |
| `domain_id` | `resources.tenant_id` |
| `assignee_id` | `resources.owner_id` on create and `attributes.assignee_id` |
| `status` | `attributes.status` and `attributes.alarm_status` |
| `metadata` | `attributes.metadata` |
| alarm-specific fields | `attributes.rule_id`, `attributes.channel_id`, `attributes.client_id`, `attributes.subtopic`, `attributes.measurement`, `attributes.value`, `attributes.unit`, `attributes.threshold`, `attributes.cause`, `attributes.severity`, and lifecycle fields |

## Deployment

### Build and run locally

```bash
make alarms

MG_ALARMS_LOG_LEVEL=debug \
MG_ALARMS_HTTP_PORT=8050 \
MG_MESSAGE_BROKER_URL=nats://localhost:4222 \
ATOM_URL=http://localhost:8080 \
ATOM_SERVICE_TOKEN=<alarms-service-token> \
./build/alarms
```

### Docker Compose

The service is available as a Docker container. Refer to [docker/docker-compose.yaml](https://github.com/absmach/magistrala/blob/main/docker/docker-compose.yaml) for the `alarms` service and its environment variables. For a full local stack, make sure Atom, `atom-bootstrap`, NATS, and nginx are also running.

```bash
docker compose -f docker/docker-compose.yaml up alarms
```

### Health check

```bash
curl -X GET http://localhost:8050/health \
  -H "accept: application/health+json"
```

## Testing

```bash
go test ./alarms ./cmd/alarms
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
