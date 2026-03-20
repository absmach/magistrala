# Writers

Writers consume messages from the message broker, normalize them (SenML or JSON), and persist them to a storage backend. Magistrala provides two writer services:

- **Postgres writer**: Stores data in PostgreSQL.
- **Timescale writer**: Stores data in TimescaleDB and uses hypertables for time-series workloads.

Writers are optional services and are treated as plugins. Core services and the message broker must be running first. For platform dependencies, see [Docker Compose](https://github.com/absmach/magistrala/blob/main/docker/docker-compose.yaml).

## Configuration

Values shown are from [docker/.env](https://github.com/absmach/magistrala/blob/main/docker/.env) and the add-on compose files in `docker/addons/*-writer/docker-compose.yaml`.

### Postgres writer

#### Postgres Service endpoints

| Variable | Description | Default |
| --- | --- | --- |
| `MG_POSTGRES_WRITER_LOG_LEVEL` | Service log level | `debug` |
| `MG_POSTGRES_WRITER_CONFIG_PATH` | Config file path (subjects/transformer) | `/config.toml` |
| `MG_POSTGRES_WRITER_HTTP_HOST` | HTTP host | `postgres-writer` |
| `MG_POSTGRES_WRITER_HTTP_PORT` | HTTP port | `9007` |
| `MG_POSTGRES_WRITER_HTTP_SERVER_CERT` | HTTPS server certificate path | "" |
| `MG_POSTGRES_WRITER_HTTP_SERVER_KEY` | HTTPS server key path | "" |
| `MG_POSTGRES_WRITER_INSTANCE_ID` | Instance ID | "" |

#### Postgres Database

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

#### Postgres Message broker and observability

| Variable | Description | Default |
| --- | --- | --- |
| `SMQ_MESSAGE_BROKER_URL` | Message broker URL | `nats://nats:4222` |
| `SMQ_JAEGER_URL` | Jaeger collector endpoint | `http://jaeger:4318/v1/traces` |
| `SMQ_JAEGER_TRACE_RATIO` | Trace sampling ratio | `1.0` |
| `SMQ_SEND_TELEMETRY` | Send telemetry to Magistrala call-home server | `true` |

### Timescale writer

#### Timescale Service endpoints

| Variable | Description | Default |
| --- | --- | --- |
| `MG_TIMESCALE_WRITER_LOG_LEVEL` | Service log level | `debug` |
| `MG_TIMESCALE_WRITER_CONFIG_PATH` | Config file path (subjects/transformer) | `/config.toml` |
| `MG_TIMESCALE_WRITER_HTTP_HOST` | HTTP host | `timescale-writer` |
| `MG_TIMESCALE_WRITER_HTTP_PORT` | HTTP port | `9012` |
| `MG_TIMESCALE_WRITER_HTTP_SERVER_CERT` | HTTPS server certificate path | "" |
| `MG_TIMESCALE_WRITER_HTTP_SERVER_KEY` | HTTPS server key path | "" |
| `MG_TIMESCALE_WRITER_INSTANCE_ID` | Instance ID | "" |

#### Timescale Database

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

#### Timescale Message broker and observability

Timescale writer uses the same broker and telemetry variables listed for Postgres writer.

### Writer config file

Both writers read a config file defined by `*_WRITER_CONFIG_PATH`. The default add-on config files are:

- `docker/addons/postgres-writer/config.toml`
- `docker/addons/timescale-writer/config.toml`

The config file controls subscription subjects and, for Postgres, optional transformer settings:

```toml
["subscriber"]
subjects = ["writers.>"]

[transformer]
format = "senml"
content_type = "application/senml+json"
time_fields = [
  { field_name = "seconds_key", field_format = "unix",    location = "UTC" },
  { field_name = "millis_key",  field_format = "unix_ms", location = "UTC" },
  { field_name = "micros_key",  field_format = "unix_us", location = "UTC" },
  { field_name = "nanos_key",   field_format = "unix_ns", location = "UTC" }
]
```

NATS uses subject `writers.>` and RabbitMQ uses routing key `writers.#` (both are handled by `consumers/writers/brokers`).

## Features

- **Message persistence**: Stores incoming SenML messages into PostgreSQL or TimescaleDB.
- **JSON payload support**: Saves JSON payloads into dynamically created tables.
- **Broker-backed ingestion**: Consumes from NATS JetStream or RabbitMQ topics.
- **Configurable subscription**: Limits ingestion to specific `writers.*` subjects.
- **Observability**: Exposes `/health` and `/metrics` endpoints, with Jaeger tracing.

## Architecture

### Runtime flow

1. The message broker publishes messages under `writers.*`.
2. The writer loads `config.toml` to select subjects and transformer settings.
3. The consumer converts messages to SenML or JSON payloads.
4. The repository writes records to the target database.

### Components

- **Message broker adapter**: `consumers/writers/brokers` (NATS JetStream or RabbitMQ).
- **Writer services**: `consumers/writers/postgres` and `consumers/writers/timescale`.
- **HTTP API**: `consumers/writers/api` exposes `/health` and `/metrics`.
- **Migrations**: `consumers/writers/*/init.go` defines the schema and indexes.

### PostgreSQL schema (SenML messages)

Defined in `consumers/writers/postgres/init.go`:

| Column | Type | Description |
| --- | --- | --- |
| `id` | `UUID` | Message ID |
| `channel` | `UUID` | Channel ID |
| `subtopic` | `VARCHAR(254)` | Subtopic |
| `publisher` | `UUID` | Publisher (client) ID |
| `protocol` | `TEXT` | Protocol name |
| `name` | `TEXT` | SenML name |
| `unit` | `TEXT` | SenML unit |
| `value` | `FLOAT` | Numeric value |
| `string_value` | `TEXT` | String value |
| `bool_value` | `BOOL` | Boolean value |
| `data_value` | `BYTEA` | Data value |
| `sum` | `FLOAT` | Sum value |
| `time` | `FLOAT` | Measurement time |
| `update_time` | `FLOAT` | Update time |

Primary key: `(time, publisher, subtopic, name)`

### TimescaleDB schema (SenML messages)

Defined in `consumers/writers/timescale/init.go`:

| Column | Type | Description |
| --- | --- | --- |
| `time` | `BIGINT` | Measurement time |
| `channel` | `UUID` | Channel ID |
| `subtopic` | `VARCHAR(254)` | Subtopic |
| `publisher` | `VARCHAR(254)` | Publisher (client) ID |
| `protocol` | `TEXT` | Protocol name |
| `name` | `VARCHAR(254)` | SenML name |
| `unit` | `TEXT` | SenML unit |
| `value` | `FLOAT` | Numeric value |
| `string_value` | `TEXT` | String value |
| `bool_value` | `BOOL` | Boolean value |
| `data_value` | `BYTEA` | Data value |
| `sum` | `FLOAT` | Sum value |
| `update_time` | `FLOAT` | Update time |

Primary key: `(time, channel, subtopic, protocol, publisher, name)`

Timescale writer creates a hypertable on `messages` and adds time-series indexes for common query paths.

### JSON payload tables (dynamic)

If the transformer emits JSON payloads, the writers create a table named after the payload format:

Postgres JSON table:
`id UUID`, `created BIGINT`, `channel VARCHAR(254)`, `subtopic VARCHAR(254)`, `publisher VARCHAR(254)`, `protocol TEXT`, `payload JSONB` (PK: `id`)

Timescale JSON table:
`created BIGINT`, `channel VARCHAR(254)`, `subtopic VARCHAR(254)`, `publisher VARCHAR(254)`, `protocol TEXT`, `payload JSONB` (PK: `created`, `publisher`, `subtopic`)

## Deployment

### Build and run locally

Postgres writer:

```bash
make postgres-writer

MG_POSTGRES_WRITER_LOG_LEVEL=debug \
MG_POSTGRES_WRITER_CONFIG_PATH=./docker/addons/postgres-writer/config.toml \
MG_POSTGRES_WRITER_HTTP_PORT=9007 \
MG_POSTGRES_HOST=localhost \
MG_POSTGRES_PORT=5432 \
MG_POSTGRES_USER=supermq \
MG_POSTGRES_PASS=supermq \
MG_POSTGRES_NAME=messages \
SMQ_MESSAGE_BROKER_URL=nats://localhost:4222 \
SMQ_JAEGER_URL=http://localhost:4318/v1/traces \
./build/postgres-writer
```

Timescale writer:

```bash
make timescale-writer

MG_TIMESCALE_WRITER_LOG_LEVEL=debug \
MG_TIMESCALE_WRITER_CONFIG_PATH=./docker/addons/timescale-writer/config.toml \
MG_TIMESCALE_WRITER_HTTP_PORT=9012 \
MG_TIMESCALE_HOST=localhost \
MG_TIMESCALE_PORT=5432 \
MG_TIMESCALE_USER=supermq \
MG_TIMESCALE_PASS=supermq \
MG_TIMESCALE_NAME=supermq \
SMQ_MESSAGE_BROKER_URL=nats://localhost:4222 \
SMQ_JAEGER_URL=http://localhost:4318/v1/traces \
./build/timescale-writer
```

### Docker Compose

Postgres writer add-on:

```bash
docker compose -f docker/docker-compose.yaml -f docker/addons/postgres-writer/docker-compose.yaml up
```

Timescale writer add-on:

```bash
docker compose -f docker/docker-compose.yaml -f docker/addons/timescale-writer/docker-compose.yaml up
```

### Health check

```bash
curl -X GET http://localhost:9007/health \
  -H "accept: application/health+json"
```

## Testing

```bash
go test ./consumers/writers/...
```

## Usage

Writers do not expose a message ingestion API. Messages are written via the message broker. The HTTP API provides only health and metrics endpoints.

| Endpoint | Description |
| --- | --- |
| `GET /health` | Service health check |
| `GET /metrics` | Prometheus metrics |

For an in-depth explanation of Writers, see the [official documentation][doc].

[doc]: https://docs.magistrala.absmach.eu/dev-guide/consumers/
