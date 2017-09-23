# Mainflux manager

Mainflux manager provides an HTTP API for managing platform resources: users,
devices, applications and channels. Through this API clients are able to do
the following actions:

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

| Variable            | Description                              | Default   |
|---------------------|------------------------------------------|-----------|
| MANAGER_DB_CLUSTER  | comma-separated Cassandra contact points | 127.0.0.1 |
| MANAGER_DB_KEYSPACE | name of the Cassandra keyspace           | manager   |
| MANAGER_SECRET      | string used for signing tokens           | manager   |

## Deployment

Before proceeding to deployment, make sure to check out the [Apache Cassandra 3.0.x
documentation][www:cassandra]. Developers are advised to get acquainted with
basic architectural concepts, data modeling techniques and deployment strategies.

> Prior to deploying the service, make sure to set up the database and create
the keyspace that will be used by the service.

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
      - [host machine port]:8180
    environment:
      MANAGER_DB_CLUSTER: [comma-separated Cassandra endpoints]
      MANAGER_DB_KEYSPACE: [name of Cassandra keyspace]
      MANAGER_SECRET: [string used for signing tokens]
```

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
go get github.com/mainflux/manager

cd $GOPATH/github.com/mainflux/manager/cmd

# compile the app; make sure to set the proper GOOS value
CGO_ENABLED=0 GOOS=[platform identifier] go build -ldflags "-s" -a -installsuffix cgo -o app

# set the environment variables and run the service
MANAGER_DB_CLUSTER=[comma-separated Cassandra endpoints] MANAGER_DB_KEYSPACE=[name of Cassandra keyspace] MANAGER_SECRET=[string used for signing tokens] app
```

## Usage

For more information about service capabilities and its usage, please check out
the [API documentation](swagger.yaml).

[doc]: http://mainflux.readthedocs.io
[www:cassandra]: http://docs.datastax.com
