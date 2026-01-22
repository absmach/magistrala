# Notifiers

The Notifiers service manages notification subscriptions and dispatches alerts for incoming messages. It stores subscription records (topic + contact), exposes an HTTP API for CRUD operations, and consumes SuperMQ messages to fan out notifications via notifier implementations (SMTP for email, SMPP for SMS). Notifiers are dependencies used by the service, not standalone services.

## Configuration

The service is configured using environment variables. Values shown are from [docker/.env](https://github.com/absmach/magistrala/blob/main/docker/.env) when available; otherwise defaults come from code or notifier-specific docs.

### SMTP notifier (email)

Used by `consumers/notifiers/smtp` via `internal/email`.

| Variable | Description | Default |
| --- | --- | --- |
| `MG_EMAIL_HOST` | SMTP host | `smtp.mailtrap.io` |
| `MG_EMAIL_PORT` | SMTP port | `2525` |
| `MG_EMAIL_USERNAME` | SMTP username | `18bf7f70705139` |
| `MG_EMAIL_PASSWORD` | SMTP password | `2b0d302e775b1e` |
| `MG_EMAIL_FROM_ADDRESS` | Default from address (used if `from` is empty) | `from@example.com` |
| `MG_EMAIL_FROM_NAME` | Default from name | `Example` |
| `MG_EMAIL_TEMPLATE` | Email template path | `email.tmpl` |

### SMPP notifier (SMS)

#### SMPP transport settings

Defined in `consumers/notifiers/smpp/config.go`.

| Variable | Description | Default |
| --- | --- | --- |
| `MG_SMPP_ADDRESS` | SMPP address in `host:port` format | "" |
| `MG_SMPP_USERNAME` | SMPP username | "" |
| `MG_SMPP_PASSWORD` | SMPP password | "" |
| `MG_SMPP_SYSTEM_TYPE` | SMPP system type | "" |
| `MG_SMPP_SRC_ADDR_TON` | SMPP source address TON | `0` |
| `MG_SMPP_DST_ADDR_TON` | SMPP source address NPI | `0` |
| `MG_SMPP_SRC_ADDR_NPI` | SMPP destination address TON | `0` |
| `MG_SMPP_DST_ADDR_NPI` | SMPP destination address NPI | `0` |

Note: The SMPP env tags are mapped exactly as defined in `consumers/notifiers/smpp/config.go`.

#### SMPP notifier service settings

Defined in `consumers/notifiers/smpp/README.md`.

| Variable | Description | Default |
| --- | --- | --- |
| `MG_SMPP_NOTIFIER_LOG_LEVEL` | Log level for SMPP notifier | `info` |
| `MG_SMPP_NOTIFIER_FROM_ADDRESS` | From address for SMS notifications | "" |
| `MG_SMPP_NOTIFIER_CONFIG_PATH` | Config file path for message broker subjects and payload type | `/config.toml` |
| `MG_SMPP_NOTIFIER_HTTP_HOST` | Service HTTP host | `localhost` |
| `MG_SMPP_NOTIFIER_HTTP_PORT` | Service HTTP port | `9014` |
| `MG_SMPP_NOTIFIER_HTTP_SERVER_CERT` | Service HTTP server certificate path | "" |
| `MG_SMPP_NOTIFIER_HTTP_SERVER_KEY` | Service HTTP server key path | "" |
| `MG_SMPP_NOTIFIER_DB_HOST` | Database host address | `localhost` |
| `MG_SMPP_NOTIFIER_DB_PORT` | Database host port | `5432` |
| `MG_SMPP_NOTIFIER_DB_USER` | Database user | `magistrala` |
| `MG_SMPP_NOTIFIER_DB_PASS` | Database password | `magistrala` |
| `MG_SMPP_NOTIFIER_DB_NAME` | Database name | `subscriptions` |
| `MG_SMPP_NOTIFIER_DB_SSL_MODE` | DB SSL mode (disable, require, verify-ca, verify-full) | `disable` |
| `MG_SMPP_NOTIFIER_DB_SSL_CERT` | DB SSL client cert path | "" |
| `MG_SMPP_NOTIFIER_DB_SSL_KEY` | DB SSL client key path | "" |
| `MG_SMPP_NOTIFIER_DB_SSL_ROOT_CERT` | DB SSL root cert path | "" |
| `SMQ_AUTH_GRPC_URL` | Auth gRPC URL | `localhost:7001` |
| `SMQ_AUTH_GRPC_TIMEOUT` | Auth gRPC timeout | `1s` |
| `MG_AUTH_GRPC_CLIENT_TLS` | Auth client TLS flag | `false` |
| `MG_AUTH_GRPC_CA_CERT` | Auth client CA certs path | "" |
| `SMQ_MESSAGE_BROKER_URL` | Message broker URL | `nats://127.0.0.1:4222` |
| `SMQ_JAEGER_URL` | Jaeger tracing URL | `http://jaeger:14268/api/traces` |
| `SMQ_SEND_TELEMETRY` | Send telemetry to Magistrala call-home server | `true` |
| `MG_SMPP_NOTIFIER_INSTANCE_ID` | SMPP notifier instance ID | "" |

## Features

- **Subscription management**: Create, view, list, and remove notification subscriptions.
- **Topic-based dispatch**: Matches subscriptions by topic and fan-outs to contacts.
- **Multiple notifier backends**: SMTP (email) and SMPP (SMS) implementations are available.
- **Observability**: Exposes `/metrics` and `/health` endpoints.
- **Uniqueness guardrails**: Prevents duplicate subscriptions for the same topic/contact pair.

## Architecture

### Runtime flow

1. Clients register subscriptions through the HTTP API (`topic` + `contact`).
2. The service authenticates the token, assigns an owner ID, and persists the subscription.
3. When a message arrives, the service builds the topic as `channel` or `channel.subtopic`, retrieves matching subscriptions, and gathers contacts.
4. The notifier implementation sends notifications using the configured backend.

### Components

- **HTTP API**: `consumers/notifiers/api` exposes `/subscriptions`, `/health`, and `/metrics`.
- **Service layer**: `consumers/notifiers/service.go` handles authn, ID creation, and notification dispatch.
- **Repository**: `consumers/notifiers/postgres` persists subscriptions and supports filtering.
- **Notifier implementations**: `consumers/notifiers/smtp` (email) and `consumers/notifiers/smpp` (SMS).
- **Email agent**: `internal/email` manages SMTP connectivity and template rendering.

### Subscriptions table

Defined in `consumers/notifiers/postgres/init.go`:

| Column | Type | Description |
| --- | --- | --- |
| `id` | `VARCHAR(254)` | Subscription identifier (primary key) |
| `owner_id` | `VARCHAR(254)` | Owner ID derived from the auth token |
| `contact` | `VARCHAR(254)` | Notification contact (email or phone) |
| `topic` | `TEXT` | Topic to match (`channel` or `channel.subtopic`) |

Constraint: `UNIQUE(topic, contact)`

## Deployment

The Notifiers service is provided as a consumer package. It is typically wired into a notifier-specific binary that provides the HTTP server and message broker subscription. For the SMPP notifier runtime configuration, see `consumers/notifiers/smpp/README.md`.

### Health check

```bash
curl -X GET http://localhost:9014/health \
  -H "accept: application/health+json"
```

## Testing

```bash
go test ./consumers/notifiers/...
```

## Usage

The Notifiers service supports the following operations (see `apidocs/openapi/notifiers.yaml`):

| Operation | Method & Path | Description |
| --- | --- | --- |
| `createSubscription` | `POST /subscriptions` | Create a new subscription |
| `listSubscriptions` | `GET /subscriptions` | List subscriptions with filters |
| `viewSubscription` | `GET /subscriptions/{id}` | Retrieve a subscription |
| `removeSubscription` | `DELETE /subscriptions/{id}` | Delete a subscription |
| `health` | `GET /health` | Service health check |

### Example: Create a subscription

```bash
curl -X POST http://localhost:9014/subscriptions \
  -H "Authorization: Bearer <your_access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "channel.subtopic",
    "contact": "user@example.com"
  }'
```

### Example: List subscriptions

```bash
curl -X GET "http://localhost:9014/subscriptions?topic=channel.subtopic&contact=user@example.com&limit=20&offset=0" \
  -H "Authorization: Bearer <your_access_token>"
```

### Example: View a subscription

```bash
curl -X GET http://localhost:9014/subscriptions/<subscriptionID> \
  -H "Authorization: Bearer <your_access_token>"
```

### Example: Remove a subscription

```bash
curl -X DELETE http://localhost:9014/subscriptions/<subscriptionID> \
  -H "Authorization: Bearer <your_access_token>"
```

### Example: Health check

```bash
curl -X GET http://localhost:9014/health \
  -H "accept: application/health+json"
```

For an in-depth explanation of the Notifiers, see the [official documentation][doc].

[doc]: https://docs.magistrala.absmach.eu/dev-guide/consumers/#notifiers
