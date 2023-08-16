# Things

Things service provides an HTTP API for managing platform resources: things and channels.
Through this API clients are able to do the following actions:

- provision new things
- create new channels
- "connect" things into the channels

For an in-depth explanation of the aforementioned scenarios, as well as thorough
understanding of Mainflux, please check out the [official documentation][doc].

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                          | Description                                                             | Default                        |
| --------------------------------- | ----------------------------------------------------------------------- | ------------------------------ |
| MF_THINGS_LOG_LEVEL               | Log level for Things (debug, info, warn, error)                         | info                           |
| MF_THINGS_HTTP_HOST               | Things service HTTP host                                                | localhost                      |
| MF_THINGS_HTTP_PORT               | Things service HTTP port                                                | 9000                           |
| MF_THINGS_SERVER_CERT             | Path to the PEM encoded server certificate file                         | ""                             |
| MF_THINGS_SERVER_KEY              | Path to the PEM encoded server key file                                 | ""                             |
| MF_THINGS_AUTH_GRPC_HOST          | Things service gRPC host                                                | localhost                      |
| MF_THINGS_AUTH_GRPC_PORT          | Things service gRPC port                                                | 7000                           |
| MF_THINGS_AUTH_GRPC_SERVER_CERT   | Path to the PEM encoded server certificate file                         | ""                             |
| MF_THINGS_AUTH_GRPC_SERVER_KEY    | Path to the PEM encoded server key file                                 | ""                             |
| MF_THINGS_DB_HOST                 | Database host address                                                   | localhost                      |
| MF_THINGS_DB_PORT                 | Database host port                                                      | 5432                           |
| MF_THINGS_DB_USER                 | Database user                                                           | mainflux                       |
| MF_THINGS_DB_PASS                 | Database password                                                       | mainflux                       |
| MF_THINGS_DB_NAME                 | Name of the database used by the service                                | things                         |
| MF_THINGS_DB_SSL_MODE             | Database connection SSL mode (disable, require, verify-ca, verify-full) | disable                        |
| MF_THINGS_DB_SSL_CERT             | Path to the PEM encoded certificate file                                | ""                             |
| MF_THINGS_DB_SSL_KEY              | Path to the PEM encoded key file                                        | ""                             |
| MF_THINGS_DB_SSL_ROOT_CERT        | Path to the PEM encoded root certificate file                           | ""                             |
| MF_THINGS_CACHE_URL               | Cache database URL                                                      | localhost:6379                 |
| MF_THINGS_CACHE_PASS              | Cache database password                                                 | ""                             |
| MF_THINGS_CACHE_DB                | Cache instance name                                                     | 0                              |
| MF_THINGS_CACHE_KEY_DURATION      | Cache key duration in seconds                                           | 3600                           |
| MF_THINGS_ES_URL                  | Event store URL                                                         | localhost:6379                 |
| MF_THINGS_ES_PASS                 | Event store password                                                    | ""                             |
| MF_THINGS_ES_DB                   | Event store instance name                                               | 0                              |
| MF_THINGS_STANDALONE_ID           | User ID for standalone mode (no gRPC communication with users)          | ""                             |
| MF_THINGS_STANDALONE_TOKEN        | User token for standalone mode that should be passed in auth header     | ""                             |
| MF_JAEGER_URL                     | Jaeger server URL                                                       | http://jaeger:14268/api/traces |
| MF_AUTH_GRPC_URL                  | Users service gRPC URL                                                  | localhost:7001                 |
| MF_AUTH_GRPC_TIMEOUT              | Users service gRPC request timeout in seconds                           | 1s                             |
| MF_AUTH_GRPC_CLIENT_TLS           | Enable TLS for gRPC client                                              | false                          |
| MF_AUTH_GRPC_CA_CERT              | Path to the CA certificate file                                         | ""                             |
| MF_SEND_TELEMETRY                 | Send telemetry to mainflux call home server.                            | true                           |
| MF_THINGS_INSTANCE_ID             | Things instance ID                                                      | ""                             |

**Note** that if you want `thin     gs` service to have only one user locally, you should use `MF_THINGS_STANDALONE` env vars. By specifying these, you don't need `auth` service in your deployment for users' authorization.

## Deployment

The service itself is distributed as Docker container. Check the [`things `](https://github.com/mainflux/mainflux/blob/master/docker/docker-compose.yml#L167-L194) service section in
docker-compose to see how service is deployed.

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/mainflux/mainflux

cd mainflux

# compile the things
make things

# copy binary to bin
make install

# set the environment variables and run the service
MF_THINGS_LOG_LEVEL=[Things log level] \
MF_THINGS_STANDALONE_ID=[User ID for standalone mode (no gRPC communication with auth)] \
MF_THINGS_STANDALONE_TOKEN=[User token for standalone mode that should be passed in auth header] \
MF_THINGS_CACHE_KEY_DURATION=[Cache key duration in seconds] \
MF_THINGS_HTTP_HOST=[Things service HTTP host] \
MF_THINGS_HTTP_PORT=[Things service HTTP port] \
MF_THINGS_HTTP_SERVER_CERT=[Path to server certificate in pem format] \
MF_THINGS_HTTP_SERVER_KEY=[Path to server key in pem format] \
MF_THINGS_AUTH_GRPC_HOST=[Things service gRPC host] \
MF_THINGS_AUTH_GRPC_PORT=[Things service gRPC port] \
MF_THINGS_AUTH_GRPC_SERVER_CERT=[Path to server certificate in pem format] \
MF_THINGS_AUTH_GRPC_SERVER_KEY=[Path to server key in pem format] \
MF_THINGS_DB_HOST=[Database host address] \
MF_THINGS_DB_PORT=[Database host port] \
MF_THINGS_DB_USER=[Database user] \
MF_THINGS_DB_PASS=[Database password] \
MF_THINGS_DB_NAME=[Name of the database used by the service] \
MF_THINGS_DB_SSL_MODE=[SSL mode to connect to the database with] \
MF_THINGS_DB_SSL_CERT=[Path to the PEM encoded certificate file] \
MF_THINGS_DB_SSL_KEY=[Path to the PEM encoded key file] \
MF_THINGS_DB_SSL_ROOT_CERT=[Path to the PEM encoded root certificate file] \
MF_THINGS_CACHE_URL=[Cache database URL] \
MF_THINGS_CACHE_PASS=[Cache database password] \
MF_THINGS_CACHE_DB=[Cache instance name] \
MF_THINGS_ES_URL=[Event store URL] \
MF_THINGS_ES_PASS=[Event store password] \
MF_THINGS_ES_DB=[Event store instance name] \
MF_AUTH_GRPC_URL=[Users service gRPC URL] \
MF_AUTH_GRPC_TIMEOUT=[Users service gRPC request timeout in seconds] \
MF_AUTH_GRPC_CLIENT_TLS=[Enable TLS for gRPC client] \
MF_AUTH_GRPC_CA_CERT=[Path to trusted CA certificate file] \
MF_JAEGER_URL=[Jaeger server URL] \
MF_SEND_TELEMETRY=[Send telemetry to mainflux call home server] \
MF_THINGS_INSTANCE_ID=[Things instance ID] \
$GOBIN/mainflux-things
```

Setting `MF_THINGS_CA_CERTS` expects a file in PEM format of trusted CAs. This will enable TLS against the Users gRPC endpoint trusting only those CAs that are provided.

In constrained environments, sometimes it makes sense to run Things service as a standalone to reduce network traffic and simplify deployment. This means that Things service
operates only using a single user and is able to authorize it without gRPC communication with Auth service.
To run service in a standalone mode, set `MF_THINGS_STANDALONE_EMAIL` and `MF_THINGS_STANDALONE_TOKEN`.

## Usage

For more information about service capabilities and its usage, please check out
the [API documentation](https://api.mainflux.io/?urls.primaryName=things-openapi.yml).

[doc]: https://docs.mainflux.io
