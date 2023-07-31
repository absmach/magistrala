# Cassandra reader

Cassandra reader provides message repository implementation for Cassandra.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                             | Description                                         | Default                        |
| ------------------------------------ | --------------------------------------------------- | ------------------------------ |
| MF_CASSANDRA_READER_LOG_LEVEL        | Cassandra service log level                         | debug                          |
| MF_CASSANDRA_READER_HTTP_HOST        | Cassandra service HTTP host                         | localhost                      |
| MF_CASSANDRA_READER_HTTP_PORT        | Cassandra service HTTP port                         | 9003                           |
| MF_CASSANDRA_READER_HTTP_SERVER_CERT | Cassandra service HTTP server cert                  | ""                             |
| MF_CASSANDRA_READER_HTTP_SERVER_KEY  | Cassandra service HTTP server key                   | ""                             |
| MF_CASSANDRA_CLUSTER                 | Cassandra cluster comma separated addresses         | localhost                      |
| MF_CASSANDRA_USER                    | Cassandra DB username                               | mainflux                       |
| MF_CASSANDRA_PASS                    | Cassandra DB password                               | mainflux                       |
| MF_CASSANDRA_KEYSPACE                | Cassandra keyspace name                             | messages                       |
| MF_CASSANDRA_PORT                    | Cassandra DB port                                   | 9042                           |
| MF_THINGS_AUTH_GRPC_URL              | Things service Auth gRPC URL                        | localhost:7000                 |
| MF_THINGS_AUTH_GRPC_TIMEOUT          | Things service Auth gRPC request timeout in seconds | 1                              |
| MF_THINGS_AUTH_GRPC_CLIENT_TLS       | Things service Auth gRPC TLS enabled                | false                          |
| MF_THINGS_AUTH_GRPC_CA_CERTS         | Things service Auth gRPC CA certificates            | ""                             |
| MF_AUTH_GRPC_URL                     | Users service gRPC URL                              | localhost:7001                 |
| MF_AUTH_GRPC_TIMEOUT                 | Users service gRPC request timeout in seconds       | 1s                             |
| MF_AUTH_GRPC_CLIENT_TLS              | Users service gRPC TLS enabled                      | false                          |
| MF_AUTH_GRPC_CA_CERT                 | Users service gRPC CA certificates                  | ""                             |
| MF_JAEGER_URL                        | Jaeger server URL                                   | http://jaeger:14268/api/traces |
| MF_SEND_TELEMETRY                    | Send telemetry to mainflux call home server         | true                           |
| MF_CASSANDRA_READER_INSTANCE_ID      | Cassandra Reader instance ID                        | ""                             |

## Deployment

The service itself is distributed as Docker container. Check the [`cassandra-reader`](https://github.com/mainflux/mainflux/blob/master/docker/addons/cassandra-reader/docker-compose.yml#L15-L35) service section in
docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/mainflux/mainflux

cd mainflux

# compile the cassandra
make cassandra-reader

# copy binary to bin
make install

# Set the environment variables and run the service
MF_CASSANDRA_READER_LOG_LEVEL=[Cassandra Service log level] \
MF_CASSANDRA_READER_HTTP_HOST=[Cassandra Service HTTP host] \
MF_CASSANDRA_READER_HTTP_PORT=[Cassandra Service HTTP port] \
MF_CASSANDRA_READER_HTTP_SERVER_CERT=[Cassandra Service HTTP server cert] \
MF_CASSANDRA_READER_HTTP_SERVER_KEY=[Cassandra Service HTTP server key] \
MF_CASSANDRA_CLUSTER=[Cassandra cluster comma separated addresses] \
MF_CASSANDRA_KEYSPACE=[Cassandra keyspace name] \
MF_CASSANDRA_USER=[Cassandra DB username] \
MF_CASSANDRA_PASS=[Cassandra DB password] \
MF_CASSANDRA_PORT=[Cassandra DB port] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout in seconds] \
MF_THINGS_AUTH_GRPC_CLIENT_TLS=[Things service Auth gRPC TLS enabled] \
MF_THINGS_AUTH_GRPC_CA_CERTS=[Things service Auth gRPC CA certificates] \
MF_AUTH_GRPC_URL=[Users service gRPC URL] \
MF_AUTH_GRPC_TIMEOUT=[Users service gRPC request timeout in seconds] \
MF_AUTH_GRPC_CLIENT_TLS=[Users service gRPC TLS enabled] \
MF_AUTH_GRPC_CA_CERT=[Users service gRPC CA certificates] \
MF_JAEGER_URL=[Jaeger server URL] \
MF_SEND_TELEMETRY=[Send telemetry to mainflux call home server] \
MF_CASSANDRA_READER_INSTANCE_ID=[Cassandra Reader instance ID] \
$GOBIN/mainflux-cassandra-reader
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

Service exposes [HTTP API](https://api.mainflux.io/?urls.primaryName=readers-openapi.yml) for fetching messages.

```
Note: Cassandra Reader doesn't support searching substrings from string_value, due to inefficient searching as the current data model is not suitable for this type of queries.
```

[doc]: https://docs.mainflux.io
