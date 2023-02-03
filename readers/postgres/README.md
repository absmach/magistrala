# Postgres reader

Postgres reader provides message repository implementation for Postgres.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                            | Description                                  | Default        |
|-------------------------------------|----------------------------------------------|----------------|
| MF_POSTGRES_READER_LOG_LEVEL        | Service log level                            | info           |
| MF_POSTGRES_READER_PORT             | Service HTTP port                            | 8180           |
| MF_POSTGRES_READER_CLIENT_TLS       | TLS mode flag                                | false          |
| MF_POSTGRES_READER_CA_CERTS         | Path to trusted CAs in PEM format            |                |
| MF_POSTGRES_READER_DB_HOST          | Postgres DB host                             | postgres       |
| MF_POSTGRES_READER_DB_PORT          | Postgres DB port                             | 5432           |
| MF_POSTGRES_READER_DB_USER          | Postgres user                                | mainflux       |
| MF_POSTGRES_READER_DB_PASS          | Postgres password                            | mainflux       |
| MF_POSTGRES_READER_DB               | Postgres database name                       | messages       |
| MF_POSTGRES_READER_DB_SSL_MODE      | Postgres SSL mode                            | disabled       |
| MF_POSTGRES_READER_DB_SSL_CERT      | Postgres SSL certificate path                | ""             |
| MF_POSTGRES_READER_DB_SSL_KEY       | Postgres SSL key                             | ""             |
| MF_POSTGRES_READER_DB_SSL_ROOT_CERT | Postgres SSL root certificate path           | ""             |
| MF_JAEGER_URL                       | Jaeger server URL                            | localhost:6831 |
| MF_THINGS_AUTH_GRPC_URL             | Things service Auth gRPC URL                 | localhost:8183 |
| MF_THINGS_AUTH_GRPC_TIMEOUT         | Things service Auth gRPC timeout in seconds  | 1s             |
| MF_AUTH_GRPC_URL                    | Auth service gRPC URL                        | localhost:8181 |
| MF_AUTH_GRPC_TIMEOUT                | Auth service gRPC request timeout in seconds | 1s             |

## Deployment

The service itself is distributed as Docker container. Check the [`postgres-reader`](https://github.com/mainflux/mainflux/blob/master/docker/addons/postgres-reader/docker-compose.yml#L17-L41) service section in 
docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/mainflux/mainflux

cd mainflux

# compile the postgres writer
make postgres-writer

# copy binary to bin
make install

# Set the environment variables and run the service
MF_POSTGRES_READER_LOG_LEVEL=[Service log level] \
MF_POSTGRES_READER_PORT=[Service HTTP port] \
MF_POSTGRES_READER_CLIENT_TLS =[TLS mode flag] \
MF_POSTGRES_READER_CA_CERTS=[Path to trusted CAs in PEM format] \
MF_POSTGRES_READER_DB_HOST=[Postgres host] \
MF_POSTGRES_READER_DB_PORT=[Postgres port] \
MF_POSTGRES_READER_DB_USER=[Postgres user] \
MF_POSTGRES_READER_DB_PASS=[Postgres password] \
MF_POSTGRES_READER_DB=[Postgres database name] \
MF_POSTGRES_READER_DB_SSL_MODE=[Postgres SSL mode] \
MF_POSTGRES_READER_DB_SSL_CERT=[Postgres SSL cert] \
MF_POSTGRES_READER_DB_SSL_KEY=[Postgres SSL key] \
MF_POSTGRES_READER_DB_SSL_ROOT_CERT=[Postgres SSL Root cert] \
MF_JAEGER_URL=[Jaeger server URL] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth GRPC URL] \
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout in seconds] \
$GOBIN/mainflux-postgres-reader
```

## Usage

Starting service will start consuming normalized messages in SenML format.
