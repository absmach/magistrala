# InfluxDB writer

InfluxDB writer provides message repository implementation for InfluxDB.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                  | Description                       | Default               |
|---------------------------|-----------------------------------|-----------------------|
| MF_NATS_URL               | NATS instance URL                 | nats://localhost:4222 |
| MF_INFLUX_WRITER_PORT     | Service HTTP port                 | 8180                  |
| MF_INFLUX_WRITER_DB_NAME  | InfluxDB database name            | mainflux              |
| MF_INFLUX_WRITER_DB_POINT | InfluxDB point to write data to   | messages              |
| MF_INFLUX_WRITER_DB_HOST  | InfluxDB host                     | localhost             |
| MF_INFLUX_WRITER_DB_PORT  | Default port of InfluxDB database | 8086                  |
| MF_INFLUX_WRITER_DB_USER  | Default user of InfluxDB database | mainflux              |
| MF_INFLUX_WRITER_DB_PASS  | Default password of InfluxDB user | mainflux              |

## Deployment

```yaml
  version: "2"
  influxdb-writer:
    image: mainflux/influxdb:[version]
    container_name: [instance name]
    expose:
      - [Service HTTP port]
    restart: on-failure
    environment:
      MF_NATS_URL: [NATS instance URL]
      MF_INFLUX_WRITER_PORT: [Service HTTP port]
      MF_INFLUX_WRITER_DB_NAME: [InfluxDB database name]
      MF_INFLUX_WRITER_DB_POINT: [point name]
      MF_INFLUX_WRITER_DB_HOST: [InfluxDB database host]
      MF_INFLUX_WRITER_DB_PORT: [InfluxDB port]
      MF_INFLUX_WRITER_DB_USER: [InfluxDB admin user]
      MF_INFLUX_WRITER_DB_PASS: [InfluxDB admin password]
    ports:
      - [host machine port]:[configured HTTP port]
```

To start the service, execute the following shell script:

```bash
# download the latest version of the service
go get github.com/mainflux/mainflux


cd $GOPATH/src/github.com/mainflux/mainflux

# compile the influxdb
make influxdb

# copy binary to bin
make install

# Set the environment variables and run the service
MF_NATS_URL=[NATS instance URL] MF_INFLUX_WRITER_PORT=[Service HTTP port] MF_INFLUX_WRITER_DB_NAME=[InfluxDB database name] MF_INFLUX_WRITER_DB_POINT=[point name] MF_INFLUX_WRITER_DB_HOST=[InfluxDB database host] MF_INFLUX_WRITER_DB_PORT=[InfluxDB port] MF_INFLUX_WRITER_DB_USER=[InfluxDB admin user] MF_INFLUX_WRITER_DB_PASS=[InfluxDB admin password] $GOBIN/mainflux-influxdb

```

## Usage

Starting service will start consuming normalized messages in SenML format.

[doc]: http://mainflux.readthedocs.io
