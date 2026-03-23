# Clients

Clients service provides an HTTP API for managing platform resources: `clients` and `channels`.
Through this API clients are able to do the following actions:

- provision new clients
- create new channels
- "connect" clients into the channels

For an in-depth explanation of the aforementioned scenarios, as well as thorough
understanding of SuperMQ, please check out the [official documentation][doc].

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                       | Description                                                             | Default                        |
| ------------------------------ | ----------------------------------------------------------------------- | ------------------------------ |
| MG_CLIENTS_LOG_LEVEL          | Log level for Clients (debug, info, warn, error)                        | info                           |
| MG_CLIENTS_HTTP_HOST          | Clients service HTTP host                                               | localhost                      |
| MG_CLIENTS_HTTP_PORT          | Clients service HTTP port                                               | 9000                           |
| MG_CLIENTS_SERVER_CERT        | Path to the PEM encoded server certificate file                         | ""                             |
| MG_CLIENTS_SERVER_KEY         | Path to the PEM encoded server key file                                 | ""                             |
| MG_CLIENTS_GRPC_HOST          | Clients service gRPC host                                               | localhost                      |
| MG_CLIENTS_GRPC_PORT          | Clients service gRPC port                                               | 7000                           |
| MG_CLIENTS_GRPC_SERVER_CERT   | Path to the PEM encoded server certificate file                         | ""                             |
| MG_CLIENTS_GRPC_SERVER_KEY    | Path to the PEM encoded server key file                                 | ""                             |
| MG_CLIENTS_DB_HOST            | Database host address                                                   | localhost                      |
| MG_CLIENTS_DB_PORT            | Database host port                                                      | 5432                           |
| MG_CLIENTS_DB_USER            | Database user                                                           | supermq                        |
| MG_CLIENTS_DB_PASS            | Database password                                                       | supermq                        |
| MG_CLIENTS_DB_NAME            | Name of the database used by the service                                | clients                        |
| MG_CLIENTS_DB_SSL_MODE        | Database connection SSL mode (disable, require, verify-ca, verify-full) | disable                        |
| MG_CLIENTS_DB_SSL_CERT        | Path to the PEM encoded certificate file                                | ""                             |
| MG_CLIENTS_DB_SSL_KEY         | Path to the PEM encoded key file                                        | ""                             |
| MG_CLIENTS_DB_SSL_ROOT_CERT   | Path to the PEM encoded root certificate file                           | ""                             |
| MG_CLIENTS_CACHE_URL          | Cache database URL                                                      | <redis://localhost:6379/0>     |
| MG_CLIENTS_CACHE_KEY_DURATION | Cache key duration in seconds                                           | 3600                           |
| MG_CLIENTS_ES_URL             | Event store URL                                                         | <localhost:6379>               |
| MG_CLIENTS_ES_PASS            | Event store password                                                    | ""                             |
| MG_CLIENTS_ES_DB              | Event store instance name                                               | 0                              |
| MG_CLIENTS_STANDALONE_ID      | User ID for standalone mode (no gRPC communication with Auth)           | ""                             |
| MG_CLIENTS_STANDALONE_TOKEN   | User token for standalone mode that should be passed in auth header     | ""                             |
| MG_JAEGER_URL                 | Jaeger server URL                                                       | <http://jaeger:4318/v1/traces> |
| MG_AUTH_GRPC_URL              | Auth service gRPC URL                                                   | localhost:7001                 |
| MG_AUTH_GRPC_TIMEOUT          | Auth service gRPC request timeout in seconds                            | 1s                             |
| MG_AUTH_GRPC_CLIENT_TLS       | Enable TLS for gRPC client                                              | false                          |
| MG_AUTH_GRPC_CA_CERT          | Path to the CA certificate file                                         | ""                             |
| MG_SEND_TELEMETRY             | Send telemetry to supermq call home server.                             | true                           |
| Clients_INSTANCE_ID            | Clients instance ID                                                     | ""                             |

**Note** that if you want `clients` service to have only one user locally, you should use `CLIENTS_STANDALONE` env vars. By specifying these, you don't need `auth` service in your deployment for users' authorization.

## Deployment

The service itself is distributed as Docker container. Check the [`clients`](https://github.com/absmach/supermq/blob/main/docker/docker-compose.yaml#L167-L194) service section in
docker-compose file to see how service is deployed.

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/supermq

cd supermq

# compile the clients
make clients

# copy binary to bin
make install

# set the environment variables and run the service
Clients_LOG_LEVEL=[Clients log level] \
Clients_STANDALONE_ID=[User ID for standalone mode (no gRPC communication with auth)] \
Clients_STANDALONE_TOKEN=[User token for standalone mode that should be passed in auth header] \
Clients_CACHE_KEY_DURATION=[Cache key duration in seconds] \
Clients_HTTP_HOST=[Clients service HTTP host] \
Clients_HTTP_PORT=[Clients service HTTP port] \
Clients_HTTP_SERVER_CERT=[Path to server certificate in pem format] \
Clients_HTTP_SERVER_KEY=[Path to server key in pem format] \
Clients_AUTH_GRPC_HOST=[Clients service gRPC host] \
Clients_AUTH_GRPC_PORT=[Clients service gRPC port] \
Clients_AUTH_GRPC_SERVER_CERT=[Path to server certificate in pem format] \
Clients_AUTH_GRPC_SERVER_KEY=[Path to server key in pem format] \
Clients_DB_HOST=[Database host address] \
Clients_DB_PORT=[Database host port] \
Clients_DB_USER=[Database user] \
Clients_DB_PASS=[Database password] \
Clients_DB_NAME=[Name of the database used by the service] \
Clients_DB_SSL_MODE=[SSL mode to connect to the database with] \
Clients_DB_SSL_CERT=[Path to the PEM encoded certificate file] \
Clients_DB_SSL_KEY=[Path to the PEM encoded key file] \
Clients_DB_SSL_ROOT_CERT=[Path to the PEM encoded root certificate file] \
Clients_CACHE_URL=[Cache database URL] \
Clients_ES_URL=[Event store URL] \
Clients_ES_PASS=[Event store password] \
Clients_ES_DB=[Event store instance name] \
MG_AUTH_GRPC_URL=[Auth service gRPC URL] \
MG_AUTH_GRPC_TIMEOUT=[Auth service gRPC request timeout in seconds] \
MG_AUTH_GRPC_CLIENT_TLS=[Enable TLS for gRPC client] \
MG_AUTH_GRPC_CA_CERT=[Path to trusted CA certificate file] \
MG_JAEGER_URL=[Jaeger server URL] \
MG_SEND_TELEMETRY=[Send telemetry to supermq call home server] \
Clients_INSTANCE_ID=[Clients instance ID] \
$GOBIN/supermq-clients
```

Setting `Clients_CA_CERTS` expects a file in PEM format of trusted CAs. This will enable TLS against the Auth gRPC endpoint trusting only those CAs that are provided.

In constrained environments, sometimes it makes sense to run Clients service as a standalone to reduce network traffic and simplify deployment. This means that Clients service
operates only using a single user and is able to authorize it without gRPC communication with Auth service.
To run service in a standalone mode, set `Clients_STANDALONE_EMAIL` and `Clients_STANDALONE_TOKEN`.

## Usage

SuperMQ supports the following operations for Clients:

| Operation     | Description                                                        |
| ------------- | ------------------------------------------------------------------ |
| `create`      | Create a new client                                                |
| `get`         | Retrieve a single client or list all clients                       |
| `update`      | Update a client’s name and metadata                                |
| `delete`      | Permanently delete a client                                        |
| `enable`      | Enable a previously disabled client                                |
| `disable`     | Disable an active client                                           |
| `setClientParentGroup`     | Add a Parent Group to a client                  |
| `removeClientParentGroup`  |  Remove a Parent Group from a client      |

### API Examples

#### Create a Client

```bash
curl -X POST http://localhost:9006/<domainID>/clients \
  -H "Authorization: Bearer <your_access_token>" \
  -H "Content-Type: application/json" \
  -d '{
  "name": "clientName",
  "tags": [
    "tag1",
    "tag2"
  ],
  "credentials": {
    "identity": "clientIDentity",
    "secret": "bb7edb32-2eac-4aad-aebe-ed96fe073879"
  },
  "metadata": {
    "model": "example"
  },
  "status": "enabled"
}'
```

The expected response should be:

```bash
{
  "id": "bb7edb32-2eac-4aad-aebe-ed96fe073879",
  "name": "clientName",
  "tags": [
    "tag1",
    "tag2"
  ],
  "domain_id": "bb7edb32-2eac-4aad-aebe-ed96fe073879",
  "credentials": {
    "identity": "clientIDentity",
    "secret": "bb7edb32-2eac-4aad-aebe-ed96fe073879"
  },
  "metadata": {
    "model": "example"
  },
  "status": "enabled",
  "created_at": "2019-11-26 13:31:52",
  "updated_at": "2019-11-26 13:31:52"
}
```

#### Get Clients

List all clients:

```bash
curl -X GET "http://localhost:9006/<domainID>/clients?limit=10" \
  -H "Authorization: Bearer <your_access_token>"
```

List a singular client:

```bash
curl -X GET http://localhost:9006/<domainID>/clients/<clientID> \
  -H "Authorization: Bearer <your_access_token>"
```

#### Update a Client

Update is performed by replacing the current resource data with values provided in a request payload. Note that the client's type and ID cannot be changed.

```bash
curl -X PATCH http://localhost:9006/<domainID>/clients/<clientID> \
  -H "Authorization: Bearer <your_access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "clientName",
    "metadata": {"role": "general"}
  }'
```

The expected response is

```bash
{
  "id": "bb7edb32-2eac-4aad-aebe-ed96fe073879",
  "name": "clientName",
  "tags": [
    "tag1",
    "tag2"
  ],
  "domain_id": "bb7edb32-2eac-4aad-aebe-ed96fe073879",
  "credentials": {
    "identity": "clientIDentity",
    "secret": "bb7edb32-2eac-4aad-aebe-ed96fe073879"
  },
  "metadata": { "model": "example" },
  "status": "enabled",
  "created_at": "2019-11-26 13:31:52",
  "updated_at": "2019-11-26 13:31:52"
}
```

#### Delete a Client

Delete client removes a client with the given id from repo and removes all the policies related to this client.

```bash
curl -X DELETE http://localhost:9006/<domainID>/clients/<clientID> \
  -H "Authorization: Bearer <your_access_token>"
```

#### Disable a Client

Disables a specific client that is identified by the client ID.

```bash
curl -X POST http://localhost:9006/<domainID>/clients/<clientID>/disable \
  -H "Authorization: Bearer <your_access_token>"
```

#### Enable a Client

Enable logically enables the client identified with the provided ID

```bash
curl -X POST http://localhost:9006/<domainID>/clients/<clientID>/enable \
  -H "Authorization: Bearer <your_access_token>"
```

## Roles Management for Clients

In addition to standard client lifecycle operations (create, get, update, delete, enable, disable), the Clients service supports robust role‑based operations for managing permissions and associations for each client.

### Supported Role Operations

| Operation                     | Description                                                                 |
|------------------------------|-----------------------------------------------------------------------------|
| `create-role`                | Create a new role for a client                                             |
| `list-roles`                 | List all roles assigned to a client                                         |
| `get-role`                   | Retrieve details for a specific client role                                 |
| `update-role`                | Update a specific client role                                               |
| `delete-role`                | Delete a specific client role                                               |
| `add-role-action`            | Add one or more actions (permissions) to a client role                        |
| `list-role-actions`          | List all actions associated with a client role                              |
| `delete-role-action`         | Remove a specific action from a client role                                 |
| `delete-all-role-actions`    | Remove all actions from a client role                                       |
| `add-role-member`            | Associate one or more users or entities to a client role                     |
| `list-role-members`          | List all members of a client role                                            |
| `delete-role-member`         | Remove one or more members from a client role                               |
| `delete-all-role-members`    | Remove all members from a client role                                       |
| `list-available-actions`     | Retrieve the global list of available actions key for role creation         |

### Example: Create a Client Role

```bash
curl -X POST http://localhost:9006/<domainID>/clients/<clientID>/roles \
  -H "Authorization: Bearer <your_access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "publisher",
    "actions": ["publish"],
    "members": []
  }'
```

## Implementation Details

Clients in SuperMQ are persisted in PostgreSQL using a schema optimized for identity management, authorization, and relationship tracking (channels, groups, and users).

### Clients Table Structure

The main `clients` table tracks all metadata, identity, and lifecycle information for each client:

| Column            | Type           | Description                                                                 |
|------------------ | -------------- | --------------------------------------------------------------------------- |
| `id`              | VARCHAR(36)    | UUID of the client (primary key).                                           |
| `name`            | VARCHAR(1024)  | Human‑readable name.                                                        |
| `domain_id`       | VARCHAR(36)    | Domain to which the client belongs.                                         |
| `parent_group_id` | VARCHAR(36)    | Optional group parent (for inheritance/scoping).                            |
| `identity`        | VARCHAR(254)   | Login identity (often an email or unique ID).                               |
| `secret`          | VARCHAR(4096)  | Hashed authentication secret.                                               |
| `tags`            | TEXT[]         | Arbitrary list of client tags.                                              |
| `metadata`        | JSONB          | Free‑form structured metadata.                                              |
| `created_at`      | TIMESTAMPTZ    | Timestamp when the client was created.                                      |
| `updated_at`      | TIMESTAMPTZ    | Timestamp when the client was last updated.                                 |
| `updated_by`      | VARCHAR(254)   | Identifier of the actor who performed the last update.                      |
| `status`          | SMALLINT       | 0 = enabled, 1 = disabled.                                                  |

#### Connections Table Structure

Client ↔ Channel relationships are stored in the `connections` table:

| Column        | Type         | Description                                                            |
|-------------- | ------------ | ---------------------------------------------------------------------- |
| `channel_id`  | VARCHAR(36)  | Channel UUID.                                                          |
| `domain_id`   | VARCHAR(36)  | Domain of the client & channel.                                        |
| `client_id`   | VARCHAR(36)  | Client UUID.                                                           |
| `type`        | SMALLINT     | Connection type: `1 = Publish`, `2 = Subscribe`.                       |

This guarantees that when a client is deleted, all channel connections are automatically removed.

## Best Practices

To ensure robust and secure usage of the Clients service, consider the following recommendations:

- **Use metadata and tags meaningfully**: Store useful attributes like model, location, environment (e.g., `production`, `test`) to filter and manage clients efficiently.
- **Keep credentials secure**: Rotate client secrets periodically. Avoid using guessable strings.
- **Disable unused clients**: Use the `disable` operation to revoke access instead of deleting clients when deactivation is preferred.
- **Audit regularly**: Periodically list client roles and connections to ensure expected configuration.
- **Prefer standalone mode for edge deployments**: Use environment variables to configure standalone mode in isolated environments without needing the Auth service.

## Versioning and Health Check

The Clients service exposes a `/health` endpoint to verify operational status and version information.

### Health Check Request

```bash
curl -X 'GET' \
  'http://localhost:9006/health' \
  -H 'accept: application/health+json'
```

The expected response is:

```bash
{
  "status": "pass",
  "version": "0.14.0",
  "commit": "7d6f4dc4f7f0c1fa3dc24eddfb18bb5073ff4f62",
  "description": "clients service",
  "build_time": "1970-01-01_00:00:00"
}
```

This endpoint can be used for monitoring, CI/CD readiness checks, or basic diagnostics.

For more information about service capabilities and its usage, please check out
the [API documentation](https://docs.api.supermq.absmach.eu/?urls.primaryName=api%2Fclients.yaml).

[doc]: https://docs.supermq.absmach.eu/
