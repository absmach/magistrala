# Auth - Authentication and Authorization service

Auth service provides authentication features as an API for managing authentication keys as well as administering groups of entities - `things` and `users`.

## Authentication

User service is using Auth service gRPC API to obtain login token or password reset token. Authentication key consists of the following fields:

- ID - key ID
- Type - one of the three types described below
- IssuerID - an ID of the Magistrala User who issued the key
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

API keys are similar to the User keys. The main difference is that API keys have configurable expiration time. If no time is set, the key will never expire. For that reason, API keys are _the only key type that can be revoked_. This also means that, despite being used as a JWT, it requires a query to the database to validate the API key. The user with API key can perform all the same actions as the user with login key (can act on behalf of the user for Thing, Channel, or user profile management), _except issuing new API keys_.

Recovery key is the password recovery key. It's short-lived token used for password recovery process.

For in-depth explanation of the aforementioned scenarios, as well as thorough understanding of Magistrala, please check out the [official documentation][doc].

The following actions are supported:

- create (all key types)
- verify (all key types)
- obtain (API keys only)
- revoke (API keys only)

## Domains

Domains are used to group users and things. Each domain has a unique alias that is used to identify the domain. Domains are used to group users and their entities.

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

| Variable                       | Description                                                             | Default                          |
| ------------------------------ | ----------------------------------------------------------------------- | -------------------------------- |
| MG_AUTH_LOG_LEVEL              | Log level for the Auth service (debug, info, warn, error)               | info                             |
| MG_AUTH_DB_HOST                | Database host address                                                   | localhost                        |
| MG_AUTH_DB_PORT                | Database host port                                                      | 5432                             |
| MG_AUTH_DB_USER                | Database user                                                           | magistrala                       |
| MG_AUTH_DB_PASSWORD            | Database password                                                       | magistrala                       |
| MG_AUTH_DB_NAME                | Name of the database used by the service                                | auth                             |
| MG_AUTH_DB_SSL_MODE            | Database connection SSL mode (disable, require, verify-ca, verify-full) | disable                          |
| MG_AUTH_DB_SSL_CERT            | Path to the PEM encoded certificate file                                | ""                               |
| MG_AUTH_DB_SSL_KEY             | Path to the PEM encoded key file                                        | ""                               |
| MG_AUTH_DB_SSL_ROOT_CERT       | Path to the PEM encoded root certificate file                           | ""                               |
| MG_AUTH_HTTP_HOST              | Auth service HTTP host                                                  | ""                               |
| MG_AUTH_HTTP_PORT              | Auth service HTTP port                                                  | 8189                             |
| MG_AUTH_HTTP_SERVER_CERT       | Path to the PEM encoded HTTP server certificate file                    | ""                               |
| MG_AUTH_HTTP_SERVER_KEY        | Path to the PEM encoded HTTP server key file                            | ""                               |
| MG_AUTH_GRPC_HOST              | Auth service gRPC host                                                  | ""                               |
| MG_AUTH_GRPC_PORT              | Auth service gRPC port                                                  | 8181                             |
| MG_AUTH_GRPC_SERVER_CERT       | Path to the PEM encoded gRPC server certificate file                    | ""                               |
| MG_AUTH_GRPC_SERVER_KEY        | Path to the PEM encoded gRPC server key file                            | ""                               |
| MG_AUTH_GRPC_SERVER_CA_CERTS   | Path to the PEM encoded gRPC server CA certificate file                 | ""                               |
| MG_AUTH_GRPC_CLIENT_CA_CERTS   | Path to the PEM encoded gRPC client CA certificate file                 | ""                               |
| MG_AUTH_SECRET_KEY             | String used for signing tokens                                          | secret                           |
| MG_AUTH_ACCESS_TOKEN_DURATION  | The access token expiration period                                      | 1h                               |
| MG_AUTH_REFRESH_TOKEN_DURATION | The refresh token expiration period                                     | 24h                              |
| MG_AUTH_INVITATION_DURATION    | The invitation token expiration period                                  | 168h                             |
| MG_SPICEDB_HOST                | SpiceDB host address                                                    | localhost                        |
| MG_SPICEDB_PORT                | SpiceDB host port                                                       | 50051                            |
| MG_SPICEDB_PRE_SHARED_KEY      | SpiceDB pre-shared key                                                  | 12345678                         |
| MG_SPICEDB_SCHEMA_FILE         | Path to SpiceDB schema file                                             | ./docker/spicedb/schema.zed      |
| MG_JAEGER_URL                  | Jaeger server URL                                                       | <http://jaeger:14268/api/traces> |
| MG_JAEGER_TRACE_RATIO          | Jaeger sampling ratio                                                   | 1.0                              |
| MG_SEND_TELEMETRY              | Send telemetry to magistrala call home server                           | true                             |
| MG_AUTH_ADAPTER_INSTANCE_ID    | Adapter instance ID                                                     | ""                               |

## Deployment

The service itself is distributed as Docker container. Check the [`auth`](https://github.com/absmach/magistrala/blob/master/docker/docker-compose.yml) service section in docker-compose to see how service is deployed.

Running this service outside of container requires working instance of the postgres database, SpiceDB, and Jaeger server.
To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/magistrala

cd magistrala

# compile the service
make auth

# copy binary to bin
make install

# set the environment variables and run the service
MG_AUTH_LOG_LEVEL=info \
MG_AUTH_DB_HOST=localhost \
MG_AUTH_DB_PORT=5432 \
MG_AUTH_DB_USER=magistrala \
MG_AUTH_DB_PASSWORD=magistrala \
MG_AUTH_DB_NAME=auth \
MG_AUTH_DB_SSL_MODE=disable \
MG_AUTH_DB_SSL_CERT="" \
MG_AUTH_DB_SSL_KEY="" \
MG_AUTH_DB_SSL_ROOT_CERT="" \
MG_AUTH_HTTP_HOST=localhost \
MG_AUTH_HTTP_PORT=8189 \
MG_AUTH_HTTP_SERVER_CERT="" \
MG_AUTH_HTTP_SERVER_KEY="" \
MG_AUTH_GRPC_HOST=localhost \
MG_AUTH_GRPC_PORT=8181 \
MG_AUTH_GRPC_SERVER_CERT="" \
MG_AUTH_GRPC_SERVER_KEY="" \
MG_AUTH_GRPC_SERVER_CA_CERTS="" \
MG_AUTH_GRPC_CLIENT_CA_CERTS="" \
MG_AUTH_SECRET_KEY=secret \
MG_AUTH_ACCESS_TOKEN_DURATION=1h \
MG_AUTH_REFRESH_TOKEN_DURATION=24h \
MG_AUTH_INVITATION_DURATION=168h \
MG_SPICEDB_HOST=localhost \
MG_SPICEDB_PORT=50051 \
MG_SPICEDB_PRE_SHARED_KEY=12345678 \
MG_SPICEDB_SCHEMA_FILE=./docker/spicedb/schema.zed \
MG_JAEGER_URL=http://localhost:14268/api/traces \
MG_JAEGER_TRACE_RATIO=1.0 \
MG_SEND_TELEMETRY=true \
MG_AUTH_ADAPTER_INSTANCE_ID="" \
$GOBIN/magistrala-auth
```

Setting `MG_AUTH_HTTP_SERVER_CERT` and `MG_AUTH_HTTP_SERVER_KEY` will enable TLS against the service. The service expects a file in PEM format for both the certificate and the key.
Setting `MG_AUTH_GRPC_SERVER_CERT` and `MG_AUTH_GRPC_SERVER_KEY` will enable TLS against the service. The service expects a file in PEM format for both the certificate and the key. Setting `MG_AUTH_GRPC_SERVER_CA_CERTS` will enable TLS against the service trusting only those CAs that are provided. The service expects a file in PEM format of trusted CAs. Setting `MG_AUTH_GRPC_CLIENT_CA_CERTS` will enable TLS against the service trusting only those CAs that are provided. The service expects a file in PEM format of trusted CAs.

## Usage

For more information about service capabilities and its usage, please check out the [API documentation](https://api.mainflux.io/?urls.primaryName=auth.yml).

[doc]: https://docs.mainflux.io
