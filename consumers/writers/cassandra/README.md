# Cassandra writer

Cassandra writer provides message repository implementation for Cassandra.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                         | Description                                                             | Default               |
| -------------------------------- | ----------------------------------------------------------------------- | --------------------- |
| MF_BROKER_URL                    | Message broker instance URL                                             | nats://localhost:4222 |
| MF_CASSANDRA_WRITER_LOG_LEVEL    | Log level for Cassandra writer (debug, info, warn, error)               | info                  |
| MF_CASSANDRA_WRITER_PORT         | Service HTTP port                                                       | 8180                  |
| MF_CASSANDRA_WRITER_DB_CLUSTER   | Cassandra cluster comma separated addresses                             | 127.0.0.1             |
| MF_CASSANDRA_WRITER_DB_KEYSPACE  | Cassandra keyspace name                                                 | mainflux              |
| MF_CASSANDRA_WRITER_DB_USER      | Cassandra DB username                                                   |                       |
| MF_CASSANDRA_WRITER_DB_PASS      | Cassandra DB password                                                   |                       |
| MF_CASSANDRA_WRITER_DB_PORT      | Cassandra DB port                                                       | 9042                  |
| MF_CASSANDRA_WRITER_CONFIG_PATH  | Config file path with NATS subjects list, payload type and content-type | /config.toml          |

## Deployment
The service itself is distributed as Docker container. Check the [`cassandra-writer`](https://github.com/mainflux/mainflux/blob/master/docker/addons/cassandra-writer/docker-compose.yml#L30-L49) service section in docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/mainflux/mainflux

cd mainflux

# compile the cassandra writer
make cassandra-writer

# copy binary to bin
make install

# Set the environment variables and run the service
MF_BROKER_URL=[NATS instance URL] \
MF_CASSANDRA_WRITER_LOG_LEVEL=[Cassandra writer log level] \
MF_CASSANDRA_WRITER_PORT=[Service HTTP port] \
MF_CASSANDRA_WRITER_DB_CLUSTER=[Cassandra cluster comma separated addresses] \
MF_CASSANDRA_WRITER_DB_KEYSPACE=[Cassandra keyspace name] \
MF_CASSANDRA_READER_DB_USER=[Cassandra DB username] \
MF_CASSANDRA_READER_DB_PASS=[Cassandra DB password] \
MF_CASSANDRA_READER_DB_PORT=[Cassandra DB port] \
MF_CASSANDRA_WRITER_CONFIG_PATH=[Config file path with NATS subjects list, payload type and content-type] \
$GOBIN/mainflux-cassandra-writer
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

[doc]: https://docs.mainflux.io
