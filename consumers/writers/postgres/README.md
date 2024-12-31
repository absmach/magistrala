# Postgres writer

Postgres writer provides message repository implementation for Postgres.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                             | Description                                                                       | Default                      |
| ------------------------------------ | --------------------------------------------------------------------------------- | ---------------------------- |
| SMQ_POSTGRES_WRITER_LOG_LEVEL        | Service log level                                                                 | info                         |
| SMQ_POSTGRES_WRITER_CONFIG_PATH      | Config file path with Message broker subjects list, payload type and content-type | /config.toml                 |
| SMQ_POSTGRES_WRITER_HTTP_HOST        | Service HTTP host                                                                 | localhost                    |
| SMQ_POSTGRES_WRITER_HTTP_PORT        | Service HTTP port                                                                 | 9010                         |
| SMQ_POSTGRES_WRITER_HTTP_SERVER_CERT | Service HTTP server certificate path                                              | ""                           |
| SMQ_POSTGRES_WRITER_HTTP_SERVER_KEY  | Service HTTP server key                                                           | ""                           |
| SMQ_POSTGRES_HOST                    | Postgres DB host                                                                  | postgres                     |
| SMQ_POSTGRES_PORT                    | Postgres DB port                                                                  | 5432                         |
| SMQ_POSTGRES_USER                    | Postgres user                                                                     | supermq                      |
| SMQ_POSTGRES_PASS                    | Postgres password                                                                 | supermq                      |
| SMQ_POSTGRES_NAME                    | Postgres database name                                                            | messages                     |
| SMQ_POSTGRES_SSL_MODE                | Postgres SSL mode                                                                 | disabled                     |
| SMQ_POSTGRES_SSL_CERT                | Postgres SSL certificate path                                                     | ""                           |
| SMQ_POSTGRES_SSL_KEY                 | Postgres SSL key                                                                  | ""                           |
| SMQ_POSTGRES_SSL_ROOT_CERT           | Postgres SSL root certificate path                                                | ""                           |
| SMQ_MESSAGE_BROKER_URL               | Message broker instance URL                                                       | nats://localhost:4222        |
| SMQ_JAEGER_URL                       | Jaeger server URL                                                                 | http://jaeger:4318/v1/traces |
| SMQ_SEND_TELEMETRY                   | Send telemetry to supermq call home server                                        | true                         |
| SMQ_POSTGRES_WRITER_INSTANCE_ID      | Service instance ID                                                               | ""                           |

## Deployment

The service itself is distributed as Docker container. Check the [`postgres-writer`](https://github.com/absmach/supermq/blob/main/docker/addons/postgres-writer/docker-compose.yml#L34-L59) service section in docker-compose file to see how service is deployed.

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
SMQ_POSTGRES_WRITER_LOG_LEVEL=[Service log level] \
SMQ_POSTGRES_WRITER_CONFIG_PATH=[Config file path with Message broker subjects list, payload type and content-type] \
SMQ_POSTGRES_WRITER_HTTP_HOST=[Service HTTP host] \
SMQ_POSTGRES_WRITER_HTTP_PORT=[Service HTTP port] \
SMQ_POSTGRES_WRITER_HTTP_SERVER_CERT=[Service HTTP server cert] \
SMQ_POSTGRES_WRITER_HTTP_SERVER_KEY=[Service HTTP server key] \
SMQ_POSTGRES_HOST=[Postgres host] \
SMQ_POSTGRES_PORT=[Postgres port] \
SMQ_POSTGRES_USER=[Postgres user] \
SMQ_POSTGRES_PASS=[Postgres password] \
SMQ_POSTGRES_NAME=[Postgres database name] \
SMQ_POSTGRES_SSL_MODE=[Postgres SSL mode] \
SMQ_POSTGRES_SSL_CERT=[Postgres SSL cert] \
SMQ_POSTGRES_SSL_KEY=[Postgres SSL key] \
SMQ_POSTGRES_SSL_ROOT_CERT=[Postgres SSL Root cert] \
SMQ_MESSAGE_BROKER_URL=[Message broker instance URL] \
SMQ_JAEGER_URL=[Jaeger server URL] \
SMQ_SEND_TELEMETRY=[Send telemetry to supermq call home server] \
SMQ_POSTGRES_WRITER_INSTANCE_ID=[Service instance ID] \

$GOBIN/supermq-postgres-writer
```

## Usage

Starting service will start consuming normalized messages in SenML format.
