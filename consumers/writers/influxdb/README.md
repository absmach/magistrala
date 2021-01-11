# InfluxDB writer

InfluxDB writer provides message repository implementation for InfluxDB.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                      | Description                                              | Default                |
| ----------------------------- | -------------------------------------------------------- | ---------------------- |
| MF_NATS_URL                   | NATS instance URL                                        | nats://localhost:4222  |
| MF_INFLUX_WRITER_LOG_LEVEL    | Log level for InfluxDB writer (debug, info, warn, error) | error                  |
| MF_INFLUX_WRITER_PORT         | Service HTTP port                                        | 8180                   |
| MF_INFLUX_WRITER_DB_HOST      | InfluxDB host                                            | localhost              |
| MF_INFLUX_WRITER_DB_PORT      | Default port of InfluxDB database                        | 8086                   |
| MF_INFLUX_WRITER_DB_USER      | Default user of InfluxDB database                        | mainflux               |
| MF_INFLUX_WRITER_DB_PASS      | Default password of InfluxDB user                        | mainflux               |
| MF_INFLUX_WRITER_DB           | InfluxDB database name                                   | messages               |
| MF_INFLUX_WRITER_CONFIG_PATH  | Configuration file path with NATS subjects list          | /configs.toml          |
| MF_INFLUX_WRITER_CONTENT_TYPE | Message payload Content Type                             | application/senml+json |

## Deployment

```yaml
  version: "3.7"
  influxdb-writer:
    image: mainflux/influxdb:[version]
    container_name: [instance name]
    expose:
      - [Service HTTP port]
    restart: on-failure
    environment:
      MF_NATS_URL: [NATS instance URL]
      MF_INFLUX_WRITER_LOG_LEVEL: [Influx writer log level]
      MF_INFLUX_WRITER_PORT: [Service HTTP port]
      MF_INFLUX_WRITER_DB: [InfluxDB name]
      MF_INFLUX_WRITER_DB_HOST: [InfluxDB host]
      MF_INFLUX_WRITER_DB_PORT: [InfluxDB port]
      MF_INFLUX_WRITER_DB_USER: [InfluxDB admin user]
      MF_INFLUX_WRITER_DB_PASS: [InfluxDB admin password]
      MF_INFLUX_WRITER_CONFIG_PATH: [Configuration file path with NATS subjects list]
      MF_INFLUX_WRITER_CONTENT_TYPE: [Message payload Content Type]
    ports:
      - [host machine port]:[configured HTTP port]
    volume:
      - ./subjects.yaml:/config/subjects.yaml
```

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
MF_NATS_URL=[NATS instance URL] MF_INFLUX_WRITER_LOG_LEVEL=[Influx writer log level] MF_INFLUX_WRITER_PORT=[Service HTTP port] MF_INFLUX_WRITER_DB=[InfluxDB database name] MF_INFLUX_WRITER_DB_HOST=[InfluxDB database host] MF_INFLUX_WRITER_DB_PORT=[InfluxDB database port] MF_INFLUX_WRITER_DB_USER=[InfluxDB admin user] MF_INFLUX_WRITER_DB_PASS=[InfluxDB admin password] MF_INFLUX_WRITER_CONFIG_PATH=[Configuration file path with filters list] $GOBIN/mainflux-influxdb
```

### Using docker-compose

This service can be deployed using docker containers.
Docker compose file is available in `<project_root>/docker/addons/influxdb-writer/docker-compose.yml`. Besides database
and writer service, it contains [Grafana platform](https://grafana.com/) which can be used for database
exploration and data visualization and analytics. In order to run Mainflux InfluxDB writer, execute the following command:

```bash
docker-compose -f docker/addons/influxdb-writer/docker-compose.yml up -d
```

_Please note that you need to start core services before the additional ones._

## Usage

Starting service will start consuming normalized messages in SenML format.

[doc]: http://mainflux.readthedocs.io
