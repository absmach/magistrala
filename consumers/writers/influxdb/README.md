# InfluxDB writer

InfluxDB writer provides message repository implementation for InfluxDB.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                          | Description                                                                       | Default                        |
| --------------------------------- | --------------------------------------------------------------------------------- | ------------------------------ |
| MF_INFLUX_WRITER_LOG_LEVEL        | Log level for InfluxDB writer (debug, info, warn, error)                          | info                           |
| MF_INFLUX_WRITER_CONFIG_PATH      | Config file path with message broker subjects list, payload type and content-type | /configs.toml                  |
| MF_INFLUX_WRITER_HTTP_HOST        | Service HTTP host                                                                 |                                |
| MF_INFLUX_WRITER_HTTP_PORT        | Service HTTP port                                                                 | 9006                           |
| MF_INFLUX_WRITER_HTTP_SERVER_CERT | Path to server certificate in pem format                                          |                                |
| MF_INFLUX_WRITER_HTTP_SERVER_KEY  | Path to server key in pem format                                                  |                                |
| MF_INFLUXDB_PROTOCOL              | InfluxDB protocol                                                                 | http                           |
| MF_INFLUXDB_HOST                  | InfluxDB host name                                                                | mainflux-influxdb              |
| MF_INFLUXDB_PORT                  | Default port of InfluxDB database                                                 | 8086                           |
| MF_INFLUXDB_ADMIN_USER            | Default user of InfluxDB database                                                 | mainflux                       |
| MF_INFLUXDB_ADMIN_PASSWORD        | Default password of InfluxDB user                                                 | mainflux                       |
| MF_INFLUXDB_NAME                  | InfluxDB database name                                                            | mainflux                       |
| MF_INFLUXDB_BUCKET                | InfluxDB bucket name                                                              | mainflux-bucket                |
| MF_INFLUXDB_ORG                   | InfluxDB organization name                                                        | mainflux                       |
| MF_INFLUXDB_TOKEN                 | InfluxDB API token                                                                | mainflux-token                 |
| MF_INFLUXDB_DBURL                 | InfluxDB database URL                                                             |                                |
| MF_INFLUXDB_USER_AGENT            | InfluxDB user agent                                                               |                                |
| MF_INFLUXDB_TIMEOUT               | InfluxDB client connection readiness timeout                                      | 1s                             |
| MF_INFLUXDB_INSECURE_SKIP_VERIFY  | InfluxDB client connection insecure skip verify                                   | false                          |
| MF_MESSAGE_BROKER_URL             | Message broker instance URL                                                       | nats://localhost:4222          |
| MF_JAEGER_URL                     | Jaeger server URL                                                                 | http://jaeger:14268/api/traces |
| MF_SEND_TELEMETRY                 | Send telemetry to mainflux call home server                                       | true                           |
| MF_INFLUX_WRITER_INSTANCE_ID      | InfluxDB writer instance ID                                                       |                                |

## Deployment

The service itself is distributed as Docker container. Check the [`influxdb-writer`](https://github.com/mainflux/mainflux/blob/master/docker/addons/influxdb-writer/docker-compose.yml#L35-L58) service section in docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/mainflux/mainflux

cd mainflux

# compile the influxdb
make influxdb

# copy binary to bin
make install

# Set the environment variables and run the service
MF_INFLUX_WRITER_LOG_LEVEL=[Influx writer log level] \
MF_INFLUX_WRITER_CONFIG_PATH=[Config file path with Message broker subjects list, payload type and content-type] \
MF_INFLUX_WRITER_HTTP_HOST=[Service HTTP host] \
MF_INFLUX_WRITER_HTTP_PORT=[Service HTTP port] \
MF_INFLUX_WRITER_HTTP_SERVER_CERT=[Service HTTP server cert] \
MF_INFLUX_WRITER_HTTP_SERVER_KEY=[Service HTTP server key] \
MF_INFLUXDB_PROTOCOL=[InfluxDB protocol] \
MF_INFLUXDB_HOST=[InfluxDB database host] \
MF_INFLUXDB_PORT=[InfluxDB database port] \
MF_INFLUXDB_ADMIN_USER=[InfluxDB admin user] \
MF_INFLUXDB_ADMIN_PASSWORD=[InfluxDB admin password] \
MF_INFLUXDB_NAME=[InfluxDB database name] \
MF_INFLUXDB_BUCKET=[InfluxDB bucket] \
MF_INFLUXDB_ORG=[InfluxDB org] \
MF_INFLUXDB_TOKEN=[InfluxDB token] \
MF_INFLUXDB_DBURL=[InfluxDB database url] \
MF_INFLUXDB_USER_AGENT=[InfluxDB user agent] \
MF_INFLUXDB_TIMEOUT=[InfluxDB timeout] \
MF_INFLUXDB_INSECURE_SKIP_VERIFY=[InfluxDB insecure skip verify] \
MF_MESSAGE_BROKER_URL=[Message broker instance URL] \
MF_JAEGER_URL=[Jaeger server URL] \
MF_SEND_TELEMETRY=[Send telemetry to mainflux call home server] \
MF_INFLUX_WRITER_INSTANCE_ID=[Influx writer instance ID] \
$GOBIN/mainflux-influxdb
```

### Using docker-compose

This service can be deployed using docker containers.
Docker compose file is available in `<project_root>/docker/addons/influxdb-writer/docker-compose.yml`. Besides database
and writer service, it contains InfluxData Web Admin Interface which can be used for database
exploration and data visualization and analytics. In order to run Mainflux InfluxDB writer, execute the following command:

```bash
docker-compose -f docker/addons/influxdb-writer/docker-compose.yml up -d
```

And, to use the default .env file, execute the following command:

```bash
docker-compose -f docker/addons/influxdb-writer/docker-compose.yml up --env-file docker/.env -d
```

_Please note that you need to start core services before the additional ones._

## Usage

Starting service will start consuming normalized messages in SenML format.

Official docs can be found [here](https://docs.mainflux.io).
