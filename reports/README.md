# Reports

The Reports service generates time-series reports from stored messages. It fetches data from the readers gRPC service, formats results as JSON, CSV, or PDF, optionally emails the report, and supports scheduled report delivery.

## Configuration

The service is configured using the following environment variables (values shown are from [docker/.env](https://github.com/absmach/magistrala/blob/main/docker/.env) where available, otherwise from service defaults):

### Core service

| Variable | Description | Default |
| --- | --- | --- |
| `MG_REPORTS_LOG_LEVEL` | Log level for the service | `debug` |
| `MG_REPORTS_HTTP_HOST` | HTTP host to bind | `reports` |
| `MG_REPORTS_HTTP_PORT` | HTTP port to bind | `9017` |
| `MG_REPORTS_HTTP_SERVER_CERT` | Path to PEM-encoded HTTPS server certificate | "" |
| `MG_REPORTS_HTTP_SERVER_KEY` | Path to PEM-encoded HTTPS server key | "" |
| `MG_REPORTS_INSTANCE_ID` | Instance ID for tracing/health | "" |
| `SMQ_JAEGER_URL` | Jaeger collector endpoint | `http://jaeger:4318/v1/traces` |
| `SMQ_JAEGER_TRACE_RATIO` | Trace sampling ratio | `1.0` |
| `SMQ_SEND_TELEMETRY` | Send telemetry to Magistrala call-home server | `true` |
| `SMQ_MESSAGE_BROKER_URL` | Message broker URL (parsed, currently unused by reports) | `nats://nats:4222` |
| `SMQ_ES_URL` | Event store URL (parsed, currently unused by reports) | `nats://nats:4222` |

### Database

| Variable | Description | Default |
| --- | --- | --- |
| `MG_REPORTS_DB_HOST` | PostgreSQL host | `reports-db` |
| `MG_REPORTS_DB_PORT` | PostgreSQL port | `5432` |
| `MG_REPORTS_DB_USER` | PostgreSQL user | `magistrala` |
| `MG_REPORTS_DB_PASS` | PostgreSQL password | `magistrala` |
| `MG_REPORTS_DB_NAME` | PostgreSQL database name | `reports` |
| `MG_REPORTS_DB_SSL_MODE` | PostgreSQL SSL mode | `disable` |
| `MG_REPORTS_DB_SSL_CERT` | PostgreSQL SSL client cert | "" |
| `MG_REPORTS_DB_SSL_KEY` | PostgreSQL SSL client key | "" |
| `MG_REPORTS_DB_SSL_ROOT_CERT` | PostgreSQL SSL root cert | "" |

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
| `SMQ_SPICEDB_PRE_SHARED_KEY` | SpiceDB pre-shared key | `12345678` |
| `SMQ_SPICEDB_HOST` | SpiceDB host | `supermq-spicedb` |
| `SMQ_SPICEDB_PORT` | SpiceDB gRPC port | `50051` |

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
| `MG_REPORTS_EMAIL_TEMPLATE` | Template file mounted by Docker Compose | `reports.tmpl` |

### Templates and PDF conversion

| Variable | Description | Default |
| --- | --- | --- |
| `MG_REPORTS_DEFAULT_TEMPLATE` | Use on-disk HTML template when non-empty | "" |
| `MG_PDF_CONVERTER_URL` | HTML-to-PDF conversion endpoint | `http://pdf-generator:3000/forms/chromium/convert/html` |

## Features

- **Report generation**: Build report data from time-series messages.
- **Multiple formats**: JSON responses, CSV exports, and PDF rendering.
- **Scheduling**: Periodic report delivery via email.
- **Template support**: Custom HTML templates for PDF reports.
- **Observability**: `/metrics` Prometheus endpoint and Jaeger tracing support.

## Architecture

### Runtime flow

1. The Reports API receives a report request or a scheduled run triggers report generation.
2. The service expands requested metrics and fetches messages via the readers gRPC API in batches of 1000.
3. Results are grouped by publisher when `client_ids` are not specified.
4. Output is returned as JSON, rendered to CSV, or converted to PDF via `MG_PDF_CONVERTER_URL`.
5. For scheduled/email actions, the report is sent as an email attachment.

### Scheduling

The scheduler runs on a 30-second ticker and selects enabled report configs with `due` time earlier than now. It updates `due` using `Schedule.NextDue()` and generates a report with the `email` action.

Recurring types are: `none`, `hourly`, `daily`, `weekly`, `monthly`. The `recurring_period` controls the interval (1 = every interval, 2 = every second interval, etc.).

### Templates

PDF templates are Go `html/template` documents. A template must include:

- `{{$.Title}}`
- `{{range .Messages}}` or `{{range .Reports}}`
- `{{formatTime .Time}}`
- `{{formatValue .}}`
- `{{end}}`

Helper functions include `formatTime`, `formatValue`, `add`, `sub`, `div`, `mod`, `iterate`, `eq`, `ge`, `lt`, `getStartRow`, and `getEndRow`.

## Data model

### report_config table

Defined in `reports/postgres/init.go`:

| Column | Type | Description |
| --- | --- | --- |
| `id` | `VARCHAR(36)` | Report config UUID (primary key) |
| `name` | `VARCHAR(1024)` | Report name |
| `description` | `TEXT` | Report description |
| `domain_id` | `VARCHAR(36)` | Domain ID |
| `status` | `SMALLINT` | 0 = enabled, 1 = disabled, 2 = deleted |
| `created_at` | `TIMESTAMP` | Creation timestamp |
| `created_by` | `VARCHAR(254)` | Creator user ID |
| `updated_at` | `TIMESTAMP` | Last update timestamp |
| `updated_by` | `VARCHAR(254)` | Last updater user ID |
| `due` | `TIMESTAMPTZ` | Next scheduled execution time |
| `recurring` | `SMALLINT` | Recurring type |
| `recurring_period` | `SMALLINT` | Recurring period |
| `start_datetime` | `TIMESTAMP` | Schedule start time |
| `config` | `JSONB` | Metric config (from/to/title/format/aggregation) |
| `email` | `JSONB` | Email settings |
| `metrics` | `JSONB` | Requested metrics list |
| `report_template` | `TEXT` | Custom HTML template |

## Deployment

### Build and run locally

```bash
make reports

MG_REPORTS_LOG_LEVEL=debug \
MG_REPORTS_HTTP_PORT=9017 \
MG_REPORTS_DB_HOST=localhost \
MG_REPORTS_DB_PORT=5432 \
MG_REPORTS_DB_USER=magistrala \
MG_REPORTS_DB_PASS=magistrala \
MG_REPORTS_DB_NAME=reports \
MG_PDF_CONVERTER_URL=http://localhost:4000/forms/chromium/convert/html \
SMQ_AUTH_GRPC_URL=localhost:7001 \
SMQ_AUTH_GRPC_TIMEOUT=300s \
SMQ_DOMAINS_GRPC_URL=localhost:7003 \
SMQ_DOMAINS_GRPC_TIMEOUT=300s \
MG_TIMESCALE_READER_GRPC_URL=localhost:7011 \
MG_TIMESCALE_READER_GRPC_TIMEOUT=300s \
./build/reports
```

### Docker Compose

The service is available as a Docker container. Refer to [docker/docker-compose.yaml](https://github.com/absmach/magistrala/blob/main/docker/docker-compose.yaml) for the `reports`, `reports-db`, and `pdf-generator` services and their environment variables. For a full local stack, ensure auth, domains, readers, and the PDF generator are running.

```bash
docker compose -f docker/docker-compose.yaml up reports reports-db pdf-generator
```

### Health check

```bash
curl -X GET http://localhost:9017/health \
  -H "accept: application/health+json"
```

## Testing

```bash
go test ./reports/...
```

## Usage

The Reports service supports the following operations:

| Operation | Method & Path | Description |
| --- | --- | --- |
| `generateReport` | `POST /{domainID}/reports` | Generate a report (`action` query param) |
| `addReportConfig` | `POST /{domainID}/reports/configs` | Create a report configuration |
| `listReportsConfig` | `GET /{domainID}/reports/configs` | List report configurations |
| `viewReportConfig` | `GET /{domainID}/reports/configs/{reportID}` | View a report configuration |
| `updateReportConfig` | `PATCH /{domainID}/reports/configs/{reportID}` | Update a report configuration |
| `updateReportSchedule` | `PATCH /{domainID}/reports/configs/{reportID}/schedule` | Update schedule |
| `enableReportConfig` | `POST /{domainID}/reports/configs/{reportID}/enable` | Enable a report configuration |
| `disableReportConfig` | `POST /{domainID}/reports/configs/{reportID}/disable` | Disable a report configuration |
| `deleteReportConfig` | `DELETE /{domainID}/reports/configs/{reportID}` | Delete a report configuration |
| `updateReportTemplate` | `PUT /{domainID}/reports/configs/{reportID}/template` | Update custom template |
| `viewReportTemplate` | `GET /{domainID}/reports/configs/{reportID}/template` | View custom template |
| `deleteReportTemplate` | `DELETE /{domainID}/reports/configs/{reportID}/template` | Delete custom template |
| `health` | `GET /health` | Service health check |

List filters: `offset`, `limit`, `status`, `name`, `order` (`name`, `created_at`, `updated_at`), and `dir` (`asc`, `desc`).

Time ranges use relative expressions parsed by `pkg/reltime`, such as `now()` or `now()-24h` (units: `s`, `m`, `h`, `d`, `w`). Aggregation intervals use Go duration strings like `15m` or `1h`. File output formats are `pdf` and `csv`.

### Example: Generate a report

```bash
curl -X POST "http://localhost:9017/<domainID>/reports" \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "temperature-view",
    "metrics": [
      {
        "channel_id": "<channelID>",
        "client_ids": ["<clientID>"],
        "name": "temperature",
        "subtopic": "sensor"
      }
    ],
    "config": {
      "from": "now()-24h",
      "to": "now()",
      "title": "Temperature (last 24h)",
      "timezone": "UTC",
      "aggregation": {
        "agg_type": "avg",
        "interval": "1h"
      }
    }
  }'
```

### Example: Generate and email a report

```bash
curl -X POST "http://localhost:9017/<domainID>/reports?action=email" \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "temperature-email",
    "metrics": [
      {
        "channel_id": "<channelID>",
        "name": "temperature"
      }
    ],
    "config": {
      "from": "now()-1d",
      "to": "now()",
      "title": "Daily Temperature",
      "file_format": "csv"
    },
    "email": {
      "to": ["ops@example.com"],
      "subject": "Daily temperature report",
      "content": "Report attached."
    }
  }'

```

### Example: Create a scheduled report config

```bash
curl -X POST "http://localhost:9017/<domainID>/reports/configs" \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "daily-temperature",
    "description": "Daily temperature summary",
    "metrics": [
      {
        "channel_id": "<channelID>",
        "name": "temperature"
      }
    ],
    "config": {
      "from": "now()-1d",
      "to": "now()",
      "title": "Daily Temperature",
      "file_format": "pdf",
      "aggregation": {
        "agg_type": "avg",
        "interval": "1h"
      }
    },
    "email": {
      "to": ["ops@example.com"],
      "subject": "Daily temperature report",
      "content": "Report attached."
    },
    "schedule": {
      "start_datetime": "2025-01-01T00:00:00Z",
      "recurring": "daily",
      "recurring_period": 1
    }
  }'
```

### Example: Update a report template

```bash
curl -X PUT "http://localhost:9017/<domainID>/reports/configs/<reportID>/template" \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "report_template": "<html><body><h1>{{$.Title}}</h1>{{range .Reports}}{{range .Messages}}{{formatTime .Time}} {{formatValue .}}{{end}}{{end}}</body></html>"
  }'
```

### Example: Enable a report config

```bash
curl -X POST "http://localhost:9017/<domainID>/reports/configs/<reportID>/enable" \
  -H "Authorization: Bearer <access_token>"
```

For an in-depth explanation of our Reports Service, see the see the [official documentation][doc].

[doc]: https://docs.magistrala.absmach.eu/dev-guide/reports/
