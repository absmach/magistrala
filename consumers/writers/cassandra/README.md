# Cassandra writer

Cassandra writer provides message repository implementation for Cassandra.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                             | Description                                                             | Default                        |
| ------------------------------------ | ----------------------------------------------------------------------- | ------------------------------ |
| MG_CASSANDRA_WRITER_LOG_LEVEL        | Log level for Cassandra writer (debug, info, warn, error)               | info                           |
| MG_CASSANDRA_WRITER_CONFIG_PATH      | Config file path with NATS subjects list, payload type and content-type | /config.toml                   |
| MG_CASSANDRA_WRITER_HTTP_HOST        | Cassandra service HTTP host                                             |                                |
| MG_CASSANDRA_WRITER_HTTP_PORT        | Cassandra service HTTP port                                             | 9004                           |
| MG_CASSANDRA_WRITER_HTTP_SERVER_CERT | Cassandra service HTTP server certificate path                          |                                |
| MG_CASSANDRA_WRITER_HTTP_SERVER_KEY  | Cassandra service HTTP server key path                                  |                                |
| MG_CASSANDRA_CLUSTER                 | Cassandra cluster comma separated addresses                             | 127.0.0.1                      |
| MG_CASSANDRA_KEYSPACE                | Cassandra keyspace name                                                 | magistrala                     |
| MG_CASSANDRA_USER                    | Cassandra DB username                                                   | magistrala                     |
| MG_CASSANDRA_PASS                    | Cassandra DB password                                                   | magistrala                     |
| MG_CASSANDRA_PORT                    | Cassandra DB port                                                       | 9042                           |
| MG_MESSAGE_BROKER_URL                | Message broker instance URL                                             | nats://localhost:4222          |
| MG_JAEGER_URL                        | Jaeger server URL                                                       | http://jaeger:14268/api/traces |
| MG_SEND_TELEMETRY                    | Send telemetry to magistrala call home server                           | true                           |
| MG_CASSANDRA_WRITER_INSANCE_ID       | Cassandra writer instance ID                                            |                                |

## Deployment

The service itself is distributed as Docker container. Check the [`cassandra-writer`](https://github.com/absmach/magistrala/blob/master/docker/addons/cassandra-writer/docker-compose.yml#L30-L49) service section in docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/magistrala

cd magistrala

# compile the cassandra writer
make cassandra-writer

# copy binary to bin
make install

# Set the environment variables and run the service
MG_CASSANDRA_WRITER_LOG_LEVEL=[Cassandra writer log level] \
MG_CASSANDRA_WRITER_CONFIG_PATH=[Config file path with NATS subjects list, payload type and content-type] \
MG_CASSANDRA_WRITER_HTTP_HOST=[Cassandra service HTTP host] \
MG_CASSANDRA_WRITER_HTTP_PORT=[Cassandra service HTTP port] \
MG_CASSANDRA_WRITER_HTTP_SERVER_CERT=[Cassandra service HTTP server cert] \
MG_CASSANDRA_WRITER_HTTP_SERVER_KEY=[Cassandra service HTTP server key] \
MG_CASSANDRA_CLUSTER=[Cassandra cluster comma separated addresses] \
MG_CASSANDRA_KEYSPACE=[Cassandra keyspace name] \
MG_CASSANDRA_USER=[Cassandra DB username] \
MG_CASSANDRA_PASS=[Cassandra DB password] \
MG_CASSANDRA_PORT=[Cassandra DB port] \
MG_MESSAGE_BROKER_URL=[Message Broker instance URL] \
MG_JAEGER_URL=[Jaeger server URL] \
MG_SEND_TELEMETRY=[Send telemetry to magistrala call home server] \
MG_CASSANDRA_WRITER_INSANCE_ID=[Cassandra writer instance ID] \
$GOBIN/magistrala-cassandra-writer
```

### Using docker-compose

This service can be deployed using docker containers. Docker compose file is
available in `<project_root>/docker/addons/cassandra-writer/docker-compose.yml`.
In order to run all Magistrala core services, as well as mentioned optional ones,
execute following command:

```bash
./docker/addons/cassandra-writer/init.sh
```

## Usage

Starting service will start consuming normalized messages in SenML format.

[doc]: https://docs.mainflux.io
