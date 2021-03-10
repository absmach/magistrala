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

| Variable                    | Description                                                            | Default        |
|-----------------------------|------------------------------------------------------------------------|----------------|
| MF_THINGS_LOG_LEVEL         | Log level for Things (debug, info, warn, error)                        | error          |
| MF_THINGS_DB_HOST           | Database host address                                                  | localhost      |
| MF_THINGS_DB_PORT           | Database host port                                                     | 5432           |
| MF_THINGS_DB_USER           | Database user                                                          | mainflux       |
| MF_THINGS_DB_PASS           | Database password                                                      | mainflux       |
| MF_THINGS_DB                | Name of the database used by the service                               | things         |
| MF_THINGS_DB_SSL_MODE       | Database connection SSL mode (disable, require, verify-ca, verify-full)| disable        |
| MF_THINGS_DB_SSL_CERT       | Path to the PEM encoded certificate file                               |                |
| MF_THINGS_DB_SSL_KEY        | Path to the PEM encoded key file                                       |                |
| MF_THINGS_DB_SSL_ROOT_CERT  | Path to the PEM encoded root certificate file                          |                |
| MF_THINGS_CLIENT_TLS        | Flag that indicates if TLS should be turned on                         | false          |
| MF_THINGS_CA_CERTS          | Path to trusted CAs in PEM format                                      |                |
| MF_THINGS_CACHE_URL         | Cache database URL                                                     | localhost:6379 |
| MF_THINGS_CACHE_PASS        | Cache database password                                                |                |
| MF_THINGS_CACHE_DB          | Cache instance name                                                    | 0              |
| MF_THINGS_ES_URL            | Event store URL                                                        | localhost:6379 |
| MF_THINGS_ES_PASS           | Event store password                                                   |                |
| MF_THINGS_ES_DB             | Event store instance name                                              | 0              |
| MF_THINGS_HTTP_PORT         | Things service HTTP port                                               | 8182           |
| MF_THINGS_AUTH_HTTP_PORT    | Things service Auth HTTP port                                          | 8989           |
| MF_THINGS_AUTH_GRPC_PORT    | Things service Auth gRPC port                                          | 8181           |
| MF_THINGS_SERVER_CERT       | Path to server certificate in pem format                               |                |
| MF_THINGS_SERVER_KEY        | Path to server key in pem format                                       |                |
| MF_THINGS_SINGLE_USER_EMAIL | User email for single user mode (no gRPC communication with users)     |                |
| MF_THINGS_SINGLE_USER_TOKEN | User token for single user mode that should be passed in auth header   |                |
| MF_JAEGER_URL               | Jaeger server URL                                                      | localhost:6831 |
| MF_AUTH_GRPC_URL            | Auth service gRPC URL                                                  | localhost:8181 |
| MF_AUTH_GRPC_TIMEOUT        | Auth service gRPC request timeout in seconds                           | 1s             |

**Note** that if you want `things` service to have only one user locally, you should use `MF_THINGS_SINGLE_USER` env vars. By specifying these, you don't need `users` service in your deployment as it won't be used for authorization.

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
MF_THINGS_DB_HOST=[Database host address] \
MF_THINGS_DB_PORT=[Database host port] \
MF_THINGS_DB_USER=[Database user] \
MF_THINGS_DB_PASS=[Database password] \
MF_THINGS_DB=[Name of the database used by the service] \
MF_THINGS_DB_SSL_MODE=[SSL mode to connect to the database with] \
MF_THINGS_DB_SSL_CERT=[Path to the PEM encoded certificate file] \
MF_THINGS_DB_SSL_KEY=[Path to the PEM encoded key file] \
MF_THINGS_DB_SSL_ROOT_CERT=[Path to the PEM encoded root certificate file] \
MF_HTTP_ADAPTER_CA_CERTS=[Path to trusted CAs in PEM format] \
MF_THINGS_CACHE_URL=[Cache database URL] \
MF_THINGS_CACHE_PASS=[Cache database password] \
MF_THINGS_CACHE_DB=[Cache instance name] \
MF_THINGS_ES_URL=[Event store URL] \
MF_THINGS_ES_PASS=[Event store password] \
MF_THINGS_ES_DB=[Event store instance name] \
MF_THINGS_HTTP_PORT=[Things service HTTP port] \
MF_THINGS_AUTH_HTTP_PORT=[Things service Auth HTTP port] \
MF_THINGS_AUTH_GRPC_PORT=[Things service Auth gRPC port] \
MF_THINGS_SERVER_CERT=[Path to server certificate] \
MF_THINGS_SERVER_KEY=[Path to server key] \
MF_THINGS_SINGLE_USER_EMAIL=[User email for single user mode (no gRPC communication with users)] \
MF_THINGS_SINGLE_USER_TOKEN=[User token for single user mode that should be passed in auth header] \
MF_JAEGER_URL=[Jaeger server URL] \
MF_AUTH_GRPC_URL=[Auth service gRPC URL] \
MF_AUTH_GRPC_TIMEOUT=[Auth service gRPC request timeout in seconds] \
$GOBIN/mainflux-things
```

Setting `MF_THINGS_CA_CERTS` expects a file in PEM format of trusted CAs. This will enable TLS against the Users gRPC endpoint trusting only those CAs that are provided.

## Usage

For more information about service capabilities and its usage, please check out
the [API documentation](openapi.yml).

[doc]: http://mainflux.readthedocs.io
