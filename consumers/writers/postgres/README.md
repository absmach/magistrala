# Postgres writer

Postgres writer provides message repository implementation for Postgres.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                            | Description                                                                       | Default                        |
| ----------------------------------- | --------------------------------------------------------------------------------- | ------------------------------ |
| MG_POSTGRES_WRITER_LOG_LEVEL        | Service log level                                                                 | info                           |
| MG_POSTGRES_WRITER_CONFIG_PATH      | Config file path with Message broker subjects list, payload type and content-type | /config.toml                   |
| MG_POSTGRES_WRITER_HTTP_HOST        | Service HTTP host                                                                 | localhost                      |
| MG_POSTGRES_WRITER_HTTP_PORT        | Service HTTP port                                                                 | 9010                           |
| MG_POSTGRES_WRITER_HTTP_SERVER_CERT | Service HTTP server certificate path                                              | ""                             |
| MG_POSTGRES_WRITER_HTTP_SERVER_KEY  | Service HTTP server key                                                           | ""                             |
| MG_POSTGRES_HOST                    | Postgres DB host                                                                  | postgres                       |
| MG_POSTGRES_PORT                    | Postgres DB port                                                                  | 5432                           |
| MG_POSTGRES_USER                    | Postgres user                                                                     | magistrala                     |
| MG_POSTGRES_PASS                    | Postgres password                                                                 | magistrala                     |
| MG_POSTGRES_NAME                    | Postgres database name                                                            | messages                       |
| MG_POSTGRES_SSL_MODE                | Postgres SSL mode                                                                 | disabled                       |
| MG_POSTGRES_SSL_CERT                | Postgres SSL certificate path                                                     | ""                             |
| MG_POSTGRES_SSL_KEY                 | Postgres SSL key                                                                  | ""                             |
| MG_POSTGRES_SSL_ROOT_CERT           | Postgres SSL root certificate path                                                | ""                             |
| MG_MESSAGE_BROKER_URL               | Message broker instance URL                                                       | nats://localhost:4222          |
| MG_JAEGER_URL                       | Jaeger server URL                                                                 | http://jaeger:14268/api/traces |
| MG_SEND_TELEMETRY                   | Send telemetry to magistrala call home server                                     | true                           |
| MG_POSTGRES_WRITER_INSTANCE_ID      | Service instance ID                                                               | ""                             |

## Deployment

The service itself is distributed as Docker container. Check the [`postgres-writer`](https://github.com/absmach/magistrala/blob/master/docker/addons/postgres-writer/docker-compose.yml#L34-L59) service section in docker-compose to see how service is deployed.

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
MG_POSTGRES_WRITER_LOG_LEVEL=[Service log level] \
MG_POSTGRES_WRITER_CONFIG_PATH=[Config file path with Message broker subjects list, payload type and content-type] \
MG_POSTGRES_WRITER_HTTP_HOST=[Service HTTP host] \
MG_POSTGRES_WRITER_HTTP_PORT=[Service HTTP port] \
MG_POSTGRES_WRITER_HTTP_SERVER_CERT=[Service HTTP server cert] \
MG_POSTGRES_WRITER_HTTP_SERVER_KEY=[Service HTTP server key] \
MG_POSTGRES_HOST=[Postgres host] \
MG_POSTGRES_PORT=[Postgres port] \
MG_POSTGRES_USER=[Postgres user] \
MG_POSTGRES_PASS=[Postgres password] \
MG_POSTGRES_NAME=[Postgres database name] \
MG_POSTGRES_SSL_MODE=[Postgres SSL mode] \
MG_POSTGRES_SSL_CERT=[Postgres SSL cert] \
MG_POSTGRES_SSL_KEY=[Postgres SSL key] \
MG_POSTGRES_SSL_ROOT_CERT=[Postgres SSL Root cert] \
MG_MESSAGE_BROKER_URL=[Message broker instance URL] \
MG_JAEGER_URL=[Jaeger server URL] \
MG_SEND_TELEMETRY=[Send telemetry to magistrala call home server] \
MG_POSTGRES_WRITER_INSTANCE_ID=[Service instance ID] \

$GOBIN/magistrala-postgres-writer
```

## Usage

Starting service will start consuming normalized messages in SenML format.
