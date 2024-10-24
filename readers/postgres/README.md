# Postgres reader

Postgres reader provides message repository implementation for Postgres.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                            | Description                                   | Default                      |
| ----------------------------------- | --------------------------------------------- | ---------------------------- |
| MG_POSTGRES_READER_LOG_LEVEL        | Service log level                             | info                         |
| MG_POSTGRES_READER_HTTP_HOST        | Service HTTP host                             | localhost                    |
| MG_POSTGRES_READER_HTTP_PORT        | Service HTTP port                             | 9009                         |
| MG_POSTGRES_READER_HTTP_SERVER_CERT | Service HTTP server cert                      | ""                           |
| MG_POSTGRES_READER_HTTP_SERVER_KEY  | Service HTTP server key                       | ""                           |
| MG_POSTGRES_HOST                    | Postgres DB host                              | localhost                    |
| MG_POSTGRES_PORT                    | Postgres DB port                              | 5432                         |
| MG_POSTGRES_USER                    | Postgres user                                 | magistrala                   |
| MG_POSTGRES_PASS                    | Postgres password                             | magistrala                   |
| MG_POSTGRES_NAME                    | Postgres database name                        | messages                     |
| MG_POSTGRES_SSL_MODE                | Postgres SSL mode                             | disabled                     |
| MG_POSTGRES_SSL_CERT                | Postgres SSL certificate path                 | ""                           |
| MG_POSTGRES_SSL_KEY                 | Postgres SSL key                              | ""                           |
| MG_POSTGRES_SSL_ROOT_CERT           | Postgres SSL root certificate path            | ""                           |
| MG_CLIENTS_AUTH_GRPC_URL            | Clients service Auth gRPC URL                 | localhost:7000               |
| MG_CLIENTS_AUTH_GRPC_TIMEOUT        | Clients service Auth gRPC timeout in seconds  | 1s                           |
| MG_CLIENTS_AUTH_GRPC_CLIENT_TLS     | Clients service Auth gRPC TLS mode flag       | false                        |
| MG_CLIENTS_AUTH_GRPC_CA_CERTS       | Clients service Auth gRPC CA certificates     | ""                           |
| MG_AUTH_GRPC_URL                    | Auth service gRPC URL                         | localhost:7001               |
| MG_AUTH_GRPC_TIMEOUT                | Auth service gRPC request timeout in seconds  | 1s                           |
| MG_AUTH_GRPC_CLIENT_TLS             | Auth service gRPC TLS mode flag               | false                        |
| MG_AUTH_GRPC_CA_CERTS               | Auth service gRPC CA certificates             | ""                           |
| MG_JAEGER_URL                       | Jaeger server URL                             | http://jaeger:4318/v1/traces |
| MG_SEND_TELEMETRY                   | Send telemetry to magistrala call home server | true                         |
| MG_POSTGRES_READER_INSTANCE_ID      | Postgres reader instance ID                   |                              |

## Deployment

The service itself is distributed as Docker container. Check the [`postgres-reader`](https://github.com/absmach/magistrala/blob/main/docker/addons/postgres-reader/docker-compose.yml#L17-L41) service section in
docker-compose file to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/magistrala

cd magistrala

# compile the postgres writer
make postgres-writer

# copy binary to bin
make install

# Set the environment variables and run the service
MG_POSTGRES_READER_LOG_LEVEL=[Service log level] \
MG_POSTGRES_READER_HTTP_HOST=[Service HTTP host] \
MG_POSTGRES_READER_HTTP_PORT=[Service HTTP port] \
MG_POSTGRES_READER_HTTP_SERVER_CERT=[Service HTTPS server certificate path] \
MG_POSTGRES_READER_HTTP_SERVER_KEY=[Service HTTPS server key path] \
MG_POSTGRES_HOST=[Postgres host] \
MG_POSTGRES_PORT=[Postgres port] \
MG_POSTGRES_USER=[Postgres user] \
MG_POSTGRES_PASS=[Postgres password] \
MG_POSTGRES_NAME=[Postgres database name] \
MG_POSTGRES_SSL_MODE=[Postgres SSL mode] \
MG_POSTGRES_SSL_CERT=[Postgres SSL cert] \
MG_POSTGRES_SSL_KEY=[Postgres SSL key] \
MG_POSTGRES_SSL_ROOT_CERT=[Postgres SSL Root cert] \
MG_CLIENTS_AUTH_GRPC_URL=[Clients service Auth GRPC URL] \
MG_CLIENTS_AUTH_GRPC_TIMEOUT=[Clients service Auth gRPC request timeout in seconds] \
MG_CLIENTS_AUTH_GRPC_CLIENT_TLS=[Clients service Auth gRPC TLS mode flag] \
MG_CLIENTS_AUTH_GRPC_CA_CERTS=[Clients service Auth gRPC CA certificates] \
MG_AUTH_GRPC_URL=[Auth service gRPC URL] \
MG_AUTH_GRPC_TIMEOUT=[Auth service gRPC request timeout in seconds] \
MG_AUTH_GRPC_CLIENT_TLS=[Auth service gRPC TLS mode flag] \
MG_AUTH_GRPC_CA_CERTS=[Auth service gRPC CA certificates] \
MG_JAEGER_URL=[Jaeger server URL] \
MG_SEND_TELEMETRY=[Send telemetry to magistrala call home server] \
MG_POSTGRES_READER_INSTANCE_ID=[Postgres reader instance ID] \
$GOBIN/magistrala-postgres-reader
```

## Usage

Starting service will start consuming normalized messages in SenML format.

Comparator Usage Guide:

| Comparator | Usage                                                                       | Example                            |
| ---------- | --------------------------------------------------------------------------- | ---------------------------------- |
| eq         | Return values that are equal to the query                                   | eq["active"] -> "active"           |
| ge         | Return values that are substrings of the query                              | ge["tiv"] -> "active" and "tiv"    |
| gt         | Return values that are substrings of the query and not equal to the query   | gt["tiv"] -> "active"              |
| le         | Return values that are superstrings of the query                            | le["active"] -> "tiv"              |
| lt         | Return values that are superstrings of the query and not equal to the query | lt["active"] -> "active" and "tiv" |

Official docs can be found [here](https://docs.magistrala.abstractmachines.fr).
