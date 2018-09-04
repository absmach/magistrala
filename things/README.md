# Things

Things service provides an HTTP API for managing platform resources: devices,
applications and channels. Through this API clients are able to do the following
actions:

- provision new things (i.e. devices & applications)
- create new channels
- "connect" things into the channels

For an in-depth explanation of the aforementioned scenarios, as well as thorough
understanding of Mainflux, please check out the [official documentation][doc].

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable              | Description                              | Default        |
|-----------------------|------------------------------------------|----------------|
| MF_THINGS_DB_HOST     | Database host address                    | localhost      |
| MF_THINGS_DB_PORT     | Database host port                       | 5432           |
| MF_THINGS_DB_USER     | Database user                            | mainflux       |
| MF_THINGS_DB_PASS     | Database password                        | mainflux       |
| MF_THINGS_DB          | Name of the database used by the service | things         |
| MF_THINGS_CACHE_URL   | Cache database URL                       | localhost:6379 |
| MF_THINGS_CACHE_PASS  | Cache database password                  |                |
| MF_THINGS_CACHE_DB    | Cache instance that should be used       | 0              |
| MF_THINGS_HTTP_PORT   | Things service HTTP port                 | 8180           |
| MF_THINGS_GRPC_PORT   | Things service gRPC port                 | 8181           |
| MF_USERS_URL          | Users service URL                        | localhost:8181 |

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
      MF_THINGS_DB_HOST: [Database host address]
      MF_THINGS_DB_PORT: [Database host port]
      MF_THINGS_DB_USER: [Database user]
      MF_THINGS_DB_PASS: [Database password]
      MF_THINGS_DB: [Name of the database used by the service]
      MF_THINGS_CACHE_URL: [Cache database URL]
      MF_THINGS_CACHE_PASS: [Cache database password]
      MF_THINGS_CACHE_DB: [Cache instance that should be used]
      MF_THINGS_HTTP_PORT: [Service HTTP port]
      MF_THINGS_GRPC_PORT: [Service gRPC port]
      MF_USERS_URL: [Users service URL]
      MF_THINGS_SECRET: [String used for signing tokens]
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
MF_THINGS_DB_HOST=[Database host address] MF_THINGS_DB_PORT=[Database host port] MF_THINGS_DB_USER=[Database user] MF_THINGS_DB_PASS=[Database password] MF_THINGS_DB=[Name of the database used by the service] MF_THINGS_CACHE_URL=[Cache database URL] MF_THINGS_CACHE_PASS=[Cache database password] MF_THINGS_CACHE_DB=[Cache instance that should be used] MF_THINGS_HTTP_PORT=[Service HTTP port] MF_THINGS_GRPC_PORT=[Service gRPC port] MF_USERS_URL=[Users service URL] $GOBIN/mainflux-things
```

## Usage

For more information about service capabilities and its usage, please check out
the [API documentation](swagger.yaml).

[doc]: http://mainflux.readthedocs.io
