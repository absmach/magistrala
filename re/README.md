# Rules Engine

The Magistrala Rules Engine (RE) processes incoming messages using user-defined scripts (Lua or Go) and routes the results to outputs such as channels, alarms, email, SenML writers, PostgreSQL, or Slack. It also supports scheduled rule execution and publishes rule events to the event store.

## Configuration

The service is configured using the following environment variables (values shown are from [docker/.env](https://github.com/absmach/magistrala/blob/main/docker/.env) as consumed by [docker/docker-compose.yaml](https://github.com/absmach/magistrala/blob/main/docker/docker-compose.yaml)):

### Core service

| Variable | Description | Default |
| --- | --- | --- |
| `MG_RE_LOG_LEVEL` | Log level for the service | `debug` |
| `MG_RE_HTTP_HOST` | HTTP host to bind | `re` |
| `MG_RE_HTTP_PORT` | HTTP port to bind | `9008` |
| `MG_RE_HTTP_SERVER_CERT` | Path to PEM-encoded HTTPS server certificate | "" |
| `MG_RE_HTTP_SERVER_KEY` | Path to PEM-encoded HTTPS server key | "" |
| `MG_RE_INSTANCE_ID` | Instance ID for tracing/health | "" |
| `SMQ_MESSAGE_BROKER_URL` | Internal message broker URL | `nats://nats:4222` |
| `SMQ_ES_URL` | Event store broker URL | `nats://nats:4222` |
| `SMQ_JAEGER_URL` | Jaeger collector endpoint | `http://jaeger:4318/v1/traces` |
| `SMQ_JAEGER_TRACE_RATIO` | Trace sampling ratio | `1.0` |
| `SMQ_SEND_TELEMETRY` | Send telemetry to Magistrala call-home server | `true` |

### Database

| Variable | Description | Default |
| --- | --- | --- |
| `MG_RE_DB_HOST` | PostgreSQL host | `re-db` |
| `MG_RE_DB_PORT` | PostgreSQL port | `5432` |
| `MG_RE_DB_USER` | PostgreSQL user | `magistrala` |
| `MG_RE_DB_PASS` | PostgreSQL password | `magistrala` |
| `MG_RE_DB_NAME` | PostgreSQL database name | `rules_engine` |
| `MG_RE_DB_SSL_MODE` | PostgreSQL SSL mode | `disable` |
| `MG_RE_DB_SSL_CERT` | PostgreSQL SSL client cert | "" |
| `MG_RE_DB_SSL_KEY` | PostgreSQL SSL client key | "" |
| `MG_RE_DB_SSL_ROOT_CERT` | PostgreSQL SSL root cert | "" |

### Auth and domains gRPC

| Variable | Description | Default |
| --- | --- | --- |
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

### Readers gRPC

| Variable | Description | Default |
| --- | --- | --- |
| `MG_TIMESCALE_READER_GRPC_URL` | Readers gRPC endpoint | `timescale-reader:7011` |
| `MG_TIMESCALE_READER_GRPC_TIMEOUT` | Readers gRPC timeout | `300s` |
| `MG_TIMESCALE_READER_GRPC_CLIENT_CERT` | Readers gRPC client cert path | `${GRPC_MTLS:+./ssl/certs/reader-grpc-client.crt}` |
| `MG_TIMESCALE_READER_GRPC_CLIENT_CA_CERTS` | Readers gRPC server CA path | `${GRPC_MTLS:+./ssl/certs/ca.crt}` |
| `MG_TIMESCALE_READER_GRPC_CLIENT_KEY` | Readers gRPC client key path | `${GRPC_MTLS:+./ssl/certs/readers-grpc-client.key}` |

### Email

| Variable | Description | Default |
| --- | --- | --- |
| `MG_EMAIL_HOST` | SMTP host | `smtp.mailtrap.io` |
| `MG_EMAIL_PORT` | SMTP port | `2525` |
| `MG_EMAIL_USERNAME` | SMTP username | `18bf7f70705139` |
| `MG_EMAIL_PASSWORD` | SMTP password | `2b0d302e775b1e` |
| `MG_EMAIL_FROM_ADDRESS` | Sender email address | `from@example.com` |
| `MG_EMAIL_FROM_NAME` | Sender display name | `Example` |
| `MG_EMAIL_TEMPLATE` | Email template path | `email.tmpl` |
| `MG_RE_EMAIL_TEMPLATE` | Template file mounted by Docker Compose | `re.tmpl` |

### Callout

| Variable | Description | Default |
| --- | --- | --- |
| `MG_RE_CALLOUT_URLS` | Callout target URLs | "" |
| `MG_RE_CALLOUT_METHOD` | Callout HTTP method | `POST` |
| `MG_RE_CALLOUT_TLS_VERIFICATION` | TLS verification for callout | `false` |
| `MG_RE_CALLOUT_TIMEOUT` | Callout timeout | `10s` |
| `MG_RE_CALLOUT_CA_CERT` | Callout CA cert path | "" |
| `MG_RE_CALLOUT_CERT` | Callout client cert path | "" |
| `MG_RE_CALLOUT_KEY` | Callout client key path | "" |
| `MG_RE_CALLOUT_OPERATIONS` | Callout operations filter | "" |

### Optional cache defaults (from code)

| Variable | Description | Default |
| --- | --- | --- |
| `MG_RE_CACHE_URL` | Cache URL | `redis://localhost:6379/0` |
| `MG_RE_CACHE_KEY_DURATION` | Cache key TTL | `10m` |

## Features

- **Rule execution**: Runs Lua or Go scripts for incoming messages.
- **Multiple outputs**: Channels, alarms, email, SenML writers, remote PostgreSQL, and Slack outputs.
- **Scheduling**: Runs rules at specific times with recurring intervals.
- **Filtering and matching**: Input channel filtering and NATS-style topic matching (`*`, `>`).
- **Observability**: `/metrics` Prometheus endpoint and Jaeger tracing support.
- **Payload limit**: Messages over 100 kB are rejected for processing.

## Architecture

### Runtime flow

1. The service subscribes to all internal broker messages.
2. For each message, it lists enabled rules for the same domain and input channel.
3. It matches the rule `input_topic` against the message subtopic using NATS-style wildcards.
4. The rule logic (Lua or Go) is executed and the result is passed to configured outputs.

### Message payloads

In Lua, the engine injects a global `message` object:

```lua
message = {
  domain = "domain_id",
  channel = "channel_id",
  subtopic = "subtopic",
  publisher = "client_id",
  protocol = "nats",
  created = timestamp,
  payload = { ... } -- JSON object/array or a byte array if payload is not JSON
}
```

For Go scripts, the message is exposed as `messaging/m.message` and `main.logicFunction` must return a value.

In rule definitions, `logic.type` uses numeric values: `0` = Lua, `1` = Go.

If a script returns `false`, outputs are skipped.

### Scheduling

The scheduler runs on a 30-second ticker and selects enabled rules with a due time (`time`) earlier than now. It updates the next due time using `Schedule.NextDue()` and executes each rule with a synthetic message containing the scheduled timestamp.

Recurring types are: `none`, `hourly`, `daily`, `weekly`, `monthly`. The `recurring_period` controls the interval (1 = every interval, 2 = every second interval, etc.).

### Outputs

Supported output types (`outputs.OutputType`) and their fields:

| Output type | Fields | Notes |
| --- | --- | --- |
| `channels` | `channel`, `topic` | Republish result to another channel/topic. |
| `alarms` | none | Emits alarms from the script result. |
| `save_senml` | none | Forwards SenML to writers. |
| `email` | `to`, `subject`, `content` | `content` is a Go template. |
| `save_remote_pg` | `host`, `port`, `user`, `password`, `database`, `table`, `mapping` | `mapping` is a Go template that must render a JSON object. |
| `slack` | `token`, `channel_id`, `message` | `message` is a Go template. |

Templates receive a `Message` (the incoming message) and a `Result` (the script output) value.

## Data model

### Rules table

Defined in `re/postgres/init.go`:

| Column | Type | Description |
| --- | --- | --- |
| `id` | `VARCHAR(36)` | Rule UUID (primary key) |
| `name` | `VARCHAR(1024)` | Rule name |
| `domain_id` | `VARCHAR(36)` | Domain ID |
| `metadata` | `JSONB` | Custom metadata |
| `tags` | `TEXT[]` | Rule tags |
| `created_by` | `VARCHAR(254)` | Creator user ID |
| `created_at` | `TIMESTAMP` | Creation timestamp |
| `updated_at` | `TIMESTAMP` | Last update timestamp |
| `updated_by` | `VARCHAR(254)` | Last updater user ID |
| `input_channel` | `VARCHAR(36)` | Input channel ID |
| `input_topic` | `TEXT` | Input topic (supports wildcards) |
| `outputs` | `JSONB` | Output definitions |
| `status` | `SMALLINT` | 0 = enabled, 1 = disabled, 2 = deleted |
| `logic_type` | `SMALLINT` | 0 = Lua, 1 = Go |
| `logic_value` | `BYTEA` | Script body |
| `start_datetime` | `TIMESTAMP` | Schedule start time |
| `time` | `TIMESTAMP` | Next scheduled execution time |
| `recurring` | `SMALLINT` | Recurring type |
| `recurring_period` | `SMALLINT` | Recurring period |

## Deployment

### Build and run locally

```bash
make re

MG_RE_LOG_LEVEL=debug \
MG_RE_HTTP_PORT=9008 \
MG_RE_DB_HOST=localhost \
MG_RE_DB_PORT=5432 \
MG_RE_DB_USER=magistrala \
MG_RE_DB_PASS=magistrala \
MG_RE_DB_NAME=rules_engine \
SMQ_MESSAGE_BROKER_URL=nats://localhost:4222 \
SMQ_ES_URL=nats://localhost:4222 \
SMQ_AUTH_GRPC_URL=localhost:7001 \
SMQ_AUTH_GRPC_TIMEOUT=300s \
SMQ_DOMAINS_GRPC_URL=localhost:7003 \
SMQ_DOMAINS_GRPC_TIMEOUT=300s \
MG_TIMESCALE_READER_GRPC_URL=localhost:7011 \
MG_TIMESCALE_READER_GRPC_TIMEOUT=300s \
./build/re
```

### Docker Compose

The service is available as a Docker container. Refer to [docker/docker-compose.yaml](https://github.com/absmach/magistrala/blob/main/docker/docker-compose.yaml) for the `re` and `re-db` services and their environment variables. For a full local stack, ensure auth, domains, readers, and the message broker are running.

```bash
docker compose -f docker/docker-compose.yaml up re re-db
```

### Health check

```bash
curl -X GET http://localhost:9008/health \
  -H "accept: application/health+json"
```

## Testing

```bash
go test ./re/...
```

## Usage

The Rules Engine service supports the following operations:

| Operation | Method & Path | Description |
| --- | --- | --- |
| `createRule` | `POST /{domainID}/rules` | Create a new rule |
| `listRules` | `GET /{domainID}/rules` | List rules with filters |
| `viewRule` | `GET /{domainID}/rules/{ruleID}` | Retrieve a rule |
| `updateRule` | `PATCH /{domainID}/rules/{ruleID}` | Update a rule |
| `updateRuleTags` | `PATCH /{domainID}/rules/{ruleID}/tags` | Update rule tags |
| `updateRuleSchedule` | `PATCH /{domainID}/rules/{ruleID}/schedule` | Update rule schedule |
| `enableRule` | `POST /{domainID}/rules/{ruleID}/enable` | Enable a rule |
| `disableRule` | `POST /{domainID}/rules/{ruleID}/disable` | Disable a rule |
| `removeRule` | `DELETE /{domainID}/rules/{ruleID}` | Delete a rule |
| `health` | `GET /health` | Service health check |

List filters: `offset`, `limit`, `name`, `input_channel`, `status`, `order` (`name`, `created_at`, `updated_at`), `dir` (`asc`, `desc`), and `tag`.

### Example: Create a rule (Lua + alarms + channels)

```bash
curl -X POST http://localhost:9008/<domainID>/rules \
  -H "Authorization: Bearer <your_access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "High Temperature Alert",
    "input_channel": "sensors",
    "input_topic": "temperature.*",
    "logic": {
      "type": 0,
      "value": "if message.payload.t > 30 then return {measurement=\"temperature\", value=tostring(message.payload.t), unit=\"C\", threshold=\"30\", cause=\"temp high\", severity=90} end"
    },
    "outputs": [
      { "type": "alarms" },
      { "type": "channels", "channel": "alerts", "topic": "temperature" }
    ],
    "tags": ["temp", "alerts"],
    "metadata": { "site": "lab" }
  }'
```

### Example: List rules

```bash
curl -X GET "http://localhost:9008/<domainID>/rules?status=enabled&input_channel=sensors&order=updated_at&dir=desc&tag=temp" \
  -H "Authorization: Bearer <your_access_token>"
```

### Example: Update rule tags

```bash
curl -X PATCH http://localhost:9008/<domainID>/rules/<ruleID>/tags \
  -H "Authorization: Bearer <your_access_token>" \
  -H "Content-Type: application/json" \
  -d '{ "tags": ["temp", "critical"] }'
```

### Example: Update rule schedule

```bash
curl -X PATCH http://localhost:9008/<domainID>/rules/<ruleID>/schedule \
  -H "Authorization: Bearer <your_access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "schedule": {
      "start_datetime": "2025-01-01T00:00:00Z",
      "time": "2025-01-01T00:00:00Z",
      "recurring": "hourly",
      "recurring_period": 1
    }
  }'
```

### Example: Enable a rule

```bash
curl -X POST http://localhost:9008/<domainID>/rules/<ruleID>/enable \
  -H "Authorization: Bearer <your_access_token>"
```

### Example: Delete a rule

```bash
curl -X DELETE http://localhost:9008/<domainID>/rules/<ruleID> \
  -H "Authorization: Bearer <your_access_token>"
```

For an in-depth explanation of our Rules Engine Service, see the [official documentation][doc].

[doc]: https://docs.magistrala.absmach.eu/dev-guide/rules-engine/
