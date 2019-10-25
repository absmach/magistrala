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
| MF_THINGS_CACHE_DB          | Cache instance that should be used                                     | 0              |
| MF_THINGS_ES_URL            | Event store URL                                                        | localhost:6379 |
| MF_THINGS_ES_PASS           | Event store password                                                   |                |
| MF_THINGS_ES_DB             | Event store instance that should be used                               | 0              |
| MF_THINGS_HTTP_PORT         | Things service HTTP port                                               | 8180           |
| MF_THINGS_AUTH_HTTP_PORT    | Things service auth HTTP port                                          | 8989           |
| MF_THINGS_AUTH_GRPC_PORT    | Things service auth gRPC port                                          | 8181           |
| MF_THINGS_SERVER_CERT       | Path to server certificate in pem format                               | 8181           |
| MF_THINGS_SERVER_KEY        | Path to server key in pem format                                       | 8181           |
| MF_USERS_URL                | Users service URL                                                      | localhost:8181 |
| MF_THINGS_SINGLE_USER_EMAIL | User email for single user mode (no gRPC communication with users)     |                |
| MF_THINGS_SINGLE_USER_TOKEN | User token for single user mode that should be passed in auth header   |                |
| MF_JAEGER_URL               | Jaeger server URL                                                      | localhost:6831 |
| MF_THINGS_USERS_TIMEOUT     | Users gRPC request timeout in seconds                                  | 1              |

**Note** that if you want `things` service to have only one user locally, you should use `MF_THINGS_SINGLE_USER` env vars. By specifying these, you don't need `users` service in your deployment as it won't be used for authorization.

## Deployment

The service itself is distributed as Docker container. The following snippet
provides a compose file template that can be used to deploy the service container
locally:

```yaml
version: "2"
services:
  things:
    image: mainflux/things:[version]
    container_name: [instance name]
    ports:
      - [host machine port]:[configured HTTP port]
    environment:
      MF_THINGS_LOG_LEVEL: [Things log level]
      MF_THINGS_DB_HOST: [Database host address]
      MF_THINGS_DB_PORT: [Database host port]
      MF_THINGS_DB_USER: [Database user]
      MF_THINGS_DB_PASS: [Database password]
      MF_THINGS_DB: [Name of the database used by the service]
      MF_THINGS_DB_SSL_MODE: [SSL mode to connect to the database with]
      MF_THINGS_DB_SSL_CERT: [Path to the PEM encoded certificate file]
      MF_THINGS_DB_SSL_KEY: [Path to the PEM encoded key file]
      MF_THINGS_DB_SSL_ROOT_CERT: [Path to the PEM encoded root certificate file]
      MF_THINGS_CA_CERTS: [Path to trusted CAs in PEM format]
      MF_THINGS_CACHE_URL: [Cache database URL]
      MF_THINGS_CACHE_PASS: [Cache database password]
      MF_THINGS_CACHE_DB: [Cache instance that should be used]
      MF_THINGS_ES_URL: [Event store URL]
      MF_THINGS_ES_PASS: [Event store password]
      MF_THINGS_ES_DB: [Event store instance that should be used]
      MF_THINGS_HTTP_PORT: [Service HTTP port]
      MF_THINGS_AUTH_HTTP_PORT: [Service auth HTTP port]
      MF_THINGS_AUTH_GRPC_PORT: [Service auth gRPC port]
      MF_THINGS_SERVER_CERT: [String path to server cert in pem format]
      MF_THINGS_SERVER_KEY: [String path to server key in pem format]
      MF_USERS_URL: [Users service URL]
      MF_THINGS_SECRET: [String used for signing tokens]
      MF_THINGS_SINGLE_USER_EMAIL: [User email for single user mode (no gRPC communication with users)]
      MF_THINGS_SINGLE_USER_TOKEN: [User token for single user mode that should be passed in auth header]
      MF_JAEGER_URL: [Jaeger server URL]
      MF_THINGS_USERS_TIMEOUT: [Users gRPC request timeout in seconds]
```

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
go get github.com/mainflux/mainflux

cd $GOPATH/src/github.com/mainflux/mainflux

# compile the things
make things

# copy binary to bin
make install

# set the environment variables and run the service
MF_THINGS_LOG_LEVEL=[Things log level] MF_THINGS_DB_HOST=[Database host address] MF_THINGS_DB_PORT=[Database host port] MF_THINGS_DB_USER=[Database user] MF_THINGS_DB_PASS=[Database password] MF_THINGS_DB=[Name of the database used by the service] MF_THINGS_DB_SSL_MODE=[SSL mode to connect to the database with] MF_THINGS_DB_SSL_CERT=[Path to the PEM encoded certificate file] MF_THINGS_DB_SSL_KEY=[Path to the PEM encoded key file] MF_THINGS_DB_SSL_ROOT_CERT=[Path to the PEM encoded root certificate file] MF_HTTP_ADAPTER_CA_CERTS=[Path to trusted CAs in PEM format] MF_THINGS_CACHE_URL=[Cache database URL] MF_THINGS_CACHE_PASS=[Cache database password] MF_THINGS_CACHE_DB=[Cache instance that should be used] MF_THINGS_ES_URL=[Event store URL] MF_THINGS_ES_PASS=[Event store password] MF_THINGS_ES_DB=[Event store instance that should be used] MF_THINGS_HTTP_PORT=[Service HTTP port] MF_THINGS_AUTH_HTTP_PORT=[Service auth HTTP port] MF_THINGS_AUTH_GRPC_PORT=[Service auth gRPC port] MF_USERS_URL=[Users service URL] MF_THINGS_SERVER_CERT=[Path to server certificate] MF_THINGS_SERVER_KEY=[Path to server key] MF_THINGS_SINGLE_USER_EMAIL=[User email for single user mode (no gRPC communication with users)] MF_THINGS_SINGLE_USER_TOKEN=[User token for single user mode that should be passed in auth header] MF_JAEGER_URL=[Jaeger server URL] MF_THINGS_USERS_TIMEOUT=[Users gRPC request timeout in seconds] $GOBIN/mainflux-things
```

Setting `MF_THINGS_CA_CERTS` expects a file in PEM format of trusted CAs. This will enable TLS against the Users gRPC endpoint trusting only those CAs that are provided.

## Usage

For more information about service capabilities and its usage, please check out
the [API documentation](swagger.yaml).

[doc]: http://mainflux.readthedocs.io
