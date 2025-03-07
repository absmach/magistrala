# Auth - Authentication and Authorization service

Auth service provides authentication features as an API for managing authentication keys as well as administering groups of entities - `clients` and `users`.

## Authentication

User service is using Auth service gRPC API to obtain login token or password reset token. Authentication key consists of the following fields:

- ID - key ID
- Type - one of the three types described below
- IssuerID - an ID of the SuperMQ User who issued the key
- Subject - user ID for which the key is issued
- IssuedAt - the timestamp when the key is issued
- ExpiresAt - the timestamp after which the key is invalid

There are four types of authentication keys:

- Access key - keys issued to the user upon login request
- Refresh key - keys used to generate new access keys
- Recovery key - password recovery key
- API key - keys issued upon the user request
- Invitation key - keys used to invite new users

Authentication keys are represented and distributed by the corresponding [JWT](jwt.io).

User keys are issued when user logs in. Each user request (other than `registration` and `login`) contains user key that is used to authenticate the user.

API keys are similar to the User keys. The main difference is that API keys have configurable expiration time. If no time is set, the key will never expire. For that reason, API keys are _the only key type that can be revoked_. This also means that, despite being used as a JWT, it requires a query to the database to validate the API key. The user with API key can perform all the same actions as the user with login key (can act on behalf of the user for Client, Channel, or user profile management), _except issuing new API keys_.

Recovery key is the password recovery key. It's short-lived token used for password recovery process.

For in-depth explanation of the aforementioned scenarios, as well as thorough understanding of SuperMQ, please check out the [official documentation][doc].

The following actions are supported:

- create (all key types)
- verify (all key types)
- obtain (API keys only)
- revoke (API keys only)

## Domains

Domains are used to group users and clients. Each domain has a unique alias that is used to identify the domain. Domains are used to group users and their entities.

Domain consists of the following fields:

- ID - UUID uniquely representing domain
- Name - name of the domain
- Tags - array of tags
- Metadata - Arbitrary, object-encoded domain's data
- Alias - unique alias of the domain
- CreatedAt - timestamp at which the domain is created
- UpdatedAt - timestamp at which the domain is updated
- UpdatedBy - user that updated the domain
- CreatedBy - user that created the domain
- Status - domain status

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                          | Description                                                             | Default                        |
| --------------------------------- | ----------------------------------------------------------------------- | ------------------------------ |
| SMQ_AUTH_LOG_LEVEL                | Log level for the Auth service (debug, info, warn, error)               | info                           |
| SMQ_AUTH_DB_HOST                  | Database host address                                                   | localhost                      |
| SMQ_AUTH_DB_PORT                  | Database host port                                                      | 5432                           |
| SMQ_AUTH_DB_USER                  | Database user                                                           | supermq                        |
| SMQ_AUTH_DB_PASSWORD              | Database password                                                       | supermq                        |
| SMQ_AUTH_DB_NAME                  | Name of the database used by the service                                | auth                           |
| SMQ_AUTH_DB_SSL_MODE              | Database connection SSL mode (disable, require, verify-ca, verify-full) | disable                        |
| SMQ_AUTH_DB_SSL_CERT              | Path to the PEM encoded certificate file                                | ""                             |
| SMQ_AUTH_DB_SSL_KEY               | Path to the PEM encoded key file                                        | ""                             |
| SMQ_AUTH_DB_SSL_ROOT_CERT         | Path to the PEM encoded root certificate file                           | ""                             |
| SMQ_AUTH_HTTP_HOST                | Auth service HTTP host                                                  | ""                             |
| SMQ_AUTH_HTTP_PORT                | Auth service HTTP port                                                  | 8189                           |
| SMQ_AUTH_HTTP_SERVER_CERT         | Path to the PEM encoded HTTP server certificate file                    | ""                             |
| SMQ_AUTH_HTTP_SERVER_KEY          | Path to the PEM encoded HTTP server key file                            | ""                             |
| SMQ_AUTH_GRPC_HOST                | Auth service gRPC host                                                  | ""                             |
| SMQ_AUTH_GRPC_PORT                | Auth service gRPC port                                                  | 8181                           |
| SMQ_AUTH_GRPC_SERVER_CERT         | Path to the PEM encoded gRPC server certificate file                    | ""                             |
| SMQ_AUTH_GRPC_SERVER_KEY          | Path to the PEM encoded gRPC server key file                            | ""                             |
| SMQ_AUTH_GRPC_SERVER_CA_CERTS     | Path to the PEM encoded gRPC server CA certificate file                 | ""                             |
| SMQ_AUTH_GRPC_CLIENT_CA_CERTS     | Path to the PEM encoded gRPC client CA certificate file                 | ""                             |
| SMQ_AUTH_SECRET_KEY               | String used for signing tokens                                          | secret                         |
| SMQ_AUTH_ACCESS_TOKEN_DURATION    | The access token expiration period                                      | 1h                             |
| SMQ_AUTH_REFRESH_TOKEN_DURATION   | The refresh token expiration period                                     | 24h                            |
| SMQ_AUTH_INVITATION_DURATION      | The invitation token expiration period                                  | 168h                           |
| SMQ_AUTH_CACHE_URL                | Redis URL for caching PAT scopes                                        | redis://localhost:6379/0       |
| SMQ_AUTH_CACHE_KEY_DURATION       | Duration for which PAT scope cache keys are valid                       | 10m                            |
| SMQ_SPICEDB_HOST                  | SpiceDB host address                                                    | localhost                      |
| SMQ_SPICEDB_PORT                  | SpiceDB host port                                                       | 50051                          |
| SMQ_SPICEDB_PRE_SHARED_KEY        | SpiceDB pre-shared key                                                  | 12345678                       |
| SMQ_SPICEDB_SCHEMA_FILE           | Path to SpiceDB schema file                                             | ./docker/spicedb/schema.zed    |
| SMQ_JAEGER_URL                    | Jaeger server URL                                                       | <http://jaeger:4318/v1/traces> |
| SMQ_JAEGER_TRACE_RATIO            | Jaeger sampling ratio                                                   | 1.0                            |
| SMQ_SEND_TELEMETRY                | Send telemetry to supermq call home server                              | true                           |
| SMQ_AUTH_ADAPTER_INSTANCE_ID      | Adapter instance ID                                                     | ""                             |
| SMQ_AUTH_CALLOUT_URLS             | Comma-separated list of callout URLs                                    | ""                             |
| SMQ_AUTH_CALLOUT_METHOD           | Callout method                                                          | POST                           |
| SMQ_AUTH_CALLOUT_TLS_VERIFICATION | Enable TLS verification for callouts                                    | true                           |
| SMQ_AUTH_CALLOUT_TIMEOUT          | Callout timeout                                                         | 10s                            |
| SMQ_AUTH_CALLOUT_CA_CERT          | Path to CA certificate file                                             | ""                             |
| SMQ_AUTH_CALLOUT_CERT             | Path to client certificate file                                         | ""                             |
| SMQ_AUTH_CALLOUT_KEY              | Path to client key file                                                 | ""                             |

## Deployment

The service itself is distributed as Docker container. Check the [`auth`](https://github.com/absmach/supermq/blob/main/docker/docker-compose.yml) service section in docker-compose file to see how service is deployed.

Running this service outside of container requires working instance of the postgres database, SpiceDB, and Jaeger server.
To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/supermq

cd supermq

# compile the service
make auth

# copy binary to bin
make install

# set the environment variables and run the service
SMQ_AUTH_LOG_LEVEL=info \
SMQ_AUTH_DB_HOST=localhost \
SMQ_AUTH_DB_PORT=5432 \
SMQ_AUTH_DB_USER=supermq \
SMQ_AUTH_DB_PASSWORD=supermq \
SMQ_AUTH_DB_NAME=auth \
SMQ_AUTH_DB_SSL_MODE=disable \
SMQ_AUTH_DB_SSL_CERT="" \
SMQ_AUTH_DB_SSL_KEY="" \
SMQ_AUTH_DB_SSL_ROOT_CERT="" \
SMQ_AUTH_HTTP_HOST=localhost \
SMQ_AUTH_HTTP_PORT=8189 \
SMQ_AUTH_HTTP_SERVER_CERT="" \
SMQ_AUTH_HTTP_SERVER_KEY="" \
SMQ_AUTH_GRPC_HOST=localhost \
SMQ_AUTH_GRPC_PORT=8181 \
SMQ_AUTH_GRPC_SERVER_CERT="" \
SMQ_AUTH_GRPC_SERVER_KEY="" \
SMQ_AUTH_GRPC_SERVER_CA_CERTS="" \
SMQ_AUTH_GRPC_CLIENT_CA_CERTS="" \
SMQ_AUTH_SECRET_KEY=secret \
SMQ_AUTH_ACCESS_TOKEN_DURATION=1h \
SMQ_AUTH_REFRESH_TOKEN_DURATION=24h \
SMQ_AUTH_INVITATION_DURATION=168h \
SMQ_SPICEDB_HOST=localhost \
SMQ_SPICEDB_PORT=50051 \
SMQ_SPICEDB_PRE_SHARED_KEY=12345678 \
SMQ_SPICEDB_SCHEMA_FILE=./docker/spicedb/schema.zed \
SMQ_JAEGER_URL=http://localhost:14268/api/traces \
SMQ_JAEGER_TRACE_RATIO=1.0 \
SMQ_SEND_TELEMETRY=true \
SMQ_AUTH_ADAPTER_INSTANCE_ID="" \
SMQ_AUTH_CALLOUT_URLS="" \
SMQ_AUTH_CALLOUT_METHOD="POST" \
SMQ_AUTH_CALLOUT_TLS_VERIFICATION=true \
$GOBIN/supermq-auth
```

Setting `SMQ_AUTH_HTTP_SERVER_CERT` and `SMQ_AUTH_HTTP_SERVER_KEY` will enable TLS against the service. The service expects a file in PEM format for both the certificate and the key.
Setting `SMQ_AUTH_GRPC_SERVER_CERT` and `SMQ_AUTH_GRPC_SERVER_KEY` will enable TLS against the service. The service expects a file in PEM format for both the certificate and the key. Setting `SMQ_AUTH_GRPC_SERVER_CA_CERTS` will enable TLS against the service trusting only those CAs that are provided. The service expects a file in PEM format of trusted CAs. Setting `SMQ_AUTH_GRPC_CLIENT_CA_CERTS` will enable TLS against the service trusting only those CAs that are provided. The service expects a file in PEM format of trusted CAs.

## Personal Access Tokens (PATs)

Personal Access Tokens (PATs) provide a secure way to authenticate with SuperMQ APIs without using your primary credentials. They are particularly useful for automation, CI/CD pipelines, and integrating with third-party services.

### Overview

PATs in SuperMQ are designed with the following features:

- **Scoped Access**: Each token can be limited to specific operations on specific resources
- **Expiration Control**: Set custom expiration times for tokens
- **Revocable**: Tokens can be revoked at any time
- **Auditable**: Track when tokens were last used
- **Secure**: Tokens are stored as hashes, not in plaintext

### Token Structure

A PAT consists of three parts separated by underscores:

```
pat_<encoded-user-and-pat-id>_<random-string>
```

Where:

- `pat` is a fixed prefix
- `<encoded-user-and-pat-id>` is a base64-encoded combination of the user ID and PAT ID
- `<random-string>` is a randomly generated string for additional security

### PAT Operations

SuperMQ supports the following operations for PATs:

| Operation   | Description                          |
| ----------- | ------------------------------------ |
| `create`    | Create a new resource                |
| `read`      | Read/view a resource                 |
| `list`      | List resources                       |
| `update`    | Update/modify a resource             |
| `delete`    | Delete a resource                    |
| `share`     | Share a resource with others         |
| `unshare`   | Remove sharing permissions           |
| `publish`   | Publish messages to a channel        |
| `subscribe` | Subscribe to messages from a channel |

### Entity Types

PATs can be scoped to the following entity types:

| Entity Type  | Description            |
| ------------ | ---------------------- |
| `groups`     | User groups            |
| `channels`   | Communication channels |
| `clients`    | Client applications    |
| `domains`    | Organizational domains |
| `users`      | User accounts          |
| `dashboards` | Dashboard interfaces   |
| `messages`   | Message content        |

### API Examples

#### Creating a PAT

```bash
curl --location 'http://localhost:9001/pats' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer <access_token>' \
--data '{
    "name": "test pat",
    "description": "testing pat",
    "duration": "24h"
}'
```

Response:

```json
{
  "id": "a2500226-95dc-4285-87e2-e693e4a0a976",
  "user_id": "user123",
  "name": "pat 1",
  "description": "for creating any client or channel",
  "secret": "pat_dXNlcjEyM19hMjUwMDIyNi05NWRjLTQyODUtODdlMi1lNjkzZTRhMGE5NzY=_randomstring...",
  "issued_at": "2025-02-27T11:20:59Z",
  "expires_at": "2025-02-28T11:20:59Z"
}
```

#### Adding Scopes to a PAT

```bash
curl --location --request PATCH 'http://localhost:9001/pats/a2500226-95dc-4285-87e2-e693e4a0a976/scope/add' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer <access_token>' \
--data '{
    "scopes": [
        {
            "optional_domain_id": "c16c980a-9d4c-4793-8fb2-c81304cf1d9f",
            "entity_type": "clients",
            "operation": "create",
            "entity_id": "*"
        },
        {
            "optional_domain_id": "c16c980a-9d4c-4793-8fb2-c81304cf1d9f",
            "entity_type": "channels",
            "operation": "create",
            "entity_id": "cfbc6936-5748-4339-a8ef-37b64b02bc96"
        },
        {
            "entity_type": "dashboards",
            "optional_domain_id": "c16c980a-9d4c-4793-8fb2-c81304cf1d9f",
            "operation": "read",
            "entity_id": "*"
        }
    ]
}'
```

#### Listing PATs

```bash
curl --location 'http://localhost:9001/pats' \
--header 'Authorization: Bearer <access_token>'
```

#### Listing Scopes for a PAT

```bash
curl --location 'http://localhost:9001/pats/a2500226-95dc-4285-87e2-e693e4a0a976/scopes' \
--header 'Authorization: Bearer <access_token>'
```

#### Revoking a PAT

```bash
curl --location --request PATCH 'http://localhost:9001/pats/a2500226-95dc-4285-87e2-e693e4a0a976/revoke' \
--header 'Authorization: Bearer <access_token>'
```

#### Resetting a PAT Secret

```bash
curl --location --request PATCH 'http://localhost:9001/pats/a2500226-95dc-4285-87e2-e693e4a0a976/reset' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer <access_token>' \
--data '{
    "duration": "720h"
}'
```

### Using PATs for Authentication

When making API requests, include the PAT in the Authorization header:

```
Authorization: Bearer pat_<encoded-user-and-pat-id>_<random-string>
```

#### Example: Creating a Client Using PAT

```bash
curl --location 'http://localhost:9006/c16c980a-9d4c-4793-8fb2-c81304cf1d9f/clients' \
--header 'accept: application/json' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer pat_etKoiXKTR6a0zdgsBHC00qJQAiaV3EKFh+Lmk+SgqXY=_u7@5fyjgti9V@#Bw^bS*SPmX3OnH=HTvKwmIbxIuyBjoI|6FASo9egjKD^u-M$b|2Dpt3CXZtv&4k+hmYYjk&C$57AV59P%-iDV0' \
--data '{
  "name": "test client",
  "tags": [
    "tag1",
    "tag2"
  ],
  "metadata":{"units":"km"},
  "status": "enabled"
}'
```

This example shows how to create a client in a specific domain (`c16c980a-9d4c-4793-8fb2-c81304cf1d9f`) using a PAT for authentication. The PAT must have the appropriate scope (e.g., `clients` entity type with `create` operation) for this domain.

### Wildcard Entity IDs

When defining scopes for PATs, you can use the wildcard character `*` for the `entity_id` field to grant permissions for all entities of a specific type. This is particularly useful for automation tasks that need to operate on multiple resources.

For example:

- `"entity_id": "*"` - Grants permission for all entities of the specified type
- `"entity_id": "specific-id"` - Grants permission only for the entity with the specified ID

Using wildcards should be done carefully, as they grant broader permissions. Always follow the principle of least privilege by granting only the permissions necessary for the intended use case.

### Scope Examples

#### Allow Creating Any Client in a Domain

```json
{
  "optional_domain_id": "domain_id",
  "entity_type": "clients",
  "operation": "create",
  "entity_id": "*"
}
```

This scope allows the PAT to create any client within the specified domain. The wildcard `*` for `entity_id` means the token can create any client, not just a specific one.

#### Allow Publishing to a Specific Channel

```json
{
  "optional_domain_id": "domain_id",
  "entity_type": "channels",
  "operation": "publish",
  "entity_id": "channel_id"
}
```

This scope restricts the PAT to only publish to a specific channel (`channel_id`) within the specified domain. No wildcard is used, so the permission is limited to just this one channel.

#### Allow Reading All Dashboards

```json
{
  "optional_domain_id": "domain_id",
  "entity_type": "dashboards",
  "operation": "read",
  "entity_id": "*"
}
```

This scope allows the PAT to read all dashboards within the specified domain. The wildcard `*` for `entity_id` means the token can read any dashboard in that domain.

### Best Practices

1. **Limit Scope**: Always use the principle of least privilege when creating PATs
2. **Set Expirations**: Use reasonable expiration times for tokens
3. **Rotate Regularly**: Reset token secrets periodically
4. **Audit Usage**: Monitor when tokens are used
5. **Revoke Unused**: Remove tokens that are no longer needed

### Implementation Details

PATs are stored in the database with the following schema:

```sql
CREATE TABLE IF NOT EXISTS pats (
    id              VARCHAR(36) PRIMARY KEY,
    name            VARCHAR(254) NOT NULL,
    user_id         VARCHAR(36),
    description     TEXT,
    secret          TEXT,
    issued_at       TIMESTAMP,
    expires_at      TIMESTAMP,
    updated_at      TIMESTAMP,
    revoked         BOOLEAN,
    revoked_at      TIMESTAMP,
    last_used_at    TIMESTAMP,
    UNIQUE          (id, name, secret)
)

CREATE TABLE IF NOT EXISTS pat_scopes (
    id                  VARCHAR(36) PRIMARY KEY,
    pat_id              VARCHAR(36) REFERENCES pats(id) ON DELETE CASCADE,
    optional_domain_id  VARCHAR(36),
    entity_type         VARCHAR(50) NOT NULL,
    operation           VARCHAR(50) NOT NULL,
    entity_id           VARCHAR(50) NOT NULL,
    UNIQUE (pat_id, optional_domain_id, entity_type, operation, entity_id)
)
```

### Authorization

When a PAT is used for authentication:

1. The system parses the token to extract the user ID and PAT ID
2. It verifies the token hasn't been revoked or expired
3. It checks if the requested operation is allowed by the token's scopes
4. If all checks pass, the operation is authorized

## Usage

For more information about service capabilities and its usage, please check out the [API documentation](https://docs.api.supermq.abstractmachines.fr/?urls.primaryName=auth.yml).

[doc]: https://docs.supermq.abstractmachines.fr
