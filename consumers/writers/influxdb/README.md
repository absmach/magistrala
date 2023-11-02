# InfluxDB writer

InfluxDB writer provides message repository implementation for InfluxDB.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                          | Description                                                                       | Default                        |
| --------------------------------- | --------------------------------------------------------------------------------- | ------------------------------ |
| MG_INFLUX_WRITER_LOG_LEVEL        | Log level for InfluxDB writer (debug, info, warn, error)                          | info                           |
| MG_INFLUX_WRITER_CONFIG_PATH      | Config file path with message broker subjects list, payload type and content-type | /configs.toml                  |
| MG_INFLUX_WRITER_HTTP_HOST        | Service HTTP host                                                                 |                                |
| MG_INFLUX_WRITER_HTTP_PORT        | Service HTTP port                                                                 | 9006                           |
| MG_INFLUX_WRITER_HTTP_SERVER_CERT | Path to server certificate in pem format                                          |                                |
| MG_INFLUX_WRITER_HTTP_SERVER_KEY  | Path to server key in pem format                                                  |                                |
| MG_INFLUXDB_PROTOCOL              | InfluxDB protocol                                                                 | http                           |
| MG_INFLUXDB_HOST                  | InfluxDB host name                                                                | magistrala-influxdb            |
| MG_INFLUXDB_PORT                  | Default port of InfluxDB database                                                 | 8086                           |
| MG_INFLUXDB_ADMIN_USER            | Default user of InfluxDB database                                                 | magistrala                     |
| MG_INFLUXDB_ADMIN_PASSWORD        | Default password of InfluxDB user                                                 | magistrala                     |
| MG_INFLUXDB_NAME                  | InfluxDB database name                                                            | magistrala                     |
| MG_INFLUXDB_BUCKET                | InfluxDB bucket name                                                              | magistrala-bucket              |
| MG_INFLUXDB_ORG                   | InfluxDB organization name                                                        | magistrala                     |
| MG_INFLUXDB_TOKEN                 | InfluxDB API token                                                                | magistrala-token               |
| MG_INFLUXDB_DBURL                 | InfluxDB database URL                                                             |                                |
| MG_INFLUXDB_USER_AGENT            | InfluxDB user agent                                                               |                                |
| MG_INFLUXDB_TIMEOUT               | InfluxDB client connection readiness timeout                                      | 1s                             |
| MG_INFLUXDB_INSECURE_SKIP_VERIFY  | InfluxDB client connection insecure skip verify                                   | false                          |
| MG_MESSAGE_BROKER_URL             | Message broker instance URL                                                       | nats://localhost:4222          |
| MG_JAEGER_URL                     | Jaeger server URL                                                                 | http://jaeger:14268/api/traces |
| MG_SEND_TELEMETRY                 | Send telemetry to magistrala call home server                                     | true                           |
| MG_INFLUX_WRITER_INSTANCE_ID      | InfluxDB writer instance ID                                                       |                                |

## Deployment

The service itself is distributed as Docker container. Check the [`influxdb-writer`](https://github.com/absmach/magistrala/blob/master/docker/addons/influxdb-writer/docker-compose.yml#L35-L58) service section in docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/magistrala

cd magistrala

# compile the influxdb
make influxdb

# copy binary to bin
make install

# Set the environment variables and run the service
MG_INFLUX_WRITER_LOG_LEVEL=[Influx writer log level] \
MG_INFLUX_WRITER_CONFIG_PATH=[Config file path with Message broker subjects list, payload type and content-type] \
MG_INFLUX_WRITER_HTTP_HOST=[Service HTTP host] \
MG_INFLUX_WRITER_HTTP_PORT=[Service HTTP port] \
MG_INFLUX_WRITER_HTTP_SERVER_CERT=[Service HTTP server cert] \
MG_INFLUX_WRITER_HTTP_SERVER_KEY=[Service HTTP server key] \
MG_INFLUXDB_PROTOCOL=[InfluxDB protocol] \
MG_INFLUXDB_HOST=[InfluxDB database host] \
MG_INFLUXDB_PORT=[InfluxDB database port] \
MG_INFLUXDB_ADMIN_USER=[InfluxDB admin user] \
MG_INFLUXDB_ADMIN_PASSWORD=[InfluxDB admin password] \
MG_INFLUXDB_NAME=[InfluxDB database name] \
MG_INFLUXDB_BUCKET=[InfluxDB bucket] \
MG_INFLUXDB_ORG=[InfluxDB org] \
MG_INFLUXDB_TOKEN=[InfluxDB token] \
MG_INFLUXDB_DBURL=[InfluxDB database url] \
MG_INFLUXDB_USER_AGENT=[InfluxDB user agent] \
MG_INFLUXDB_TIMEOUT=[InfluxDB timeout] \
MG_INFLUXDB_INSECURE_SKIP_VERIFY=[InfluxDB insecure skip verify] \
MG_MESSAGE_BROKER_URL=[Message broker instance URL] \
MG_JAEGER_URL=[Jaeger server URL] \
MG_SEND_TELEMETRY=[Send telemetry to magistrala call home server] \
MG_INFLUX_WRITER_INSTANCE_ID=[Influx writer instance ID] \
$GOBIN/magistrala-influxdb
```

### Using docker-compose

This service can be deployed using docker containers.
Docker compose file is available in `<project_root>/docker/addons/influxdb-writer/docker-compose.yml`. Besides database
and writer service, it contains InfluxData Web Admin Interface which can be used for database
exploration and data visualization and analytics. In order to run Magistrala InfluxDB writer, execute the following command:

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
