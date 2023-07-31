# Timescale reader

Timescale reader provides message repository implementation for Timescale.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                             | Description                                 | Default                        |
| ------------------------------------ | ------------------------------------------- | ------------------------------ |
| MF_TIMESCALE_READER_LOG_LEVEL        | Service log level                           | info                           |
| MF_TIMESCALE_READER_HTTP_HOST        | Service HTTP host                           | localhost                      |
| MF_TIMESCALE_READER_HTTP_PORT        | Service HTTP port                           | 8180                           |
| MF_TIMESCALE_READER_HTTP_SERVER_CERT | Service HTTP server certificate path        | ""                             |
| MF_TIMESCALE_READER_HTTP_SERVER_KEY  | Service HTTP server key path                | ""                             |
| MF_TIMESCALE_HOST                    | Timescale DB host                           | localhost                      |
| MF_TIMESCALE_PORT                    | Timescale DB port                           | 5432                           |
| MF_TIMESCALE_USER                    | Timescale user                              | mainflux                       |
| MF_TIMESCALE_PASS                    | Timescale password                          | mainflux                       |
| MF_TIMESCALE_NAME                    | Timescale database name                     | messages                       |
| MF_TIMESCALE_SSL_MODE                | Timescale SSL mode                          | disabled                       |
| MF_TIMESCALE_SSL_CERT                | Timescale SSL certificate path              | ""                             |
| MF_TIMESCALE_SSL_KEY                 | Timescale SSL key                           | ""                             |
| MF_TIMESCALE_SSL_ROOT_CERT           | Timescale SSL root certificate path         | ""                             |
| MF_THINGS_AUTH_GRPC_URL              | Things service Auth gRPC URL                | localhost:7000                 |
| MF_THINGS_AUTH_GRPC_TIMEOUT          | Things service Auth gRPC timeout in seconds | 1s                             |
| MF_THINGS_AUTH_GRPC_CLIENT_TLS       | Things service Auth gRPC TLS enabled flag   | false                          |
| MF_THINGS_AUTH_GRPC_CA_CERTS         | Things service Auth gRPC CA certificates    | ""                             |
| MF_AUTH_GRPC_URL                     | Users service gRPC URL                      | localhost:7001                 |
| MF_AUTH_GRPC_TIMEOUT                 | Users service gRPC timeout in seconds       | 1s                             |
| MF_AUTH_GRPC_CLIENT_TLS              | Users service gRPC TLS enabled flag         | false                          |
| MF_AUTH_GRPC_CA_CERT                 | Users service gRPC CA certificate           | ""                             |
| MF_JAEGER_URL                        | Jaeger server URL                           | http://jaeger:14268/api/traces |
| MF_SEND_TELEMETRY                    | Send telemetry to mainflux call home server | true                           |
| MF_TIMESCALE_READER_INSTANCE_ID      | Timescale reader instance ID                | ""                             |

## Deployment

The service itself is distributed as Docker container. Check the [`timescale-reader`](https://github.com/mainflux/mainflux/blob/master/docker/addons/timescale-reader/docker-compose.yml#L17-L41) service section in docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/mainflux/mainflux

cd mainflux

# compile the timescale writer
make timescale-writer

# copy binary to bin
make install

# Set the environment variables and run the service
MF_TIMESCALE_READER_LOG_LEVEL=[Service log level] \
MF_TIMESCALE_READER_HTTP_HOST=[Service HTTP host] \
MF_TIMESCALE_READER_HTTP_PORT=[Service HTTP port] \
MF_TIMESCALE_READER_HTTP_SERVER_CERT=[Service HTTP server cert] \
MF_TIMESCALE_READER_HTTP_SERVER_KEY=[Service HTTP server key] \
MF_TIMESCALE_HOST=[Timescale host] \
MF_TIMESCALE_PORT=[Timescale port] \
MF_TIMESCALE_USER=[Timescale user] \
MF_TIMESCALE_PASS=[Timescale password] \
MF_TIMESCALE_NAME=[Timescale database name] \
MF_TIMESCALE_SSL_MODE=[Timescale SSL mode] \
MF_TIMESCALE_SSL_CERT=[Timescale SSL cert] \
MF_TIMESCALE_SSL_KEY=[Timescale SSL key] \
MF_TIMESCALE_SSL_ROOT_CERT=[Timescale SSL Root cert] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth GRPC URL] \
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout in seconds] \
MF_THINGS_AUTH_GRPC_CLIENT_TLS=[Things service Auth gRPC TLS enabled flag] \
MF_THINGS_AUTH_GRPC_CA_CERTS=[Things service Auth gRPC CA certificates] \
MF_AUTH_GRPC_URL=[Users service Auth gRPC URL] \
MF_AUTH_GRPC_TIMEOUT=[Users service Auth gRPC request timeout in seconds] \
MF_AUTH_GRPC_CLIENT_TLS=[Users service Auth gRPC TLS enabled flag] \
MF_AUTH_GRPC_CA_CERT=[Users service Auth gRPC CA certificates] \
MF_JAEGER_URL=[Jaeger server URL] \
MF_SEND_TELEMETRY=[Send telemetry to mainflux call home server] \
MF_TIMESCALE_READER_INSTANCE_ID=[Timescale reader instance ID] \
$GOBIN/mainflux-timescale-reader
```

## Usage

Starting service will start consuming normalized messages in SenML format.

Comparator Usage Guide:
| Comparator | Usage | Example |  
|----------------------|-----------------------------------------------------------------------------|------------------------------------|
| eq | Return values that are equal to the query | eq["active"] -> "active" |  
| ge | Return values that are substrings of the query | ge["tiv"] -> "active" and "tiv" |  
| gt | Return values that are substrings of the query and not equal to the query | gt["tiv"] -> "active" |  
| le | Return values that are superstrings of the query | le["active"] -> "tiv" |  
| lt | Return values that are superstrings of the query and not equal to the query | lt["active"] -> "active" and "tiv" |

Official docs can be found [here](https://docs.mainflux.io).
