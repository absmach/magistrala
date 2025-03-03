# Postgres reader

Postgres reader provides message repository implementation for Postgres.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                             | Description                                  | Default                      |
| ------------------------------------ | -------------------------------------------- | ---------------------------- |
| SMQ_POSTGRES_READER_LOG_LEVEL        | Service log level                            | info                         |
| SMQ_POSTGRES_READER_HTTP_HOST        | Service HTTP host                            | localhost                    |
| SMQ_POSTGRES_READER_HTTP_PORT        | Service HTTP port                            | 9009                         |
| SMQ_POSTGRES_READER_HTTP_SERVER_CERT | Service HTTP server cert                     | ""                           |
| SMQ_POSTGRES_READER_HTTP_SERVER_KEY  | Service HTTP server key                      | ""                           |
| SMQ_POSTGRES_HOST                    | Postgres DB host                             | localhost                    |
| SMQ_POSTGRES_PORT                    | Postgres DB port                             | 5432                         |
| SMQ_POSTGRES_USER                    | Postgres user                                | supermq                      |
| SMQ_POSTGRES_PASS                    | Postgres password                            | supermq                      |
| SMQ_POSTGRES_NAME                    | Postgres database name                       | messages                     |
| SMQ_POSTGRES_SSL_MODE                | Postgres SSL mode                            | disabled                     |
| SMQ_POSTGRES_SSL_CERT                | Postgres SSL certificate path                | ""                           |
| SMQ_POSTGRES_SSL_KEY                 | Postgres SSL key                             | ""                           |
| SMQ_POSTGRES_SSL_ROOT_CERT           | Postgres SSL root certificate path           | ""                           |
| SMQ_CLIENTS_GRPC_URL            | Clients service Auth gRPC URL                | localhost:7000               |
| SMQ_CLIENTS_GRPC_TIMEOUT        | Clients service Auth gRPC timeout in seconds | 1s                           |
| SMQ_CLIENTS_GRPC_CLIENT_TLS     | Clients service Auth gRPC TLS mode flag      | false                        |
| SMQ_CLIENTS_GRPC_CA_CERTS       | Clients service Auth gRPC CA certificates    | ""                           |
| SMQ_AUTH_GRPC_URL                    | Auth service gRPC URL                        | localhost:7001               |
| SMQ_AUTH_GRPC_TIMEOUT                | Auth service gRPC request timeout in seconds | 1s                           |
| SMQ_AUTH_GRPC_CLIENT_TLS             | Auth service gRPC TLS mode flag              | false                        |
| SMQ_AUTH_GRPC_CA_CERTS               | Auth service gRPC CA certificates            | ""                           |
| SMQ_JAEGER_URL                       | Jaeger server URL                            | http://jaeger:4318/v1/traces |
| SMQ_SEND_TELEMETRY                   | Send telemetry to supermq call home server   | true                         |
| SMQ_POSTGRES_READER_INSTANCE_ID      | Postgres reader instance ID                  |                              |

## Deployment

The service itself is distributed as Docker container. Check the [`postgres-reader`](https://github.com/absmach/supermq/blob/main/docker/addons/postgres-reader/docker-compose.yml#L17-L41) service section in
docker-compose file to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/supermq

cd supermq

# compile the postgres writer
make postgres-writer

# copy binary to bin
make install

# Set the environment variables and run the service
SMQ_POSTGRES_READER_LOG_LEVEL=[Service log level] \
SMQ_POSTGRES_READER_HTTP_HOST=[Service HTTP host] \
SMQ_POSTGRES_READER_HTTP_PORT=[Service HTTP port] \
SMQ_POSTGRES_READER_HTTP_SERVER_CERT=[Service HTTPS server certificate path] \
SMQ_POSTGRES_READER_HTTP_SERVER_KEY=[Service HTTPS server key path] \
SMQ_POSTGRES_HOST=[Postgres host] \
SMQ_POSTGRES_PORT=[Postgres port] \
SMQ_POSTGRES_USER=[Postgres user] \
SMQ_POSTGRES_PASS=[Postgres password] \
SMQ_POSTGRES_NAME=[Postgres database name] \
SMQ_POSTGRES_SSL_MODE=[Postgres SSL mode] \
SMQ_POSTGRES_SSL_CERT=[Postgres SSL cert] \
SMQ_POSTGRES_SSL_KEY=[Postgres SSL key] \
SMQ_POSTGRES_SSL_ROOT_CERT=[Postgres SSL Root cert] \
SMQ_CLIENTS_GRPC_URL=[Clients service Auth GRPC URL] \
SMQ_CLIENTS_GRPC_TIMEOUT=[Clients service Auth gRPC request timeout in seconds] \
SMQ_CLIENTS_GRPC_CLIENT_TLS=[Clients service Auth gRPC TLS mode flag] \
SMQ_CLIENTS_GRPC_CA_CERTS=[Clients service Auth gRPC CA certificates] \
SMQ_AUTH_GRPC_URL=[Auth service gRPC URL] \
SMQ_AUTH_GRPC_TIMEOUT=[Auth service gRPC request timeout in seconds] \
SMQ_AUTH_GRPC_CLIENT_TLS=[Auth service gRPC TLS mode flag] \
SMQ_AUTH_GRPC_CA_CERTS=[Auth service gRPC CA certificates] \
SMQ_JAEGER_URL=[Jaeger server URL] \
SMQ_SEND_TELEMETRY=[Send telemetry to supermq call home server] \
SMQ_POSTGRES_READER_INSTANCE_ID=[Postgres reader instance ID] \
$GOBIN/supermq-postgres-reader
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

Official docs can be found [here](https://docs.supermq.abstractmachines.fr).
