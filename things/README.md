# Things

Things service provides an HTTP API for managing platform resources: things and channels.
Through this API clients are able to do the following actions:

- provision new things
- create new channels
- "connect" things into the channels

For an in-depth explanation of the aforementioned scenarios, as well as thorough
understanding of Magistrala, please check out the [official documentation][doc].

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                        | Description                                                             | Default                          |
| ------------------------------- | ----------------------------------------------------------------------- | -------------------------------- |
| MG_THINGS_LOG_LEVEL             | Log level for Things (debug, info, warn, error)                         | info                             |
| MG_THINGS_HTTP_HOST             | Things service HTTP host                                                | localhost                        |
| MG_THINGS_HTTP_PORT             | Things service HTTP port                                                | 9000                             |
| MG_THINGS_SERVER_CERT           | Path to the PEM encoded server certificate file                         | ""                               |
| MG_THINGS_SERVER_KEY            | Path to the PEM encoded server key file                                 | ""                               |
| MG_THINGS_AUTH_GRPC_HOST        | Things service gRPC host                                                | localhost                        |
| MG_THINGS_AUTH_GRPC_PORT        | Things service gRPC port                                                | 7000                             |
| MG_THINGS_AUTH_GRPC_SERVER_CERT | Path to the PEM encoded server certificate file                         | ""                               |
| MG_THINGS_AUTH_GRPC_SERVER_KEY  | Path to the PEM encoded server key file                                 | ""                               |
| MG_THINGS_DB_HOST               | Database host address                                                   | localhost                        |
| MG_THINGS_DB_PORT               | Database host port                                                      | 5432                             |
| MG_THINGS_DB_USER               | Database user                                                           | magistrala                       |
| MG_THINGS_DB_PASS               | Database password                                                       | magistrala                       |
| MG_THINGS_DB_NAME               | Name of the database used by the service                                | things                           |
| MG_THINGS_DB_SSL_MODE           | Database connection SSL mode (disable, require, verify-ca, verify-full) | disable                          |
| MG_THINGS_DB_SSL_CERT           | Path to the PEM encoded certificate file                                | ""                               |
| MG_THINGS_DB_SSL_KEY            | Path to the PEM encoded key file                                        | ""                               |
| MG_THINGS_DB_SSL_ROOT_CERT      | Path to the PEM encoded root certificate file                           | ""                               |
| MG_THINGS_CACHE_URL             | Cache database URL                                                      | <redis://localhost:6379/0>       |
| MG_THINGS_CACHE_KEY_DURATION    | Cache key duration in seconds                                           | 3600                             |
| MG_THINGS_ES_URL                | Event store URL                                                         | <localhost:6379>                 |
| MG_THINGS_ES_PASS               | Event store password                                                    | ""                               |
| MG_THINGS_ES_DB                 | Event store instance name                                               | 0                                |
| MG_THINGS_STANDALONE_ID         | User ID for standalone mode (no gRPC communication with Auth)           | ""                               |
| MG_THINGS_STANDALONE_TOKEN      | User token for standalone mode that should be passed in auth header     | ""                               |
| MG_JAEGER_URL                   | Jaeger server URL                                                       | <http://jaeger:14268/api/traces> |
| MG_AUTH_GRPC_URL                | Auth service gRPC URL                                                   | localhost:7001                   |
| MG_AUTH_GRPC_TIMEOUT            | Auth service gRPC request timeout in seconds                            | 1s                               |
| MG_AUTH_GRPC_CLIENT_TLS         | Enable TLS for gRPC client                                              | false                            |
| MG_AUTH_GRPC_CA_CERT            | Path to the CA certificate file                                         | ""                               |
| MG_SEND_TELEMETRY               | Send telemetry to magistrala call home server.                          | true                             |
| MG_THINGS_INSTANCE_ID           | Things instance ID                                                      | ""                               |

**Note** that if you want `things` service to have only one user locally, you should use `MG_THINGS_STANDALONE` env vars. By specifying these, you don't need `auth` service in your deployment for users' authorization.

## Deployment

The service itself is distributed as Docker container. Check the [`things `](https://github.com/absmach/magistrala/blob/master/docker/docker-compose.yml#L167-L194) service section in
docker-compose to see how service is deployed.

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/magistrala

cd magistrala

# compile the things
make things

# copy binary to bin
make install

# set the environment variables and run the service
MG_THINGS_LOG_LEVEL=[Things log level] \
MG_THINGS_STANDALONE_ID=[User ID for standalone mode (no gRPC communication with auth)] \
MG_THINGS_STANDALONE_TOKEN=[User token for standalone mode that should be passed in auth header] \
MG_THINGS_CACHE_KEY_DURATION=[Cache key duration in seconds] \
MG_THINGS_HTTP_HOST=[Things service HTTP host] \
MG_THINGS_HTTP_PORT=[Things service HTTP port] \
MG_THINGS_HTTP_SERVER_CERT=[Path to server certificate in pem format] \
MG_THINGS_HTTP_SERVER_KEY=[Path to server key in pem format] \
MG_THINGS_AUTH_GRPC_HOST=[Things service gRPC host] \
MG_THINGS_AUTH_GRPC_PORT=[Things service gRPC port] \
MG_THINGS_AUTH_GRPC_SERVER_CERT=[Path to server certificate in pem format] \
MG_THINGS_AUTH_GRPC_SERVER_KEY=[Path to server key in pem format] \
MG_THINGS_DB_HOST=[Database host address] \
MG_THINGS_DB_PORT=[Database host port] \
MG_THINGS_DB_USER=[Database user] \
MG_THINGS_DB_PASS=[Database password] \
MG_THINGS_DB_NAME=[Name of the database used by the service] \
MG_THINGS_DB_SSL_MODE=[SSL mode to connect to the database with] \
MG_THINGS_DB_SSL_CERT=[Path to the PEM encoded certificate file] \
MG_THINGS_DB_SSL_KEY=[Path to the PEM encoded key file] \
MG_THINGS_DB_SSL_ROOT_CERT=[Path to the PEM encoded root certificate file] \
MG_THINGS_CACHE_URL=[Cache database URL] \
MG_THINGS_ES_URL=[Event store URL] \
MG_THINGS_ES_PASS=[Event store password] \
MG_THINGS_ES_DB=[Event store instance name] \
MG_AUTH_GRPC_URL=[Auth service gRPC URL] \
MG_AUTH_GRPC_TIMEOUT=[Auth service gRPC request timeout in seconds] \
MG_AUTH_GRPC_CLIENT_TLS=[Enable TLS for gRPC client] \
MG_AUTH_GRPC_CA_CERT=[Path to trusted CA certificate file] \
MG_JAEGER_URL=[Jaeger server URL] \
MG_SEND_TELEMETRY=[Send telemetry to magistrala call home server] \
MG_THINGS_INSTANCE_ID=[Things instance ID] \
$GOBIN/magistrala-things
```

Setting `MG_THINGS_CA_CERTS` expects a file in PEM format of trusted CAs. This will enable TLS against the Auth gRPC endpoint trusting only those CAs that are provided.

In constrained environments, sometimes it makes sense to run Things service as a standalone to reduce network traffic and simplify deployment. This means that Things service
operates only using a single user and is able to authorize it without gRPC communication with Auth service.
To run service in a standalone mode, set `MG_THINGS_STANDALONE_EMAIL` and `MG_THINGS_STANDALONE_TOKEN`.

## Usage

For more information about service capabilities and its usage, please check out
the [API documentation](https://api.mainflux.io/?urls.primaryName=things-openapi.yml).

[doc]: https://docs.mainflux.io
