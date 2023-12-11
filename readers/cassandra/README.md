# Cassandra reader

Cassandra reader provides message repository implementation for Cassandra.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                             | Description                                         | Default                        |
| ------------------------------------ | --------------------------------------------------- | ------------------------------ |
| MG_CASSANDRA_READER_LOG_LEVEL        | Cassandra service log level                         | debug                          |
| MG_CASSANDRA_READER_HTTP_HOST        | Cassandra service HTTP host                         | localhost                      |
| MG_CASSANDRA_READER_HTTP_PORT        | Cassandra service HTTP port                         | 9003                           |
| MG_CASSANDRA_READER_HTTP_SERVER_CERT | Cassandra service HTTP server cert                  | ""                             |
| MG_CASSANDRA_READER_HTTP_SERVER_KEY  | Cassandra service HTTP server key                   | ""                             |
| MG_CASSANDRA_CLUSTER                 | Cassandra cluster comma separated addresses         | localhost                      |
| MG_CASSANDRA_USER                    | Cassandra DB username                               | magistrala                     |
| MG_CASSANDRA_PASS                    | Cassandra DB password                               | magistrala                     |
| MG_CASSANDRA_KEYSPACE                | Cassandra keyspace name                             | messages                       |
| MG_CASSANDRA_PORT                    | Cassandra DB port                                   | 9042                           |
| MG_THINGS_AUTH_GRPC_URL              | Things service Auth gRPC URL                        | localhost:7000                 |
| MG_THINGS_AUTH_GRPC_TIMEOUT          | Things service Auth gRPC request timeout in seconds | 1                              |
| MG_THINGS_AUTH_GRPC_CLIENT_TLS       | Things service Auth gRPC TLS enabled                | false                          |
| MG_THINGS_AUTH_GRPC_CA_CERTS         | Things service Auth gRPC CA certificates            | ""                             |
| MG_AUTH_GRPC_URL                     | Auth service gRPC URL                               | localhost:7001                 |
| MG_AUTH_GRPC_TIMEOUT                 | Auth service gRPC request timeout in seconds        | 1s                             |
| MG_AUTH_GRPC_CLIENT_TLS              | Auth service gRPC TLS enabled                       | false                          |
| MG_AUTH_GRPC_CA_CERT                 | Auth service gRPC CA certificates                   | ""                             |
| MG_JAEGER_URL                        | Jaeger server URL                                   | http://jaeger:14268/api/traces |
| MG_SEND_TELEMETRY                    | Send telemetry to magistrala call home server       | true                           |
| MG_CASSANDRA_READER_INSTANCE_ID      | Cassandra Reader instance ID                        | ""                             |

## Deployment

The service itself is distributed as Docker container. Check the [`cassandra-reader`](https://github.com/absmach/magistrala/blob/master/docker/addons/cassandra-reader/docker-compose.yml#L15-L35) service section in
docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/magistrala

cd magistrala

# compile the cassandra
make cassandra-reader

# copy binary to bin
make install

# Set the environment variables and run the service
MG_CASSANDRA_READER_LOG_LEVEL=[Cassandra Service log level] \
MG_CASSANDRA_READER_HTTP_HOST=[Cassandra Service HTTP host] \
MG_CASSANDRA_READER_HTTP_PORT=[Cassandra Service HTTP port] \
MG_CASSANDRA_READER_HTTP_SERVER_CERT=[Cassandra Service HTTP server cert] \
MG_CASSANDRA_READER_HTTP_SERVER_KEY=[Cassandra Service HTTP server key] \
MG_CASSANDRA_CLUSTER=[Cassandra cluster comma separated addresses] \
MG_CASSANDRA_KEYSPACE=[Cassandra keyspace name] \
MG_CASSANDRA_USER=[Cassandra DB username] \
MG_CASSANDRA_PASS=[Cassandra DB password] \
MG_CASSANDRA_PORT=[Cassandra DB port] \
MG_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MG_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout in seconds] \
MG_THINGS_AUTH_GRPC_CLIENT_TLS=[Things service Auth gRPC TLS enabled] \
MG_THINGS_AUTH_GRPC_CA_CERTS=[Things service Auth gRPC CA certificates] \
MG_AUTH_GRPC_URL=[Auth service gRPC URL] \
MG_AUTH_GRPC_TIMEOUT=[Auth service gRPC request timeout in seconds] \
MG_AUTH_GRPC_CLIENT_TLS=[Auth service gRPC TLS enabled] \
MG_AUTH_GRPC_CA_CERT=[Auth service gRPC CA certificates] \
MG_JAEGER_URL=[Jaeger server URL] \
MG_SEND_TELEMETRY=[Send telemetry to magistrala call home server] \
MG_CASSANDRA_READER_INSTANCE_ID=[Cassandra Reader instance ID] \
$GOBIN/magistrala-cassandra-reader
```

### Using docker-compose

This service can be deployed using docker containers. Docker compose file is
available in `<project_root>/docker/addons/cassandra-reader/docker-compose.yml`.
In order to run all Magistrala core services, as well as mentioned optional ones,
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
