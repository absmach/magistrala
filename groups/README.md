# Groups

The Groups service exposes HTTP and gRPC APIs for organizing entities into hierarchical groups within a domain, managing membership, permissions, and roles. It handles group lifecycle (create/update/enable/disable/delete), parent/child relationships, listings (flat or tree), and role-based access.

For a deeper overview of SuperMQ, see the [official documentation][doc].

## Configuration

The service is configured via environment variables (unset values fall back to defaults).

| Variable                               | Description                                                                                       | Default                               |
| -------------------------------------- | ------------------------------------------------------------------------------------------------- | ------------------------------------- |
| `MG_GROUPS_LOG_LEVEL`                 | Log level for Groups (debug, info, warn, error)                                                   | debug                                  |
| `MG_GROUPS_HTTP_HOST`                 | Groups service HTTP host                                                                          | groups                                 |
| `MG_GROUPS_HTTP_PORT`                 | Groups service HTTP port                                                                          | 9004                                   |
| `MG_GROUPS_HTTP_SERVER_CERT`          | Path to PEM-encoded HTTP server certificate                                                       | ""                                     |
| `MG_GROUPS_HTTP_SERVER_KEY`           | Path to PEM-encoded HTTP server key                                                               | ""                                     |
| `MG_GROUPS_HTTP_SERVER_CA_CERTS`      | Path to trusted CA bundle for the HTTP server                                                     | ""                                     |
| `MG_GROUPS_HTTP_CLIENT_CA_CERTS`      | Path to client CA bundle to require HTTP mTLS                                                     | ""                                     |
| `MG_GROUPS_GRPC_HOST`                 | Groups service gRPC host                                                                          | groups                                 |
| `MG_GROUPS_GRPC_PORT`                 | Groups service gRPC port                                                                          | 7004                                   |
| `MG_GROUPS_GRPC_SERVER_CERT`          | Path to PEM-encoded gRPC server certificate                                                       | ""                                     |
| `MG_GROUPS_GRPC_SERVER_KEY`           | Path to PEM-encoded gRPC server key                                                               | ""                                     |
| `MG_GROUPS_GRPC_SERVER_CA_CERTS`      | Path to trusted CA bundle for the gRPC server                                                     | ""                                     |
| `MG_GROUPS_GRPC_CLIENT_CA_CERTS`      | Path to client CA bundle to require gRPC mTLS                                                     | ""                                     |
| `MG_GROUPS_DB_HOST`                   | Database host address                                                                             | groups-db                              |
| `MG_GROUPS_DB_PORT`                   | Database host port                                                                                | 5432                                   |
| `MG_GROUPS_DB_USER`                   | Database user                                                                                     | supermq                                |
| `MG_GROUPS_DB_PASS`                   | Database password                                                                                 | supermq                                |
| `MG_GROUPS_DB_NAME`                   | Name of the database used by the service                                                          | groups                                 |
| `MG_GROUPS_DB_SSL_MODE`               | Database connection SSL mode (disable, require, verify-ca, verify-full)                           | disable                                |
| `MG_GROUPS_DB_SSL_CERT`               | Path to the PEM-encoded certificate file                                                          | ""                                     |
| `MG_GROUPS_DB_SSL_KEY`                | Path to the PEM-encoded key file                                                                  | ""                                     |
| `MG_GROUPS_DB_SSL_ROOT_CERT`          | Path to the PEM-encoded root certificate file                                                     | ""                                     |
| `MG_GROUPS_INSTANCE_ID`               | Groups instance ID (auto-generated when empty)                                                    | ""                                     |
| `MG_GROUPS_EVENT_CONSUMER`            | NATS consumer name for domain events                                                              | groups                                 |
| `MG_SPICEDB_HOST`                     | SpiceDB host for policy checks                                                                    | supermq-spicedb                              |
| `MG_SPICEDB_PORT`                     | SpiceDB port                                                                                      | 50051                                  |
| `MG_SPICEDB_SCHEMA_FILE`              | Path to SpiceDB schema file used to seed available actions                                        | "/schema.zed"                              |
| `MG_SPICEDB_PRE_SHARED_KEY`           | SpiceDB preshared key                                                                             | 12345678                               |
| `MG_ES_URL`                           | Event store URL                                                                                   | nats://nats:4222                  |
| `MG_JAEGER_URL`                       | Jaeger server URL                                                                                 | <http://jaeger:4318/v1/traces>      |
| `MG_JAEGER_TRACE_RATIO`               | Trace sampling ratio                                                                              | 1.0                                    |
| `MG_SEND_TELEMETRY`                   | Send telemetry to the SuperMQ call-home server                                                    | true                                   |
| `MG_AUTH_GRPC_URL`                    | Auth service gRPC URL                                                                             | ""                                     |
| `MG_AUTH_GRPC_TIMEOUT`                | Auth service gRPC request timeout                                                                 | 1s                                     |
| `MG_AUTH_GRPC_CLIENT_CERT`            | Path to the PEM-encoded Auth gRPC client certificate                                              | ""                                     |
| `MG_AUTH_GRPC_CLIENT_KEY`             | Path to the PEM-encoded Auth gRPC client key                                                      | ""                                     |
| `MG_AUTH_GRPC_SERVER_CA_CERTS`        | Path to the PEM-encoded Auth gRPC trusted CA bundle                                               | ""                                     |
| `MG_GROUPS_CALLOUT_URLS`              | Comma-separated list of HTTP callout targets invoked on group operations                          | ""                                     |
| `MG_GROUPS_CALLOUT_METHOD`            | HTTP method for callouts (POST or GET)                                                            | POST                                   |
| `MG_GROUPS_CALLOUT_TLS_VERIFICATION`  | Verify TLS certificates for callouts                                                              | false                                  |
| `MG_GROUPS_CALLOUT_TIMEOUT`           | Callout request timeout                                                                           | 10s                                    |
| `MG_GROUPS_CALLOUT_CA_CERT`           | CA bundle for verifying callout targets                                                           | ""                                     |
| `MG_GROUPS_CALLOUT_CERT`              | Client certificate for mTLS callouts                                                              | ""                                     |
| `MG_GROUPS_CALLOUT_KEY`               | Client key for mTLS callouts                                                                      | ""                                     |
| `MG_GROUPS_CALLOUT_OPERATIONS`        | Comma-separated list of operation names that should trigger callouts                              | ""                                     |

**Note**: Set `MG_GROUPS_CALLOUT_OPERATIONS` to a subset of `OpCreateGroup`, `OpViewGroup`, `OpUpdateGroup`, `OpEnableGroup`, `OpDisableGroup`, `OpDeleteGroup`, `OpListGroups`, `OpHierarchy`, `OpAddParentGroup`, `OpRemoveParentGroup`, `OpAddChildrenGroups`, `OpRemoveChildrenGroups`, `OpRemoveAllChildrenGroups`, or `OpListChildrenGroups` to filter which actions produce callouts.

## Deployment

The service ships as a Docker container. See the [`groups` section](https://github.com/absmach/supermq/blob/main/docker/docker-compose.yaml#L950-L1035) in `docker-compose.yaml` for deployment configuration.

To build and run locally:

```bash
# download the latest version of the service
git clone https://github.com/absmach/supermq
cd supermq

# compile the groups service
make groups

# copy binary to $GOBIN
make install

# set the environment variables and run the service
MG_GROUPS_LOG_LEVEL=debug \
MG_GROUPS_HTTP_HOST=groups \
MG_GROUPS_HTTP_PORT=9004 \
MG_GROUPS_HTTP_SERVER_CERT="" \
MG_GROUPS_HTTP_SERVER_KEY="" \
MG_GROUPS_GRPC_HOST=groups \
MG_GROUPS_GRPC_PORT=7004 \
MG_GROUPS_GRPC_SERVER_CERT="" \
MG_GROUPS_GRPC_SERVER_KEY="" \
MG_GROUPS_GRPC_SERVER_CA_CERTS="" \
MG_GROUPS_GRPC_CLIENT_CA_CERTS="" \
MG_GROUPS_DB_HOST=groups-db \
MG_GROUPS_DB_PORT=5432 \
MG_GROUPS_DB_USER=supermq \
MG_GROUPS_DB_PASS=supermq \
MG_GROUPS_DB_NAME=groups \
MG_GROUPS_DB_SSL_MODE=disable \
MG_GROUPS_DB_SSL_CERT="" \
MG_GROUPS_DB_SSL_KEY="" \
MG_GROUPS_DB_SSL_ROOT_CERT="" \
MG_AUTH_GRPC_URL="" \
MG_AUTH_GRPC_TIMEOUT=1s \
MG_AUTH_GRPC_CLIENT_CERT="" \
MG_AUTH_GRPC_CLIENT_KEY="" \
MG_AUTH_GRPC_SERVER_CA_CERTS="" \
MG_DOMAINS_GRPC_URL=domains:7003 \
MG_DOMAINS_GRPC_TIMEOUT=1s \
MG_DOMAINS_GRPC_CLIENT_CERT="" \
MG_DOMAINS_GRPC_CLIENT_KEY="" \
MG_DOMAINS_GRPC_SERVER_CA_CERTS="" \
MG_CHANNELS_GRPC_URL=channels:7005 \
MG_CHANNELS_GRPC_TIMEOUT=1s \
MG_CHANNELS_GRPC_CLIENT_CERT="" \
MG_CHANNELS_GRPC_CLIENT_KEY="" \
MG_CHANNELS_GRPC_SERVER_CA_CERTS="" \
MG_CLIENTS_GRPC_URL=clients:7000 \
MG_CLIENTS_GRPC_TIMEOUT=1s \
MG_CLIENTS_GRPC_CLIENT_CERT="" \
MG_CLIENTS_GRPC_CLIENT_KEY="" \
MG_CLIENTS_GRPC_SERVER_CA_CERTS="" \
MG_SPICEDB_HOST=localhost \
MG_SPICEDB_PORT=50051 \
MG_SPICEDB_SCHEMA_FILE=schema.zed \
MG_SPICEDB_PRE_SHARED_KEY=12345678 \
MG_ES_URL=nats://localhost:4222 \
MG_JAEGER_URL=<http://localhost:4318/v1/traces> \
MG_JAEGER_TRACE_RATIO=1.0 \
MG_GROUPS_CALLOUT_URLS="" \
MG_GROUPS_CALLOUT_METHOD=POST \
MG_GROUPS_CALLOUT_TLS_VERIFICATION=false \
MG_GROUPS_CALLOUT_TIMEOUT=10s \
MG_GROUPS_CALLOUT_CA_CERT="" \
MG_GROUPS_CALLOUT_CERT="" \
MG_GROUPS_CALLOUT_KEY="" \
MG_GROUPS_CALLOUT_OPERATIONS="" \
MG_SEND_TELEMETRY=true \
MG_GROUPS_INSTANCE_ID="" \
$GOBIN/supermq-groups
```

## Usage

Groups supports the following operations:

| Operation                 | Description                                                                 |
| ------------------------- | --------------------------------------------------------------------------- |
| `create`                  | Create a new group within a domain                                          |
| `list`                    | List groups (flat list or tree) with filters for metadata, tags, status     |
| `get`                     | Retrieve a single group (optionally with role memberships)                  |
| `update`                  | Update a group’s name, description, tags, or metadata                       |
| `enable` / `disable`      | Enable or disable a group                                                   |
| `delete`                  | Permanently delete a group                                                  |
| `add-parent` / `remove-parent` | Assign or remove a parent group                                        |
| `add-children` / `remove-children` | Attach or detach child groups (or remove all children)            |
| `list-children`           | List children at specific depth ranges                                      |
| `hierarchy`               | Fetch ancestors/descendants as a tree or list                               |
| `roles`                   | Create/list/update/delete group roles; manage role actions and members      |

### API Examples

#### Create a Group

```bash
curl -X POST http://localhost:9004/<domainID>/groups \
  -H "Authorization: Bearer <your_access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "edge-devices",
    "description": "All edge devices",
    "metadata": { "region": "eu-west-1" },
    "tags": ["iot","edge"],
    "parent_id": "",
    "status": "enabled"
  }'
```

#### List Groups (flat)

```bash
curl -X GET "http://localhost:9004/<domainID>/groups?limit=10&status=enabled" \
  -H "Authorization: Bearer <your_access_token>"
```

#### Retrieve a Group (with Roles)

```bash
curl -X GET "http://localhost:9004/<domainID>/groups/<groupID>?roles=true" \
  -H "Authorization: Bearer <your_access_token>"
```

#### Update a Group

```bash
curl -X PUT http://localhost:9004/<domainID>/groups/<groupID> \
  -H "Authorization: Bearer <your_access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "edge-ops",
    "description": "Edge operations",
    "metadata": { "region": "eu-west-1", "env": "prod" },
    "tags": ["iot","ops"]
  }'
```

#### Enable or Disable a Group

```bash
curl -X POST http://localhost:9004/<domainID>/groups/<groupID>/enable \
  -H "Authorization: Bearer <your_access_token>"

curl -X POST http://localhost:9004/<domainID>/groups/<groupID>/disable \
  -H "Authorization: Bearer <your_access_token>"
```

#### Manage Hierarchy

```bash
# Add a parent
curl -X POST http://localhost:9004/<domainID>/groups/<groupID>/parents \
  -H "Authorization: Bearer <your_access_token>" \
  -H "Content-Type: application/json" \
  -d '{ "parent_id": "<parentID>" }'

# List children between levels 1 and 2
curl -X GET "http://localhost:9004/<domainID>/groups/<groupID>/children?start_level=1&end_level=2&limit=10" \
  -H "Authorization: Bearer <your_access_token>"
```

## Roles Management for Groups

Group roles use the shared role manager. Supported operations mirror domain roles (create, list, view, update, delete roles; add/list/remove actions; add/list/remove members; list available actions).

Example: create a group role

```bash
curl -X POST http://localhost:9004/<domainID>/groups/<groupID>/roles \
  -H "Authorization: Bearer <your_access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "role_name": "group-admin",
    "optional_actions": ["manage_role_permission", "update_permission"],
    "optional_members": ["<userID>"]
  }'
```

List available actions for groups:

```bash
curl -X GET http://localhost:9004/<domainID>/groups/roles/available-actions \
  -H "Authorization: Bearer <your_access_token>"
```

## Implementation Details

- Groups are stored in PostgreSQL with `ltree` paths for hierarchy queries; domain migrations are applied alongside group migrations for referential integrity.
- Role tables are provisioned per entity with a `groups_` prefix.
- Event notifications are published to `MG_ES_URL`; domain events are consumed to keep group data aligned.
- Authorization and roles are enforced through SpiceDB and shared policy middleware.
- Optional HTTP callouts (pre-operation hooks) are controlled via `MG_GROUPS_CALLOUT_*`.
- Observability: Jaeger tracing, Prometheus metrics at `/metrics`, and a `/health` endpoint.

### Groups Table

| Column        | Type          | Description                                                        |
| ------------- | ------------- | ------------------------------------------------------------------ |
| `id`          | VARCHAR(36)   | UUID of the group (primary key)                                    |
| `parent_id`   | VARCHAR(36)   | Optional parent group (self-referential FK)                        |
| `domain_id`   | VARCHAR(36)   | Owning domain                                                      |
| `name`        | VARCHAR(1024) | Group name                                                         |
| `description` | VARCHAR(1024) | Optional description                                               |
| `metadata`    | JSONB         | Arbitrary metadata                                                 |
| `tags`        | TEXT[]        | Group tags                                                         |
| `path`        | LTREE         | Hierarchical path for fast ancestor/descendant queries             |
| `created_at`  | TIMESTAMPTZ   | Creation timestamp                                                 |
| `updated_at`  | TIMESTAMPTZ   | Last update timestamp                                              |
| `updated_by`  | VARCHAR(254)  | Actor who last updated the group                                   |
| `status`      | SMALLINT      | 0 = enabled, 1 = disabled, 2 = deleted                             |

## Best Practices

- Model hierarchy deliberately: keep depth reasonable and avoid cycles by design.
- Use tags/metadata to segment groups by environment, region, or ownership for filtering.
- Prefer `disable` before `delete` when you need reversible off-boarding.
- Use roles sparingly and audit with `list-role-members`; grant only required actions.
- Fetch children with bounded levels to keep queries efficient.
- Limit callouts to necessary operations via `MG_GROUPS_CALLOUT_OPERATIONS`.

## Versioning and Health Check

The Groups service exposes `/health` with status and build metadata.

```bash
curl -X GET http://localhost:9004/health \
  -H "accept: application/health+json"
```

Example response:

```json
{
  "status": "pass",
  "version": "0.18.0",
  "commit": "7d6f4dc4f7f0c1fa3dc24eddfb18bb5073ff4f62",
  "description": "groups service",
  "build_time": "1970-01-01_00:00:00"
}
```

For full API coverage, see the [Groups API documentation](https://docs.api.supermq.absmach.eu/?urls.primaryName=api%2Fgroups.yaml).

[doc]: https://docs.supermq.absmach.eu/
