# Domains

The Domains service provides an HTTP API for managing platform domains in SuperMQ. Through this API you can create, list, retrieve, update, enable/disable/freeze domains, manage roles & invitations associated with domains, and more.

For more background on SuperMQ concepts, see the [official documentation][doc].

## Configuration

The service is configured through environment variables (unset variables fall back to defaults).

| Variable                               | Description                                                               | Default                            |
| -------------------------------------- | ------------------------------------------------------------------------- | ---------------------------------- |
| `SMQ_DOMAINS_LOG_LEVEL`                | Log level for Domains (debug, info, warn, error)                          | debug                              |
| `SMQ_DOMAINS_HTTP_HOST`                | Domains service HTTP host                                                 | domains                            |
| `SMQ_DOMAINS_HTTP_PORT`                | Domains service HTTP port                                                 | 9003                               |
| `SMQ_DOMAINS_HTTP_SERVER_CERT`         | Path to PEM-encoded HTTP server certificate                               | ""                                 |
| `SMQ_DOMAINS_HTTP_SERVER_KEY`          | Path to PEM-encoded HTTP server key                                       | ""                                 |
| `SMQ_DOMAINS_GRPC_PORT`                | Domains service gRPC port                                                 | 7003                               |
| `SMQ_DOMAINS_GRPC_SERVER_CERT`         | Path to PEM-encoded gRPC server certificate                               | ""                                 |
| `SMQ_DOMAINS_GRPC_SERVER_KEY`          | Path to PEM-encoded gRPC server key                                       | ""                                 |
| `SMQ_DOMAINS_GRPC_SERVER_CA_CERTS`     | Path to trusted CA bundle for the gRPC server                             | ""                                 |
| `SMQ_DOMAINS_GRPC_CLIENT_CA_CERTS`     | Path to client CA bundle to require gRPC mTLS                             | ""                                 |
| `SMQ_DOMAINS_DB_HOST`                  | Database host address                                                     | domains-db                         |
| `SMQ_DOMAINS_DB_PORT`                  | Database host port                                                        | 5432                               |
| `SMQ_DOMAINS_DB_USER`                  | Database user                                                             | supermq                            |
| `SMQ_DOMAINS_DB_PASS`                  | Database password                                                         | supermq                            |
| `SMQ_DOMAINS_DB_NAME`                  | Name of the database used by the service                                  | domains                            |
| `SMQ_DOMAINS_DB_SSL_MODE`              | Database connection SSL mode (disable, require, verify-ca, verify-full)   | ""                                 |
| `SMQ_DOMAINS_DB_SSL_CERT`              | Path to the PEM-encoded certificate file                                  | ""                                 |
| `SMQ_DOMAINS_DB_SSL_KEY`               | Path to the PEM-encoded key file                                          | ""                                 |
| `SMQ_DOMAINS_DB_SSL_ROOT_CERT`         | Path to the PEM-encoded root certificate file                             | ""                                 |
| `SMQ_DOMAINS_CACHE_URL`                | Cache database URL                                                        | redis://domains-redis:6379/0       |
| `SMQ_DOMAINS_CACHE_KEY_DURATION`       | Cache key duration for domain status/route lookups                        | 10m                                |
| `SMQ_DOMAINS_INSTANCE_ID`              | Domains instance ID (auto-generated when empty)                           | ""                                 |
| `SMQ_SPICEDB_HOST`                     | SpiceDB host for policy checks                                            | supermq-spicedb                    |
| `SMQ_SPICEDB_PORT`                     | SpiceDB port                                                              | 50051                              |
| `SMQ_SPICEDB_SCHEMA_FILE`              | Path to SpiceDB schema file used to seed available actions                | ./docker/spicedb/schema.schema.zed |
| `SMQ_SPICEDB_PRE_SHARED_KEY`           | SpiceDB preshared key                                                     | 12345678                           |
| `SMQ_ES_URL`                           | Event store URL                                                           | nats://localhost:4222              |
| `SMQ_JAEGER_URL`                       | Jaeger server URL                                                         | <http://localhost:4318/v1/traces>  |
| `SMQ_JAEGER_TRACE_RATIO`               | Trace sampling ratio                                                      | 1.0                                |
| `SMQ_SEND_TELEMETRY`                   | Send telemetry to the SuperMQ call-home server                            | true                               |
| `SMQ_AUTH_GRPC_URL`                    | Auth service gRPC URL                                                     | ""                                 |
| `SMQ_AUTH_GRPC_TIMEOUT`                | Auth service gRPC request timeout                                         | 1s                                 |
| `SMQ_AUTH_GRPC_CLIENT_CERT`            | Path to the PEM-encoded Auth gRPC client certificate                      | ""                                 |
| `SMQ_AUTH_GRPC_CLIENT_KEY`             | Path to the PEM-encoded Auth gRPC client key                              | ""                                 |
| `SMQ_AUTH_GRPC_SERVER_CA_CERTS`        | Path to the PEM-encoded Auth gRPC trusted CA bundle                       | ""                                 |
| `SMQ_DOMAINS_CALLOUT_URLS`             | Comma-separated list of HTTP callout targets invoked on domain operations | ""                                 |
| `SMQ_DOMAINS_CALLOUT_METHOD`           | HTTP method for callouts (POST or GET)                                    | POST                               |
| `SMQ_DOMAINS_CALLOUT_TLS_VERIFICATION` | Verify TLS certificates for callouts                                      | true                               |
| `SMQ_DOMAINS_CALLOUT_TIMEOUT`          | Callout request timeout                                                   | 10s                                |
| `SMQ_DOMAINS_CALLOUT_KEY`              | Client key for mTLS callouts                                              | ""                                 |
| `SMQ_DOMAINS_CALLOUT_OPERATIONS`       | Comma-separated list of operation names that should trigger callouts      | ""                                 |
| `SMQ_DOMAINS_DELETE_INTERVAL`          | Interval between checks for domains to delete                             | 24h                                |
| `SMQ_DOMAINS_DELETE_AFTER`             | Duration after which domains of deleted status are deleted                | 720h                               |

**Note**: Set `SMQ_DOMAINS_CALLOUT_OPERATIONS` to a subset of `OpCreateDomain`, `OpRetrieveDomain`, `OpUpdateDomain`, `OpEnableDomain`, `OpDisableDomain`, `OpFreezeDomain`, `OpListDomains`, `OpViewDomainInvitation`, `OpSendInvitation`, `OpAcceptInvitation`, `OpListInvitations`, `OpListDomainInvitations`, `OpRejectInvitation`, or `OpDeleteInvitation` to filter which actions produce callouts.

## Deployment

The service is distributed as a Docker container. See the [`domains` section](https://github.com/absmach/supermq/blob/main/docker/docker-compose.yaml#L215-L310) of the compose file for an example deployment.

To run the service outside of a container:

```bash
# download the latest version of the service
git clone https://github.com/absmach/supermq
cd supermq

# compile the domains service
make domains

# copy binary to $GOBIN
make install

# set the environment variables and run the service
SMQ_DOMAINS_LOG_LEVEL=debug \
SMQ_DOMAINS_CACHE_URL=redis://domains-redis:6379/0 \
SMQ_DOMAINS_CACHE_KEY_DURATION=10m \
SMQ_DOMAINS_HTTP_HOST=domains \
SMQ_DOMAINS_HTTP_PORT=9003 \
SMQ_DOMAINS_HTTP_SERVER_CERT="" \
SMQ_DOMAINS_HTTP_SERVER_KEY="" \
SMQ_DOMAINS_GRPC_HOST=domains \
SMQ_DOMAINS_GRPC_PORT=7003 \
SMQ_DOMAINS_GRPC_SERVER_CERT="" \
SMQ_DOMAINS_GRPC_SERVER_KEY="" \
SMQ_DOMAINS_GRPC_SERVER_CA_CERTS="" \
SMQ_DOMAINS_GRPC_CLIENT_CA_CERTS="" \
SMQ_DOMAINS_DB_HOST=domains-db \
SMQ_DOMAINS_DB_PORT=5432 \
SMQ_DOMAINS_DB_USER=supermq \
SMQ_DOMAINS_DB_PASS=supermq \
SMQ_DOMAINS_DB_NAME=domains \
SMQ_DOMAINS_DB_SSL_MODE="" \
SMQ_DOMAINS_DB_SSL_CERT="" \
SMQ_DOMAINS_DB_SSL_KEY="" \
SMQ_DOMAINS_DB_SSL_ROOT_CERT="" \
SMQ_AUTH_GRPC_URL="" \
SMQ_AUTH_GRPC_TIMEOUT=1s \
SMQ_AUTH_GRPC_CLIENT_CERT="" \
SMQ_AUTH_GRPC_CLIENT_KEY="" \
SMQ_AUTH_GRPC_SERVER_CA_CERTS="" \
SMQ_SPICEDB_HOST=localhost \
SMQ_SPICEDB_PORT=50051 \
SMQ_SPICEDB_SCHEMA_FILE=./docker/spicedb/schema.schema.zed \
SMQ_SPICEDB_PRE_SHARED_KEY=12345678 \
SMQ_ES_URL=nats://localhost:4222 \
SMQ_JAEGER_URL=<http://localhost:4318/v1/traces> \
SMQ_JAEGER_TRACE_RATIO=1.0 \
SMQ_DOMAINS_CALLOUT_URLS="" \
SMQ_DOMAINS_CALLOUT_METHOD=POST \
SMQ_DOMAINS_CALLOUT_TLS_VERIFICATION=true \
SMQ_DOMAINS_CALLOUT_TIMEOUT=10s \
SMQ_DOMAINS_CALLOUT_KEY="" \
SMQ_DOMAINS_CALLOUT_OPERATIONS="" \
SMQ_SEND_TELEMETRY=true \
SMQ_DOMAINS_INSTANCE_ID="" \
$GOBIN/supermq-domains
```

## Usage

Domains supports the following operations:

| Operation           | Description                                                                           |
| ------------------- | ------------------------------------------------------------------------------------- |
| `create`            | Create a new domain with a unique route                                               |
| `get`               | Retrieve a domain (optionally with role memberships) or list accessible domains       |
| `update`            | Update a domain’s name, tags, or metadata                                             |
| `enable`            | Enable a previously disabled domain                                                   |
| `disable`           | Disable an active domain                                                              |
| `freeze`            | Freeze a domain (platform administrators only)                                        |
| `invite`            | Send an invitation for a user to join a domain with a specific role                   |
| `invitations`       | List invitations for the current user or for a specific domain                        |
| `accept/reject`     | Accept or reject a pending domain invitation                                          |
| `delete-invitation` | Delete an invitation (inviter, invitee, or admin)                                     |
| `roles`             | Create/list/update/delete domain roles; manage role actions and members; list actions |

### API Examples

#### Create a Domain

```bash
curl -X POST http://localhost:9004/domains \
  -H "Authorization: Bearer <your_access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Edge Tenant",
    "route": "edge",
    "tags": ["iot", "prod"],
    "metadata": { "region": "eu-west-1" }
  }'
```

Expected response:

```json
{
  "id": "f2b16e2c-5ad1-4c44-9c1b-0f862ec1c0c8",
  "name": "Edge Tenant",
  "tags": ["iot", "prod"],
  "route": "edge",
  "metadata": { "region": "eu-west-1" },
  "status": "enabled",
  "created_by": "a5b6c7d8-e901-4fab-9bcd-123456789abc",
  "created_at": "2024-10-24T13:31:52Z"
}
```

#### List Domains

```bash
curl -X GET "http://localhost:9004/domains?limit=10&status=enabled" \
  -H "Authorization: Bearer <your_access_token>"
```

```json
{
  "total": 2,
  "offset": 0,
  "limit": 10,
  "domains": [
    {
      "id": "f2b16e2c-5ad1-4c44-9c1b-0f862ec1c0c8",
      "name": "Edge Tenant",
      "route": "edge",
      "status": "enabled",
      "created_at": "2024-10-24T13:31:52Z"
    },
    {
      "id": "7f6a5b4c-3210-4fed-ba98-76543210fedc",
      "name": "Sandbox",
      "route": "sandbox",
      "status": "disabled",
      "created_at": "2024-10-10T08:12:04Z"
    }
  ]
}
```

#### Retrieve a Domain (with Roles)

```bash
curl -X GET "http://localhost:9004/domains/<domainID>?roles=true" \
  -H "Authorization: Bearer <your_access_token>"
```

```json
{
  "id": "f2b16e2c-5ad1-4c44-9c1b-0f862ec1c0c8",
  "name": "Edge Tenant",
  "route": "edge",
  "status": "enabled",
  "roles": [
    {
      "role_id": "b83d25e7-6a49-4c2e-98c5-9323d4d9af7d",
      "role_name": "admin",
      "actions": [
        "manage_role_permission",
        "update_permission",
        "add_role_users_permission"
      ]
    }
  ],
  "created_at": "2024-10-24T13:31:52Z",
  "updated_at": "2024-10-24T13:31:52Z"
}
```

#### Update a Domain

```bash
curl -X PATCH http://localhost:9004/domains/<domainID> \
  -H "Authorization: Bearer <your_access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Edge Operations",
    "tags": ["iot", "ops"],
    "metadata": { "region": "eu-west-1", "env": "prod" }
  }'
```

#### Enable, Disable, or Freeze a Domain

```bash
curl -X POST http://localhost:9004/domains/<domainID>/disable \
  -H "Authorization: Bearer <your_access_token>"

curl -X POST http://localhost:9004/domains/<domainID>/enable \
  -H "Authorization: Bearer <your_access_token>"

curl -X POST http://localhost:9004/domains/<domainID>/freeze \
  -H "Authorization: Bearer <your_access_token>"
```

#### Send an Invitation

```bash
curl -X POST http://localhost:9004/domains/<domainID>/invitations \
  -H "Authorization: Bearer <your_access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "invitee_user_id": "<userID>",
    "role_id": "<roleID>",
    "resend": false
  }'
```

#### List Domain or User Invitations

```bash
# For a specific domain
curl -X GET "http://localhost:9004/domains/<domainID>/invitations?limit=10&state=pending" \
  -H "Authorization: Bearer <your_access_token>"

# For the current user
curl -X GET "http://localhost:9004/invitations?limit=10" \
  -H "Authorization: Bearer <your_access_token>"
```

```json
{
  "total": 1,
  "offset": 0,
  "limit": 10,
  "invitations": [
    {
      "invited_by": "a5b6c7d8-e901-4fab-9bcd-123456789abc",
      "invitee_user_id": "2c4d6e8f-0a12-4b3c-9d8e-7f6a5b4c3d2e",
      "domain_id": "f2b16e2c-5ad1-4c44-9c1b-0f862ec1c0c8",
      "domain_name": "Edge Tenant",
      "role_id": "b83d25e7-6a49-4c2e-98c5-9323d4d9af7d",
      "role_name": "admin",
      "actions": ["manage_role_permission", "update_permission"],
      "created_at": "2024-10-25T11:03:42Z"
    }
  ]
}
```

#### Accept, Reject, or Delete an Invitation

```bash
# Accept
curl -X POST http://localhost:9004/invitations/accept \
  -H "Authorization: Bearer <your_access_token>" \
  -H "Content-Type: application/json" \
  -d '{ "domain_id": "<domainID>" }'

# Reject
curl -X POST http://localhost:9004/invitations/reject \
  -H "Authorization: Bearer <your_access_token>" \
  -H "Content-Type: application/json" \
  -d '{ "domain_id": "<domainID>" }'

# Delete
curl -X DELETE http://localhost:9004/domains/<domainID>/invitations \
  -H "Authorization: Bearer <your_access_token>" \
  -H "Content-Type: application/json" \
  -d '{ "user_id": "<inviteeUserID>" }'
```

## Roles Management for Domains

Domain roles reuse the shared role manager. Supported operations:

| Operation                 | Description                                                          |
| ------------------------- | -------------------------------------------------------------------- |
| `create-role`             | Create a new role for a domain                                       |
| `list-roles`              | List all roles assigned to a domain                                  |
| `get-role`                | Retrieve details for a specific domain role                          |
| `update-role`             | Update a domain role name                                            |
| `delete-role`             | Delete a domain role                                                 |
| `add-role-action`         | Add one or more actions to a domain role                             |
| `list-role-actions`       | List all actions associated with a domain role                       |
| `delete-role-action`      | Remove a specific action from a domain role                          |
| `delete-all-role-actions` | Remove all actions from a domain role                                |
| `add-role-member`         | Associate one or more members with a domain role                     |
| `list-role-members`       | List all members of a domain role                                    |
| `delete-role-member`      | Remove one or more members from a domain role                        |
| `delete-all-role-members` | Remove all members from a domain role                                |
| `list-available-actions`  | Retrieve the global list of available domain actions from the schema |

Example: create a domain role

```bash
curl -X POST http://localhost:9004/domains/<domainID>/roles \
  -H "Authorization: Bearer <your_access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "role_name": "domain-editor",
    "optional_actions": ["update_permission", "read_permission"],
    "optional_members": ["<userID>"]
  }'
```

To discover allowed actions before creating roles, call:

```bash
curl -X GET http://localhost:9004/domains/roles/available-actions \
  -H "Authorization: Bearer <your_access_token>"
```

## Implementation Details

- Domains and invitations are persisted in PostgreSQL; migrations also create role tables with a `domains_` prefix.
- Redis caches domain status and route-to-ID lookups to speed up authorization.
- Domain lifecycle events are published to the configured event store (`SMQ_ES_URL`).
- Authorization and role checks are enforced via SpiceDB-backed policy service.
- Optional HTTP callouts can be triggered before operations, using the `SMQ_DOMAINS_CALLOUT_*` settings.
- Observability: Jaeger tracing, Prometheus metrics at `/metrics`, and a `/health` endpoint.

### Domains Table

| Column       | Type         | Description                                         |
| ------------ | ------------ | --------------------------------------------------- |
| `id`         | VARCHAR(36)  | UUID of the domain (primary key)                    |
| `name`       | VARCHAR(254) | Human-readable domain name                          |
| `tags`       | TEXT[]       | Domain tags                                         |
| `metadata`   | JSONB        | Arbitrary metadata                                  |
| `route`      | VARCHAR(254) | Unique domain route/alias                           |
| `created_at` | TIMESTAMPTZ  | Creation timestamp                                  |
| `updated_at` | TIMESTAMPTZ  | Last update timestamp                               |
| `updated_by` | VARCHAR(254) | Actor who last updated the domain                   |
| `created_by` | VARCHAR(254) | Actor who created the domain                        |
| `status`     | SMALLINT     | 0 = enabled, 1 = disabled, 2 = freezed, 3 = deleted |

### Invitations Table

| Column            | Type        | Description                                      |
| ----------------- | ----------- | ------------------------------------------------ |
| `invited_by`      | VARCHAR(36) | User who sent the invitation                     |
| `invitee_user_id` | VARCHAR(36) | User being invited                               |
| `domain_id`       | VARCHAR(36) | Domain to join (FK to `domains.id`)              |
| `role_id`         | VARCHAR(36) | Role to grant on acceptance                      |
| `created_at`      | TIMESTAMPTZ | Invitation creation time                         |
| `updated_at`      | TIMESTAMPTZ | Last modification time                           |
| `confirmed_at`    | TIMESTAMPTZ | When the invitation was accepted (if applicable) |
| `rejected_at`     | TIMESTAMPTZ | When the invitation was rejected (if applicable) |

## Best Practices

- Reserve concise, DNS-friendly `route` values for external-facing domains.
- Use metadata and tags to capture environment, region, and ownership for filtering.
- Prefer `disable` over delete when you need reversible off-boarding; use `freeze` for emergency locks by admins.
- Keep role definitions minimal; grant only the actions needed and audit with `list-role-members`.
- Clean up stale invitations regularly using the domain/user invitation listing endpoints.
- When enabling callouts, narrow `SMQ_DOMAINS_CALLOUT_OPERATIONS` to the events you must observe.

## Versioning and Health Check

The Domains service exposes `/health` with status and build metadata.

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
  "description": "domains service",
  "build_time": "1970-01-01_00:00:00"
}
```

For full API coverage, see the [Domains API documentation](https://docs.api.supermq.absmach.eu/?urls.primaryName=api%2Fdomains.yaml).

[doc]: https://docs.supermq.absmach.eu/
