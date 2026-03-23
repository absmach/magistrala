# Journal

The Journal service listens to the platform event stream, persists each event to PostgreSQL for auditability and exposes HTTP endpoints to query journals or view per-client telemetry (first/last seen, subscriptions, in/out message counters).

## Configuration

The service is configured with the following environment variables (unset values fall back to defaults).

| Variable | Description | Default |
| --- | --- | --- |
| `MG_JOURNAL_LOG_LEVEL` | Log level for Journal (debug, info, warn, error) | info |
| `MG_JOURNAL_HTTP_HOST` | Journal HTTP host | localhost |
| `MG_JOURNAL_HTTP_PORT` | Journal HTTP port | 9021 |
| `MG_JOURNAL_HTTP_SERVER_CERT` | Path to PEM-encoded HTTP server certificate | "" |
| `MG_JOURNAL_HTTP_SERVER_KEY` | Path to PEM-encoded HTTP server key | "" |
| `MG_JOURNAL_HTTP_SERVER_CA_CERTS` | Path to trusted CA bundle for the HTTP server | "" |
| `MG_JOURNAL_HTTP_CLIENT_CA_CERTS` | Path to client CA bundle to require HTTP mTLS | "" |
| `MG_JOURNAL_DB_HOST` | Database host address | localhost |
| `MG_JOURNAL_DB_PORT` | Database host port | 5432 |
| `MG_JOURNAL_DB_USER` | Database user | supermq |
| `MG_JOURNAL_DB_PASS` | Database password | supermq |
| `MG_JOURNAL_DB_NAME` | Name of the database used by the service | journal |
| `MG_JOURNAL_DB_SSL_MODE` | Database connection SSL mode (disable, require, verify-ca, verify-full) | disable |
| `MG_JOURNAL_DB_SSL_CERT` | Path to the PEM-encoded certificate file | "" |
| `MG_JOURNAL_DB_SSL_KEY` | Path to the PEM-encoded key file | "" |
| `MG_JOURNAL_DB_SSL_ROOT_CERT` | Path to the PEM-encoded root certificate file | "" |
| `MG_ES_URL` | Event store URL (NATS) consumed for journal entries | nats://localhost:4222 |
| `MG_JAEGER_URL` | Jaeger tracing endpoint | <http://localhost:4318/v1/traces> |
| `MG_JAEGER_TRACE_RATIO` | Trace sampling ratio | 1.0 |
| `MG_SEND_TELEMETRY` | Send telemetry to the SuperMQ call-home server | true |
| `MG_AUTH_GRPC_URL` | Auth service gRPC URL | "" |
| `MG_AUTH_GRPC_TIMEOUT` | Auth service gRPC timeout | 1s |
| `MG_AUTH_GRPC_CLIENT_CERT` | Path to PEM-encoded Auth gRPC client certificate | "" |
| `MG_AUTH_GRPC_CLIENT_KEY` | Path to PEM-encoded Auth gRPC client key | "" |
| `MG_AUTH_GRPC_SERVER_CA_CERTS` | Path to PEM-encoded Auth gRPC trusted CA bundle | "" |
| `MG_DOMAINS_GRPC_URL` | Domains service gRPC URL | "" |
| `MG_DOMAINS_GRPC_TIMEOUT` | Domains service gRPC timeout | 1s |
| `MG_DOMAINS_GRPC_CLIENT_CERT` | Path to PEM-encoded Domains gRPC client certificate | "" |
| `MG_DOMAINS_GRPC_CLIENT_KEY` | Path to PEM-encoded Domains gRPC client key | "" |
| `MG_DOMAINS_GRPC_SERVER_CA_CERTS` | Path to PEM-encoded Domains gRPC trusted CA bundle | "" |
| `MG_JOURNAL_INSTANCE_ID` | Journal instance ID (auto-generated when empty) | "" |
| `MG_ALLOW_UNVERIFIED_USER` | Allow unverified users to authenticate (useful in dev) | false |

## Deployment

The service is distributed as a Docker container. Check [`docker/docker-compose.yaml`](https://github.com/absmach/supermq/tree/main/docker/docker-compose.yaml) for the `journal` and `journal-db` services and how they are wired into the base stack.

To start the service outside of the container, execute the following shell script:

```bash
git clone https://github.com/absmach/supermq
cd supermq

# build and install the binary
make journal
make install

# run with the essentials; requires Postgres, Auth gRPC, Domains gRPC, and NATS running
MG_JOURNAL_HTTP_HOST=localhost \
MG_JOURNAL_HTTP_PORT=9021 \
MG_JOURNAL_DB_HOST=localhost \
MG_JOURNAL_DB_PORT=5432 \
MG_JOURNAL_DB_USER=supermq \
MG_JOURNAL_DB_PASS=supermq \
MG_JOURNAL_DB_NAME=journal \
MG_AUTH_GRPC_URL=localhost:7001 \
MG_DOMAINS_GRPC_URL=localhost:7003 \
MG_ES_URL=nats://localhost:4222 \
$GOBIN/supermq-journal
```

## HTTP API

Base URL defaults to `http://localhost:9021`. All journal and telemetry endpoints require `Authorization: Bearer <token>` (health is public).

### Usage

| Operation | Description |
| --- | --- |
| List user journals | Page through journals for a user across domains. |
| List entity journals | Page through journals for a group, client, channel, or user within a domain. |
| View client telemetry | Aggregate telemetry counters for a client in a domain. |
| Health check | Liveness and build info. |

### API examples

#### List user journals

```bash
curl -X GET "http://localhost:9021/journal/user/${USER_ID}?limit=5&with_attributes=true&dir=desc" \
  -H "Authorization: Bearer $TOKEN"
```

Expected response:

```json
{
  "journals": [
    {
      "operation": "user.create",
      "occurred_at": "2024-01-11T12:05:07.449053Z",
      "attributes": {
        "created_at": "2024-06-12T11:34:32.991591Z",
        "id": "29d425c8-542b-4614-8a4d-a5951945d720",
        "identity": "Gawne-Havlicek@email.com",
        "name": "Newgard-Frisina",
        "status": "enabled",
        "updated_at": "2024-06-12T11:34:33.116795Z",
        "updated_by": "ad228f20-4741-47c5-bef7-d871b541c019"
      },
      "metadata": {
        "Update": "Calvo-Felkins"
      }
    }
  ],
  "total": 1,
  "offset": 0,
  "limit": 5
}
```

#### List entity journals in a domain

Retrieves telemetry data for a specific client within a domain. This includes connection status, messages sent/received, and other metrics.

```bash
curl -X GET "http://localhost:9021/${DOMAIN_ID}/journal/client/${CLIENT_ID}?operation=client.create&with_metadata=true&dir=desc&limit=10" \
  -H "Authorization: Bearer $TOKEN"
```

Expected response:

```json
{
  "total": 2,
  "offset": 0,
  "limit": 10,
  "journals": [
    {
      "operation": "client.create",
      "occurred_at": "2024-06-12T11:34:33Z",
      "domain": "29d425c8-542b-4614-8a4d-a5951945d720",
      "attributes": {
        "id": "bb7edb32-2eac-4aad-aebe-ed96fe073879",
        "domain": "29d425c8-542b-4614-8a4d-a5951945d720",
        "name": "clientName",
        "status": "enabled"
      },
      "metadata": {
        "trace_id": "6efb4c24b1b4a684"
      }
    }
  ]
}
```

#### View client telemetry

Retrieves telemetry data for a specific client within a domain. This includes connection status, messages sent/received, and other metrics.

```bash
curl -X GET "http://localhost:9021/${DOMAIN_ID}/journal/client/${CLIENT_ID}/telemetry" \
  -H "Authorization: Bearer $TOKEN"
```

Expected response:

```json
{
  "client_id": "bb7edb32-2eac-4aad-aebe-ed96fe073879",
  "domain_id": "29d425c8-542b-4614-8a4d-a5951945d720",
  "subscriptions": 5,
  "inbound_messages": 1234567,
  "outbound_messages": 987654,
  "first_seen": "2024-01-11T10:00:00Z",
  "last_seen": "2024-01-11T12:05:07.449053Z"
}
```

#### Health check

```bash
curl "http://localhost:9021/health"
```

Expected response:

```json
{
  "status": "pass",
  "version": "0.18.0",
  "commit": "ffffffff",
  "description": "journal service",
  "build_time": "1970-01-01_00:00:00",
  "instance_id": "b4f1d5d2-4f24-4c2a-9a40-123456789abc"
}
```
