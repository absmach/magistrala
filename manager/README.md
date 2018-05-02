# Manager

Manager provides an HTTP API for managing platform resources: users, devices,
applications and channels. Through this API clients are able to do the following
actions:

- register new accounts and obtain access tokens
- provision new clients (i.e. devices & applications)
- create new channels
- "connect" clients into the channels

For in-depth explanation of the aforementioned scenarios, as well as thorough
understanding of Mainflux, please check out the [official documentation][doc].

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable          | Description                              | Default   |
|-------------------|------------------------------------------|-----------|
| MF_DB_HOST        | Database host address                    | localhost |
| MF_DB_PORT        | Database host port                       | 5432      |
| MF_DB_USER        | Database user                            | mainflux  |
| MF_DB_PASSWORD    | Database password                        | mainflux  |
| MF_MANAGER_DB     | Name of the database used by the service | manager   |
| MF_MANAGER_PORT   | Manager service HTTP port                | 8180      |
| MF_MANAGER_SECRET | string used for signing tokens           | manager   |

## Deployment

The service itself is distributed as Docker container. The following snippet
provides a compose file template that can be used to deploy the service container
locally:

```yaml
version: "2"
services:
  manager:
    image: mainflux/manager:[version]
    container_name: [instance name]
    ports:
      - [host machine port]:[configured HTTP port]
    environment:
      MF_DB_HOST: [Database host address]
      MF_DB_PORT: [Database host port]
      MF_DB_USER: [Database user]
      MF_DB_PASS: [Database password]
      MF_MANAGER_DB: [Name of the database used by the service]
      MF_MANAGER_PORT: [Service HTTP port]
      MF_MANAGER_SECRET: [String used for signing tokens]
```

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
go get github.com/mainflux/mainflux

cd $GOPATH/src/github.com/mainflux/mainflux

# compile the manager
make manager

# copy binary to bin
make install

# set the environment variables and run the service
MF_DB_HOST=[Database host address] MF_DB_PORT=[Database host port] MF_DB_USER=[Database user] MF_DB_PASS=[Database password] MF_MANAGER_DB=[Name of the database used by the service] MF_MANAGER_PORT=[Service HTTP port] MF_MANAGER_SECRET=[String used for signing tokens] $GOBIN/mainflux-manager
```

## Usage

For more information about service capabilities and its usage, please check out
the [API documentation](swagger.yaml).

[doc]: http://mainflux.readthedocs.io
