# Postgres writer

Postgres writer provides message repository implementation for Postgres.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                            | Description                                                                       | Default                        |
| ----------------------------------- | --------------------------------------------------------------------------------- | ------------------------------ |
| MF_POSTGRES_WRITER_LOG_LEVEL        | Service log level                                                                 | info                           |
| MF_POSTGRES_WRITER_CONFIG_PATH      | Config file path with Message broker subjects list, payload type and content-type | /config.toml                   |
| MF_POSTGRES_WRITER_HTTP_HOST        | Service HTTP host                                                                 | localhost                      |
| MF_POSTGRES_WRITER_HTTP_PORT        | Service HTTP port                                                                 | 9010                           |
| MF_POSTGRES_WRITER_HTTP_SERVER_CERT | Service HTTP server certificate path                                              | ""                             |
| MF_POSTGRES_WRITER_HTTP_SERVER_KEY  | Service HTTP server key                                                           | ""                             |
| MF_POSTGRES_HOST                    | Postgres DB host                                                                  | postgres                       |
| MF_POSTGRES_PORT                    | Postgres DB port                                                                  | 5432                           |
| MF_POSTGRES_USER                    | Postgres user                                                                     | mainflux                       |
| MF_POSTGRES_PASS                    | Postgres password                                                                 | mainflux                       |
| MF_POSTGRES_NAME                    | Postgres database name                                                            | messages                       |
| MF_POSTGRES_SSL_MODE                | Postgres SSL mode                                                                 | disabled                       |
| MF_POSTGRES_SSL_CERT                | Postgres SSL certificate path                                                     | ""                             |
| MF_POSTGRES_SSL_KEY                 | Postgres SSL key                                                                  | ""                             |
| MF_POSTGRES_SSL_ROOT_CERT           | Postgres SSL root certificate path                                                | ""                             |
| MF_BROKER_URL                       | Message broker instance URL                                                       | nats://localhost:4222          |
| MF_JAEGER_URL                       | Jaeger server URL                                                                 | http://jaeger:14268/api/traces |
| MF_SEND_TELEMETRY                   | Send telemetry to mainflux call home server                                       | true                           |
| MF_POSTGRES_WRITER_INSTANCE_ID      | Service instance ID                                                               | ""                             |

## Deployment

The service itself is distributed as Docker container. Check the [`postgres-writer`](https://github.com/mainflux/mainflux/blob/master/docker/addons/postgres-writer/docker-compose.yml#L34-L59) service section in docker-compose to see how service is deployed.

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
MF_POSTGRES_WRITER_LOG_LEVEL=[Service log level] \
MF_POSTGRES_WRITER_CONFIG_PATH=[Config file path with Message broker subjects list, payload type and content-type] \
MF_POSTGRES_WRITER_HTTP_HOST=[Service HTTP host] \
MF_POSTGRES_WRITER_HTTP_PORT=[Service HTTP port] \
MF_POSTGRES_WRITER_HTTP_SERVER_CERT=[Service HTTP server cert] \
MF_POSTGRES_WRITER_HTTP_SERVER_KEY=[Service HTTP server key] \
MF_POSTGRES_HOST=[Postgres host] \
MF_POSTGRES_PORT=[Postgres port] \
MF_POSTGRES_USER=[Postgres user] \
MF_POSTGRES_PASS=[Postgres password] \
MF_POSTGRES_NAME=[Postgres database name] \
MF_POSTGRES_SSL_MODE=[Postgres SSL mode] \
MF_POSTGRES_SSL_CERT=[Postgres SSL cert] \
MF_POSTGRES_SSL_KEY=[Postgres SSL key] \
MF_POSTGRES_SSL_ROOT_CERT=[Postgres SSL Root cert] \
MF_BROKER_URL=[Message broker instance URL] \
MF_JAEGER_URL=[Jaeger server URL] \
MF_SEND_TELEMETRY=[Send telemetry to mainflux call home server] \
MF_POSTGRES_WRITER_INSTANCE_ID=[Service instance ID] \

$GOBIN/mainflux-postgres-writer
```

## Usage

Starting service will start consuming normalized messages in SenML format.
