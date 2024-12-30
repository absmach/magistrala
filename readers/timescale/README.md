# Timescale reader

Timescale reader provides message repository implementation for Timescale.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                              | Description                                  | Default                      |
| ------------------------------------- | -------------------------------------------- | ---------------------------- |
| SMQ_TIMESCALE_READER_LOG_LEVEL        | Service log level                            | info                         |
| SMQ_TIMESCALE_READER_HTTP_HOST        | Service HTTP host                            | localhost                    |
| SMQ_TIMESCALE_READER_HTTP_PORT        | Service HTTP port                            | 8180                         |
| SMQ_TIMESCALE_READER_HTTP_SERVER_CERT | Service HTTP server certificate path         | ""                           |
| SMQ_TIMESCALE_READER_HTTP_SERVER_KEY  | Service HTTP server key path                 | ""                           |
| SMQ_TIMESCALE_HOST                    | Timescale DB host                            | localhost                    |
| SMQ_TIMESCALE_PORT                    | Timescale DB port                            | 5432                         |
| SMQ_TIMESCALE_USER                    | Timescale user                               | supermq                      |
| SMQ_TIMESCALE_PASS                    | Timescale password                           | supermq                      |
| SMQ_TIMESCALE_NAME                    | Timescale database name                      | messages                     |
| SMQ_TIMESCALE_SSL_MODE                | Timescale SSL mode                           | disabled                     |
| SMQ_TIMESCALE_SSL_CERT                | Timescale SSL certificate path               | ""                           |
| SMQ_TIMESCALE_SSL_KEY                 | Timescale SSL key                            | ""                           |
| SMQ_TIMESCALE_SSL_ROOT_CERT           | Timescale SSL root certificate path          | ""                           |
| SMQ_CLIENTS_AUTH_GRPC_URL             | Clients service Auth gRPC URL                | localhost:7000               |
| SMQ_CLIENTS_AUTH_GRPC_TIMEOUT         | Clients service Auth gRPC timeout in seconds | 1s                           |
| SMQ_CLIENTS_AUTH_GRPC_CLIENT_TLS      | Clients service Auth gRPC TLS enabled flag   | false                        |
| SMQ_CLIENTS_AUTH_GRPC_CA_CERTS        | Clients service Auth gRPC CA certificates    | ""                           |
| SMQ_AUTH_GRPC_URL                     | Auth service gRPC URL                        | localhost:7001               |
| SMQ_AUTH_GRPC_TIMEOUT                 | Auth service gRPC timeout in seconds         | 1s                           |
| SMQ_AUTH_GRPC_CLIENT_TLS              | Auth service gRPC TLS enabled flag           | false                        |
| SMQ_AUTH_GRPC_CA_CERT                 | Auth service gRPC CA certificate             | ""                           |
| SMQ_JAEGER_URL                        | Jaeger server URL                            | http://jaeger:4318/v1/traces |
| SMQ_SEND_TELEMETRY                    | Send telemetry to supermq call home server   | true                         |
| SMQ_TIMESCALE_READER_INSTANCE_ID      | Timescale reader instance ID                 | ""                           |

## Deployment

The service itself is distributed as Docker container. Check the [`timescale-reader`](https://github.com/absmach/supermq/blob/main/docker/addons/timescale-reader/docker-compose.yml#L17-L41) service section in docker-compose file to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/supermq

cd supermq

# compile the timescale writer
make timescale-writer

# copy binary to bin
make install

# Set the environment variables and run the service
SMQ_TIMESCALE_READER_LOG_LEVEL=[Service log level] \
SMQ_TIMESCALE_READER_HTTP_HOST=[Service HTTP host] \
SMQ_TIMESCALE_READER_HTTP_PORT=[Service HTTP port] \
SMQ_TIMESCALE_READER_HTTP_SERVER_CERT=[Service HTTP server cert] \
SMQ_TIMESCALE_READER_HTTP_SERVER_KEY=[Service HTTP server key] \
SMQ_TIMESCALE_HOST=[Timescale host] \
SMQ_TIMESCALE_PORT=[Timescale port] \
SMQ_TIMESCALE_USER=[Timescale user] \
SMQ_TIMESCALE_PASS=[Timescale password] \
SMQ_TIMESCALE_NAME=[Timescale database name] \
SMQ_TIMESCALE_SSL_MODE=[Timescale SSL mode] \
SMQ_TIMESCALE_SSL_CERT=[Timescale SSL cert] \
SMQ_TIMESCALE_SSL_KEY=[Timescale SSL key] \
SMQ_TIMESCALE_SSL_ROOT_CERT=[Timescale SSL Root cert] \
SMQ_CLIENTS_AUTH_GRPC_URL=[Clients service Auth GRPC URL] \
SMQ_CLIENTS_AUTH_GRPC_TIMEOUT=[Clients  service Auth gRPC request timeout in seconds] \
SMQ_CLIENTS_AUTH_GRPC_CLIENT_TLS=[Clients  service Auth gRPC TLS enabled flag] \
SMQ_CLIENTS_AUTH_GRPC_CA_CERTS=[Clients  service Auth gRPC CA certificates] \
SMQ_AUTH_GRPC_URL=[Auth service Auth gRPC URL] \
SMQ_AUTH_GRPC_TIMEOUT=[Auth service Auth gRPC request timeout in seconds] \
SMQ_AUTH_GRPC_CLIENT_TLS=[Auth service Auth gRPC TLS enabled flag] \
SMQ_AUTH_GRPC_CA_CERT=[Auth service Auth gRPC CA certificates] \
SMQ_JAEGER_URL=[Jaeger server URL] \
SMQ_SEND_TELEMETRY=[Send telemetry to supermq call home server] \
SMQ_TIMESCALE_READER_INSTANCE_ID=[Timescale reader instance ID] \
$GOBIN/supermq-timescale-reader
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
