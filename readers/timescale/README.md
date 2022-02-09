# Timescale reader

Timescale reader provides message repository implementation for Timescale.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                             | Description                                 | Default        |
|--------------------------------------|---------------------------------------------|----------------|
| MF_TIMESCALE_READER_LOG_LEVEL        | Service log level                           | debug          |
| MF_TIMESCALE_READER_PORT             | Service HTTP port                           | 8180           |
| MF_TIMESCALE_READER_CLIENT_TLS       | TLS mode flag                               | false          |
| MF_TIMESCALE_READER_CA_CERTS         | Path to trusted CAs in PEM format           |                |
| MF_TIMESCALE_READER_DB_HOST          | Timescale DB host                           | timescale       |
| MF_TIMESCALE_READER_DB_PORT          | Timescale DB port                           | 5432           |
| MF_TIMESCALE_READER_DB_USER          | Timescale user                              | mainflux       |
| MF_TIMESCALE_READER_DB_PASS          | Timescale password                          | mainflux       |
| MF_TIMESCALE_READER_DB               | Timescale database name                     | messages       |
| MF_TIMESCALE_READER_DB_SSL_MODE      | Timescale SSL mode                          | disabled       |
| MF_TIMESCALE_READER_DB_SSL_CERT      | Timescale SSL certificate path              | ""             |
| MF_TIMESCALE_READER_DB_SSL_KEY       | Timescale SSL key                           | ""             |
| MF_TIMESCALE_READER_DB_SSL_ROOT_CERT | Timescale SSL root certificate path         | ""             |
| MF_JAEGER_URL                        | Jaeger server URL                           | localhost:6831 |
| MF_THINGS_AUTH_GRPC_URL              | Things service Auth gRPC URL                | localhost:8183 |
| MF_THINGS_AUTH_GRPC_TIMEOUT          | Things service Auth gRPC timeout in seconds | 1s             |

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
MF_TIMESCALE_READER_PORT=[Service HTTP port] \
MF_TIMESCALE_READER_CLIENT_TLS =[TLS mode flag] \
MF_TIMESCALE_READER_CA_CERTS=[Path to trusted CAs in PEM format] \
MF_TIMESCALE_READER_DB_HOST=[Timescale host] \
MF_TIMESCALE_READER_DB_PORT=[Timescale port] \
MF_TIMESCALE_READER_DB_USER=[Timescale user] \
MF_TIMESCALE_READER_DB_PASS=[Timescale password] \
MF_TIMESCALE_READER_DB=[Timescale database name] \
MF_TIMESCALE_READER_DB_SSL_MODE=[Timescale SSL mode] \
MF_TIMESCALE_READER_DB_SSL_CERT=[Timescale SSL cert] \
MF_TIMESCALE_READER_DB_SSL_KEY=[Timescale SSL key] \
MF_TIMESCALE_READER_DB_SSL_ROOT_CERT=[Timescale SSL Root cert] \
MF_JAEGER_URL=[Jaeger server URL] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth GRPC URL] \
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout in seconds] \
$GOBIN/mainflux-timescale-reader
```

## Usage

Starting service will start consuming normalized messages in SenML format.
