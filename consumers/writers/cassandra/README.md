# Cassandra writer

Cassandra writer provides message repository implementation for Cassandra.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                             | Description                                                             | Default                        |
| ------------------------------------ | ----------------------------------------------------------------------- | ------------------------------ |
| MF_CASSANDRA_WRITER_LOG_LEVEL        | Log level for Cassandra writer (debug, info, warn, error)               | info                           |
| MF_CASSANDRA_WRITER_CONFIG_PATH      | Config file path with NATS subjects list, payload type and content-type | /config.toml                   |
| MF_CASSANDRA_WRITER_HTTP_HOST        | Cassandra service HTTP host                                             |                                |
| MF_CASSANDRA_WRITER_HTTP_PORT        | Cassandra service HTTP port                                             | 9004                           |
| MF_CASSANDRA_WRITER_HTTP_SERVER_CERT | Cassandra service HTTP server certificate path                          |                                |
| MF_CASSANDRA_WRITER_HTTP_SERVER_KEY  | Cassandra service HTTP server key path                                  |                                |
| MF_CASSANDRA_CLUSTER                 | Cassandra cluster comma separated addresses                             | 127.0.0.1                      |
| MF_CASSANDRA_KEYSPACE                | Cassandra keyspace name                                                 | mainflux                       |
| MF_CASSANDRA_USER                    | Cassandra DB username                                                   | mainflux                       |
| MF_CASSANDRA_PASS                    | Cassandra DB password                                                   | mainflux                       |
| MF_CASSANDRA_PORT                    | Cassandra DB port                                                       | 9042                           |
| MF_MESSAGE_BROKER_URL                | Message broker instance URL                                             | nats://localhost:4222          |
| MF_JAEGER_URL                        | Jaeger server URL                                                       | http://jaeger:14268/api/traces |
| MF_SEND_TELEMETRY                    | Send telemetry to mainflux call home server                             | true                           |
| MF_CASSANDRA_WRITER_INSANCE_ID       | Cassandra writer instance ID                                            |                                |

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
MF_CASSANDRA_WRITER_LOG_LEVEL=[Cassandra writer log level] \
MF_CASSANDRA_WRITER_CONFIG_PATH=[Config file path with NATS subjects list, payload type and content-type] \
MF_CASSANDRA_WRITER_HTTP_HOST=[Cassandra service HTTP host] \
MF_CASSANDRA_WRITER_HTTP_PORT=[Cassandra service HTTP port] \
MF_CASSANDRA_WRITER_HTTP_SERVER_CERT=[Cassandra service HTTP server cert] \
MF_CASSANDRA_WRITER_HTTP_SERVER_KEY=[Cassandra service HTTP server key] \
MF_CASSANDRA_CLUSTER=[Cassandra cluster comma separated addresses] \
MF_CASSANDRA_KEYSPACE=[Cassandra keyspace name] \
MF_CASSANDRA_USER=[Cassandra DB username] \
MF_CASSANDRA_PASS=[Cassandra DB password] \
MF_CASSANDRA_PORT=[Cassandra DB port] \
MF_MESSAGE_BROKER_URL=[Message Broker instance URL] \
MF_JAEGER_URL=[Jaeger server URL] \
MF_SEND_TELEMETRY=[Send telemetry to mainflux call home server] \
MF_CASSANDRA_WRITER_INSANCE_ID=[Cassandra writer instance ID] \
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
