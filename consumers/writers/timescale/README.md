# Timescale writer

Timescale writer provides message repository implementation for Timescale.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                             | Description                                               | Default                        |
| ------------------------------------ | --------------------------------------------------------- | ------------------------------ |
| MG_TIMESCALE_WRITER_LOG_LEVEL        | Service log level                                         | info                           |
| MG_TIMESCALE_WRITER_CONFIG_PATH      | Configuration file path with Message broker subjects list | /config.toml                   |
| MG_TIMESCALE_WRITER_HTTP_HOST        | Service HTTP host                                         | localhost                      |
| MG_TIMESCALE_WRITER_HTTP_PORT        | Service HTTP port                                         | 9012                           |
| MG_TIMESCALE_WRITER_HTTP_SERVER_CERT | Service HTTP server certificate path                      | ""                             |
| MG_TIMESCALE_WRITER_HTTP_SERVER_KEY  | Service HTTP server key                                   | ""                             |
| MG_TIMESCALE_HOST                    | Timescale DB host                                         | timescale                      |
| MG_TIMESCALE_PORT                    | Timescale DB port                                         | 5432                           |
| MG_TIMESCALE_USER                    | Timescale user                                            | magistrala                     |
| MG_TIMESCALE_PASS                    | Timescale password                                        | magistrala                     |
| MG_TIMESCALE_NAME                    | Timescale database name                                   | messages                       |
| MG_TIMESCALE_SSL_MODE                | Timescale SSL mode                                        | disabled                       |
| MG_TIMESCALE_SSL_CERT                | Timescale SSL certificate path                            | ""                             |
| MG_TIMESCALE_SSL_KEY                 | Timescale SSL key                                         | ""                             |
| MG_TIMESCALE_SSL_ROOT_CERT           | Timescale SSL root certificate path                       | ""                             |
| MG_MESSAGE_BROKER_URL                | Message broker instance URL                               | nats://localhost:4222          |
| MG_JAEGER_URL                        | Jaeger server URL                                         | http://jaeger:14268/api/traces |
| MG_SEND_TELEMETRY                    | Send telemetry to magistrala call home server             | true                           |
| MG_TIMESCALE_WRITER_INSTANCE_ID      | Timescale writer instance ID                              | ""                             |

## Deployment

The service itself is distributed as Docker container. Check the [`timescale-writer`](https://github.com/absmach/magistrala/blob/master/docker/addons/timescale-writer/docker-compose.yml#L34-L59) service section in docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/magistrala

cd magistrala

# compile the timescale writer
make timescale-writer

# copy binary to bin
make install

# Set the environment variables and run the service
MG_TIMESCALE_WRITER_LOG_LEVEL=[Service log level] \
MG_TIMESCALE_WRITER_CONFIG_PATH=[Configuration file path with Message broker subjects list] \
MG_TIMESCALE_WRITER_HTTP_HOST=[Service HTTP host] \
MG_TIMESCALE_WRITER_HTTP_PORT=[Service HTTP port] \
MG_TIMESCALE_WRITER_HTTP_SERVER_CERT=[Service HTTP server cert] \
MG_TIMESCALE_WRITER_HTTP_SERVER_KEY=[Service HTTP server key] \
MG_TIMESCALE_HOST=[Timescale host] \
MG_TIMESCALE_PORT=[Timescale port] \
MG_TIMESCALE_USER=[Timescale user] \
MG_TIMESCALE_PASS=[Timescale password] \
MG_TIMESCALE_NAME=[Timescale database name] \
MG_TIMESCALE_SSL_MODE=[Timescale SSL mode] \
MG_TIMESCALE_SSL_CERT=[Timescale SSL cert] \
MG_TIMESCALE_SSL_KEY=[Timescale SSL key] \
MG_TIMESCALE_SSL_ROOT_CERT=[Timescale SSL Root cert] \
MG_MESSAGE_BROKER_URL=[Message broker instance URL] \
MG_JAEGER_URL=[Jaeger server URL] \
MG_SEND_TELEMETRY=[Send telemetry to magistrala call home server] \
MG_TIMESCALE_WRITER_INSTANCE_ID=[Timescale writer instance ID] \
$GOBIN/magistrala-timescale-writer
```

## Usage

Starting service will start consuming normalized messages in SenML format.
