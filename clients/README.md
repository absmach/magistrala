# Clients

Clients service provides an HTTP API for managing platform resources: devices,
applications and channels. Through this API clients are able to do the following
actions:

- provision new clients (i.e. devices & applications)
- create new channels
- "connect" clients into the channels

For in-depth explanation of the aforementioned scenarios, as well as thorough
understanding of Mainflux, please check out the [official documentation][doc].

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable               | Description                              | Default        |
|------------------------|------------------------------------------|----------------|
| MF_CLIENTS_DB_HOST     | Database host address                    | localhost      |
| MF_CLIENTS_DB_PORT     | Database host port                       | 5432           |
| MF_CLIENTS_DB_USER     | Database user                            | mainflux       |
| MF_CLIENTS_DB_PASSWORD | Database password                        | mainflux       |
| MF_CLIENTS_DB          | Name of the database used by the service | clients        |
| MF_CLIENTS_HTTP_PORT   | Clients service HTTP port                | 8180           |
| MF_CLIENTS_GRPC_PORT   | Clients service gRPC port                | 8181           |
| MF_USERS_URL           | Users service URL                        | localhost:8181 |
| MF_CLIENTS_SECRET      | String used for signing tokens           | clients        |

## Deployment

The service itself is distributed as Docker container. The following snippet
provides a compose file template that can be used to deploy the service container
locally:

```yaml
version: "2"
services:
  clients:
    image: mainflux/clients:[version]
    container_name: [instance name]
    ports:
      - [host machine port]:[configured HTTP port]
    environment:
      MF_CLIENTS_DB_HOST: [Database host address]
      MF_CLIENTS_DB_PORT: [Database host port]
      MF_CLIENTS_DB_USER: [Database user]
      MF_CLIENTS_DB_PASS: [Database password]
      MF_CLIENTS_DB: [Name of the database used by the service]
      MF_CLIENTS_HTTP_PORT: [Service HTTP port]
      MF_CLIENTS_GRPC_PORT: [Service gRPC port]
      MF_USERS_URL: [Users service URL]
      MF_CLIENTS_SECRET: [String used for signing tokens]
```

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
go get github.com/mainflux/mainflux

cd $GOPATH/src/github.com/mainflux/mainflux

# compile the clients
make clients

# copy binary to bin
make install

# set the environment variables and run the service
MF_CLIENTS_DB_HOST=[Database host address] MF_CLIENTS_DB_PORT=[Database host port] MF_CLIENTS_DB_USER=[Database user] MF_CLIENTS_DB_PASS=[Database password] MF_CLIENTS_DB=[Name of the database used by the service] MF_CLIENTS_HTTP_PORT=[Service HTTP port] MF_CLIENTS_GRPC_PORT=[Service gRPC port] MF_USERS_URL=[Users service URL] MF_CLIENTS_SECRET=[String used for signing tokens] $GOBIN/mainflux-clients
```

## Usage

For more information about service capabilities and its usage, please check out
the [API documentation](swagger.yaml).

[doc]: http://mainflux.readthedocs.io
