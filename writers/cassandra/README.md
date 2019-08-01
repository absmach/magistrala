# Cassandra writer

Cassandra writer provides message repository implementation for Cassandra.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                            | Description                                                | Default               |
|-------------------------------------|------------------------------------------------------------|-----------------------|
| MF_NATS_URL                         | NATS instance URL                                          | nats://localhost:4222 |
| MF_CASSANDRA_WRITER_LOG_LEVEL       | Log level for Cassandra writer (debug, info, warn, error)  | error                 |
| MF_CASSANDRA_WRITER_PORT            | Service HTTP port                                          | 8180                  |
| MF_CASSANDRA_WRITER_DB_CLUSTER      | Cassandra cluster comma separated addresses                | 127.0.0.1             |
| MF_CASSANDRA_WRITER_DB_KEYSPACE     | Cassandra keyspace name                                    | mainflux              |
| MF_CASSANDRA_WRITER_DB_USERNAME     | Cassandra DB username                                      |                       |
| MF_CASSANDRA_WRITER_DB_PASSWORD     | Cassandra DB password                                      |                       |
| MF_CASSANDRA_WRITER_DB_PORT         | Cassandra DB port                                          | 9042                  |
| MF_CASSANDRA_WRITER_CHANNELS_CONFIG | Configuration file path with channels list                 | /config/channels.toml |
## Deployment

```yaml
  version: "2"
  cassandra-writer:
    image: mainflux/cassandra-writer:[version]
    container_name: [instance name]
    expose:
      - [Service HTTP port]
    restart: on-failure
    environment:
      MF_NATS_URL: [NATS instance URL]
      MF_CASSANDRA_WRITER_LOG_LEVEL: [Cassandra writer log level]
      MF_CASSANDRA_WRITER_PORT: [Service HTTP port]
      MF_CASSANDRA_WRITER_DB_CLUSTER: [Cassandra cluster comma separated addresses]
      MF_CASSANDRA_WRITER_DB_KEYSPACE: [Cassandra keyspace name]
      MF_CASSANDRA_WRITER_DB_USERNAME: [Cassandra DB username]
      MF_CASSANDRA_WRITER_DB_PASSWORD: [Cassandra DB password]
      MF_CASSANDRA_WRITER_DB_PORT: [Cassandra DB port]
      MF_CASSANDRA_WRITER_CHANNELS_CONFIG: [Configuration file path with channels list]
    ports:
      - [host machine port]:[configured HTTP port]
    volume:
      - ./channels.yaml:/config/channels.yaml
```

To start the service, execute the following shell script:

```bash
# download the latest version of the service
go get github.com/mainflux/mainflux


cd $GOPATH/src/github.com/mainflux/mainflux

# compile the cassandra writer
make cassandra-writer

# copy binary to bin
make install

# Set the environment variables and run the service
MF_NATS_URL=[NATS instance URL] MF_CASSANDRA_WRITER_LOG_LEVEL=[Cassandra writer log level] MF_CASSANDRA_WRITER_PORT=[Service HTTP port] MF_CASSANDRA_WRITER_DB_CLUSTER=[Cassandra cluster comma separated addresses] MF_CASSANDRA_WRITER_DB_KEYSPACE=[Cassandra keyspace name] MF_CASSANDRA_READER_DB_USERNAME=[Cassandra DB username] MF_CASSANDRA_READER_DB_PASSWORD=[Cassandra DB password] MF_CASSANDRA_READER_DB_PORT=[Cassandra DB port] MF_CASSANDRA_WRITER_CHANNELS_CONFIG=[Configuration file path with channels list] $GOBIN/mainflux-cassandra-writer

```

### Using docker-compose

This service can be deployed using docker containers. Docker compose file is
available in `<project_root>/docker/addons/cassandra-writer/docker-compose.yml`.
In order to run all Mainflux core services, as well as mentioned optional ones,
execute following command:

```bash
./docker/addons/cassandra-writer/init.sh
```

## Usage

Starting service will start consuming normalized messages in SenML format.

[doc]: http://mainflux.readthedocs.io
