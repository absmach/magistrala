# Channels

The Channels service is a core component of SuperMQ that manages communication channels between devices and applications. It handles channel creation, configuration, access control and message routing within the SuperMQ ecosystem.

## Configuration

The service is configured using the following environment variables (unset variables use default values):

| Variable                     | Description                                                  | Default     |
|-----------------------------|--------------------------------------------------------------|-------------|
| `MG_CHANNELS_LOG_LEVEL`      | Log level (debug, info, warn, error)                          | info        |
| `MG_CHANNELS_HTTP_HOST`      | HTTP host for Channels service                               | localhost   |
| `MG_CHANNELS_HTTP_PORT`      | HTTP port for Channels service                               | 9005        |
| `MG_CHANNELS_SERVER_CERT`    | Path to PEM encoded server certificate                       | ""          |
| `MG_CHANNELS_SERVER_KEY`     | Path to PEM encoded server key file                          | ""          |
| `MG_CHANNELS_GRPC_HOST`      | gRPC host for Channels service                               | localhost   |
| `MG_CHANNELS_GRPC_PORT`      | gRPC port for Channels service                               | 7005        |
| `MG_CHANNELS_DB_HOST`        | Database host address                                        | localhost   |
| `MG_CHANNELS_DB_PORT`        | Database port                                                | 5432        |
| `MG_CHANNELS_DB_USER`        | Database user                                                | supermq     |
| `MG_CHANNELS_DB_PASS`        | Database password                                            | supermq     |
| `MG_CHANNELS_DB_NAME`        | Name of the database used by the service                    | channels    |
| `MG_CHANNELS_DB_SSL_MODE`    | Database connection SSL mode                                 | disable     |
| `MG_CHANNELS_CACHE_URL`      | Cache database URL                                           | <redis://localhost:6379/0> |
| `MG_JAEGER_URL`              | Jaeger tracing server URL                                    | <http://jaeger:4318/v1/traces> |
| `MG_SEND_TELEMETRY`          | Send telemetry to SuperMQ call-home server                   | true        |

## Features

- **Channel Management**: Create, update, delete and list channels
- **Access Control**: Manage channel permissions and user access
- **Message Routing**: Route messages between connected devices and services  
- **Channel Groups**: Organize channels into logical groups
- **Metadata Support**: Attach custom metadata to channels
- **Real-time Updates**: Live channel state synchronization

## Architecture

The service is built using:

- **Go**: Core service implementation
- **gRPC**: Inter-service communication
- **PostgreSQL**: Primary data storage
- **Redis**: Caching and pub/sub messaging
- **Docker**: Containerized deployment

### Channels Table

| Column             | Type           | Description                                                    |
|--------------------|----------------|----------------------------------------------------------------|
| `id`               | VARCHAR(36)    | UUID of the channel (primary key)                              |
| `name`             | VARCHAR(1024)  | Human-readable name                                            |
| `domain_id`        | VARCHAR(36)    | Domain to which the channel belongs                            |
| `parent_group_id`  | VARCHAR(36)    | Optional group parent                                          |
| `tags`             | TEXT[]         | Array of tags                                                  |
| `metadata`         | JSONB          | Free-form structured metadata                                  |
| `created_by`       | VARCHAR(254)   | User that created the channel                                  |
| `created_at`       | TIMESTAMPTZ    | Timestamp of creation                                          |
| `updated_at`       | TIMESTAMPTZ    | Timestamp of last update                                       |
| `updated_by`       | VARCHAR(254)   | User that performed last update                                |
| `status`           | SMALLINT       | 0 = enabled, 1 = disabled                                      |
| `route`            | VARCHAR(36)    | Optional route identifier unique within domain if set          |

### Connections Table

| Column        | Type         | Description                                          |
|---------------|--------------|------------------------------------------------------|
| `channel_id`  | VARCHAR(36)  | Channel UUID                                         |
| `domain_id`   | VARCHAR(36)  | Domain of channel and client                         |
| `client_id`   | VARCHAR(36)  | Client UUID                                          |
| `type`        | SMALLINT     | Connection type: `1 = Publish`, `2 = Subscribe`      |

## Deployment

The service is available as a Docker container. Refer to the Docker Compose section for the `channels` service in `docker-compose.yaml` for deployment configuration.

To build and run locally:

```bash
# download the latest version of the service
git clone https://github.com/absmach/supermq
cd supermq

# compile the channels
make channels

make install

MG_CHANNELS_HTTP_HOST=localhost \
MG_CHANNELS_HTTP_PORT=9005 \
MG_CHANNELS_DB_HOST=localhost \
MG_CHANNELS_DB_PORT=5432 \
MG_CHANNELS_DB_USER=supermq \
MG_CHANNELS_DB_PASS=supermq \
MG_CHANNELS_DB_NAME=channels \
$GOBIN/supermq-channels
```

### Running the Service

```bash
# Set environment variables
export MQ_CHANNELS_DB_HOST=localhost
export MQ_CHANNELS_DB_PORT=5432

# Run the service
go run cmd/main.go
```

### Docker Deployment

```bash
docker run -p 8180:8180 supermq/channels
```

## Testing

```bash
# Run unit tests
go test ./...

# Run integration tests
make test-integration
```

## Usage

The Channels service supports the following operations:

| Operation        | Description                                                   |
|------------------|---------------------------------------------------------------|
| `create`         | Create a new channel                                          |
| `list`           | Retrieve all channels (paged)                                 |
| `get`            | Retrieve a single channel by ID                               |
| `update`         | Update a channel’s name & metadata                            |
| `delete`         | Permanently delete a channel                                  |
| `enable`         | Enable a previously disabled channel                          |
| `disable`        | Disable an active channel                                     |
| `set-parent`     | Assign a parent group to a channel                            |
| `remove-parent`  | Remove parent group from a channel                            |
| `connect`        | Connect one or more clients to channels                       |
| `disconnect`     | Disconnect one or more clients from channels                  |

### Example: Create a Channel

```bash
curl -X POST http://localhost:9005/<domainID>/channels \
  -H "Authorization: Bearer <your_access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "myChannel",
    "metadata": { "location": "lab" },
    "route": "sensor-data",
    "tags": ["sensor","edge"],
    "status": "enabled"
  }'
```

### Example: Connect Clients & Channels

```bash
curl -X POST http://localhost:9005/<domainID>/channels/connect \
  -H "Authorization: Bearer <your_access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_ids": ["<chanID1>", "<chanID2>"],
    "client_ids": ["<clientID1>", "<clientID2>"],
    "types": ["publish", "subscribe"]
  }'
```

### Example: Disconnect Clients from a Channel

```bash
curl -X POST http://localhost:9005/<domainID>/channels/disconnect \
  -H "Authorization: Bearer <your_access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_ids": ["<chanID>"],
    "client_ids": ["<clientID>"],
    "types": ["publish"]
  }'
```

## Best Practices

- Use tags and metadata to manage and categorize channels (e.g., environment, region, purpose).
- Assign `route` thoughtfully when channels need a predictable identifier.
- Keep channel hierarchies shallow for easier navigation (avoid deep nesting unless required).
- Use `disable` rather than immediate delete when you want to suspend a channel temporarily.
- Clean up unused connections: regularly review which clients are connected to channels and remove stale links.
- Enforce minimal privileges: only allow clients to connect to channels they truly need.
- Monitoring: use the `/health` endpoint and version metadata for service stability.

## Versioning & Health Check

The Channels service exposes a `/health` endpoint to provide operational status and version info.

### Health Check Request

```bash
curl -X GET http://localhost:9005/health \
  -H "accept: application/health+json"
```

### Example Response

```json
{
  "status": "pass",
  "version": "0.18.0",
  "commit": "<commit-hash>",
  "description": "channels service",
  "build_time": "2025-11-19T..."
}
```
