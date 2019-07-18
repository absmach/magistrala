# InfluxDB reader

InfluxDB reader provides message repository implementation for InfluxDB.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                        | Description                                    | Default        |
|---------------------------------|------------------------------------------------|----------------|
| MF_INFLUX_READER_PORT           | Service HTTP port                              | 8180           |
| MF_INFLUX_READER_DB_NAME        | InfluxDB database name                         | mainflux       |
| MF_INFLUX_READER_DB_HOST        | InfluxDB host                                  | localhost      |
| MF_INFLUX_READER_DB_PORT        | Default port of InfluxDB database              | 8086           |
| MF_INFLUX_READER_DB_USER        | Default user of InfluxDB database              | mainflux       |
| MF_INFLUX_READER_DB_PASS        | Default password of InfluxDB user              | mainflux       |
| MF_INFLUX_READER_CLIENT_TLS     | Flag that indicates if TLS should be turned on | false          |
| MF_INFLUX_READER_CA_CERTS       | Path to trusted CAs in PEM format              |                |
| MF_JAEGER_URL                   | Jaeger server URL                              | localhost:6831 |
| MF_INFLUX_READER_THINGS_TIMEOUT | Things gRPC request timeout in seconds         | 1              |

## Deployment

```yaml
  version: "2"
  influxdb-reader:
    image: mainflux/influxdb-reader:[version]
    container_name: [instance name]
    restart: on-failure
    environment:
      MF_THINGS_URL: [Things service URL]
      MF_INFLUX_READER_PORT: [Service HTTP port]
      MF_INFLUX_READER_DB_NAME: [InfluxDB name]
      MF_INFLUX_READER_DB_HOST: [InfluxDB host]
      MF_INFLUX_READER_DB_PORT: [InfluxDB port]
      MF_INFLUX_READER_DB_USER: [InfluxDB admin user]
      MF_INFLUX_READER_DB_PASS: [InfluxDB admin password]
      MF_INFLUX_READER_CLIENT_TLS: [Flag that indicates if TLS should be turned on]
      MF_INFLUX_READER_CA_CERTS: [Path to trusted CAs in PEM format]
      MF_JAEGER_URL: [Jaeger server URL]
      MF_INFLUX_READER_THINGS_TIMEOUT: [Things gRPC request timeout in seconds]
    ports:
      - [host machine port]:[configured HTTP port]
```

To start the service, execute the following shell script:

```bash
# download the latest version of the service
go get github.com/mainflux/mainflux

cd $GOPATH/src/github.com/mainflux/mainflux

# compile the influxdb-reader
make influxdb-reader

# copy binary to bin
make install

# Set the environment variables and run the service
MF_THINGS_URL=[Things service URL] MF_INFLUX_READER_PORT=[Service HTTP port] MF_INFLUX_READER_DB_NAME=[InfluxDB database name] MF_INFLUX_READER_DB_HOST=[InfluxDB database host] MF_INFLUX_READER_DB_PORT=[InfluxDB database port] MF_INFLUX_READER_DB_USER=[InfluxDB admin user] MF_INFLUX_READER_DB_PASS=[InfluxDB admin password] MF_INFLUX_READER_CLIENT_TLS=[Flag that indicates if TLS should be turned on] MF_INFLUX_READER_CA_CERTS=[Path to trusted CAs in PEM format] MF_JAEGER_URL=[Jaeger server URL] MF_INFLUX_READER_THINGS_TIMEOUT=[Things gRPC request timeout in seconds] $GOBIN/mainflux-influxdb

```

### Using docker-compose

This service can be deployed using docker containers. Docker compose file is
available in `<project_root>/docker/addons/influxdb-reader/docker-compose.yml`.
In order to run all Mainflux core services, as well as mentioned optional ones,
execute following command:

```bash
docker-compose -f docker/docker-compose.yml up -d
docker-compose -f docker/addons/influxdb-reader/docker-compose.yml up -d
```

## Usage

Service exposes [HTTP API][doc] for fetching messages.

[doc]: ../swagger.yml
