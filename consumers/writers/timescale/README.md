# Timescale writer

Timescale writer provides message repository implementation for Timescale.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                              | Description                                               | Default                      |
| ------------------------------------- | --------------------------------------------------------- | ---------------------------- |
| SMQ_TIMESCALE_WRITER_LOG_LEVEL        | Service log level                                         | info                         |
| SMQ_TIMESCALE_WRITER_CONFIG_PATH      | Configuration file path with Message broker subjects list | /config.toml                 |
| SMQ_TIMESCALE_WRITER_HTTP_HOST        | Service HTTP host                                         | localhost                    |
| SMQ_TIMESCALE_WRITER_HTTP_PORT        | Service HTTP port                                         | 9012                         |
| SMQ_TIMESCALE_WRITER_HTTP_SERVER_CERT | Service HTTP server certificate path                      | ""                           |
| SMQ_TIMESCALE_WRITER_HTTP_SERVER_KEY  | Service HTTP server key                                   | ""                           |
| SMQ_TIMESCALE_HOST                    | Timescale DB host                                         | timescale                    |
| SMQ_TIMESCALE_PORT                    | Timescale DB port                                         | 5432                         |
| SMQ_TIMESCALE_USER                    | Timescale user                                            | supermq                      |
| SMQ_TIMESCALE_PASS                    | Timescale password                                        | supermq                      |
| SMQ_TIMESCALE_NAME                    | Timescale database name                                   | messages                     |
| SMQ_TIMESCALE_SSL_MODE                | Timescale SSL mode                                        | disabled                     |
| SMQ_TIMESCALE_SSL_CERT                | Timescale SSL certificate path                            | ""                           |
| SMQ_TIMESCALE_SSL_KEY                 | Timescale SSL key                                         | ""                           |
| SMQ_TIMESCALE_SSL_ROOT_CERT           | Timescale SSL root certificate path                       | ""                           |
| SMQ_MESSAGE_BROKER_URL                | Message broker instance URL                               | nats://localhost:4222        |
| SMQ_JAEGER_URL                        | Jaeger server URL                                         | http://jaeger:4318/v1/traces |
| SMQ_SEND_TELEMETRY                    | Send telemetry to supermq call home server                | true                         |
| SMQ_TIMESCALE_WRITER_INSTANCE_ID      | Timescale writer instance ID                              | ""                           |

## Deployment

The service itself is distributed as Docker container. Check the [`timescale-writer`](https://github.com/absmach/supermq/blob/main/docker/addons/timescale-writer/docker-compose.yml#L34-L59) service section in docker-compose file to see how service is deployed.

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
SMQ_TIMESCALE_WRITER_LOG_LEVEL=[Service log level] \
SMQ_TIMESCALE_WRITER_CONFIG_PATH=[Configuration file path with Message broker subjects list] \
SMQ_TIMESCALE_WRITER_HTTP_HOST=[Service HTTP host] \
SMQ_TIMESCALE_WRITER_HTTP_PORT=[Service HTTP port] \
SMQ_TIMESCALE_WRITER_HTTP_SERVER_CERT=[Service HTTP server cert] \
SMQ_TIMESCALE_WRITER_HTTP_SERVER_KEY=[Service HTTP server key] \
SMQ_TIMESCALE_HOST=[Timescale host] \
SMQ_TIMESCALE_PORT=[Timescale port] \
SMQ_TIMESCALE_USER=[Timescale user] \
SMQ_TIMESCALE_PASS=[Timescale password] \
SMQ_TIMESCALE_NAME=[Timescale database name] \
SMQ_TIMESCALE_SSL_MODE=[Timescale SSL mode] \
SMQ_TIMESCALE_SSL_CERT=[Timescale SSL cert] \
SMQ_TIMESCALE_SSL_KEY=[Timescale SSL key] \
SMQ_TIMESCALE_SSL_ROOT_CERT=[Timescale SSL Root cert] \
SMQ_MESSAGE_BROKER_URL=[Message broker instance URL] \
SMQ_JAEGER_URL=[Jaeger server URL] \
SMQ_SEND_TELEMETRY=[Send telemetry to supermq call home server] \
SMQ_TIMESCALE_WRITER_INSTANCE_ID=[Timescale writer instance ID] \
$GOBIN/supermq-timescale-writer
```

## Usage

Starting service will start consuming normalized messages in SenML format.
