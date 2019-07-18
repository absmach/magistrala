# Cassandra reader

Cassandra reader provides message repository implementation for Cassandra.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                           | Description                                    | Default        |
|------------------------------------|------------------------------------------------|----------------|
| MF_CASSANDRA_READER_PORT           | Service HTTP port                              | 8180           |
| MF_CASSANDRA_READER_DB_CLUSTER     | Cassandra cluster comma separated addresses    | 127.0.0.1      |
| MF_CASSANDRA_READER_DB_KEYSPACE    | Cassandra keyspace name                        | mainflux       |
| MF_CASSANDRA_READER_DB_USERNAME    | Cassandra DB username                          |                |
| MF_CASSANDRA_READER_DB_PASSWORD    | Cassandra DB password                          |                |
| MF_CASSANDRA_READER_DB_PORT        | Cassandra DB port                              | 9042           |
| MF_THINGS_URL                      | Things service URL                             | localhost:8181 |
| MF_CASSANDRA_READER_CLIENT_TLS     | Flag that indicates if TLS should be turned on | false          |
| MF_CASSANDRA_READER_CA_CERTS       | Path to trusted CAs in PEM format              |                |
| MF_JAEGER_URL                      | Jaeger server URL                              | localhost:6831 |
| MF_CASSANDRA_READER_THINGS_TIMEOUT | Things gRPC request timeout in seconds         | 1              |


## Deployment

```yaml
  version: "2"
  cassandra-reader:
    image: mainflux/cassandra-reader:[version]
    container_name: [instance name]
    expose:
      - [Service HTTP port]
    restart: on-failure
    environment:
      MF_THINGS_URL: [Things service URL]
      MF_CASSANDRA_READER_PORT: [Service HTTP port]
      MF_CASSANDRA_READER_DB_CLUSTER: [Cassandra cluster comma separated addresses]
      MF_CASSANDRA_READER_DB_KEYSPACE: [Cassandra keyspace name]
      MF_CASSANDRA_READER_DB_USERNAME: [Cassandra DB username]
      MF_CASSANDRA_READER_DB_PASSWORD: [Cassandra DB password]
      MF_CASSANDRA_READER_DB_PORT: [Cassandra DB port]
      MF_CASSANDRA_READER_CLIENT_TLS: [Flag that indicates if TLS should be turned on]
      MF_CASSANDRA_READER_CA_CERTS: [Path to trusted CAs in PEM format]
      MF_JAEGER_URL: [Jaeger server URL]
      MF_CASSANDRA_READER_THINGS_TIMEOUT: [Things gRPC request timeout in seconds]
    ports:
      - [host machine port]:[configured HTTP port]
```

To start the service, execute the following shell script:

```bash
# download the latest version of the service
go get github.com/mainflux/mainflux


cd $GOPATH/src/github.com/mainflux/mainflux

# compile the cassandra
make cassandra-reader

# copy binary to bin
make install

# Set the environment variables and run the service
MF_THINGS_URL=[Things service URL] MF_CASSANDRA_READER_PORT=[Service HTTP port] MF_CASSANDRA_READER_DB_CLUSTER=[Cassandra cluster comma separated addresses] MF_CASSANDRA_READER_DB_KEYSPACE=[Cassandra keyspace name] MF_CASSANDRA_READER_DB_USERNAME=[Cassandra DB username] MF_CASSANDRA_READER_DB_PASSWORD=[Cassandra DB password] MF_CASSANDRA_READER_DB_PORT=[Cassandra DB port] MF_CASSANDRA_READER_CLIENT_TLS=[Flag that indicates if TLS should be turned on] MF_CASSANDRA_READER_CA_CERTS=[Path to trusted CAs in PEM format] MF_JAEGER_URL=[Jaeger server URL] MF_CASSANDRA_READER_THINGS_TIMEOUT=[Things gRPC request timeout in seconds] $GOBIN/mainflux-cassandra-reader

```

### Using docker-compose

This service can be deployed using docker containers. Docker compose file is
available in `<project_root>/docker/addons/cassandra-reader/docker-compose.yml`.
In order to run all Mainflux core services, as well as mentioned optional ones,
execute following command:

```bash
docker-compose -f docker/docker-compose.yml up -d
./docker/addons/cassandra-writer/init.sh
docker-compose -f docker/addons/casandra-reader/docker-compose.yml up -d
```

## Usage

Service exposes [HTTP API][doc]  for fetching messages.

[doc]: ../swagger.yml
