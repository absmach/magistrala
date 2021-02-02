# InfluxDB reader

InfluxDB reader provides message repository implementation for InfluxDB.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                     | Description                                         | Default        |
|------------------------------|-----------------------------------------------------|----------------|
| MF_INFLUX_READER_PORT        | Service HTTP port                                   | 8180           |
| MF_INFLUX_READER_DB_HOST     | InfluxDB host                                       | localhost      |
| MF_INFLUX_READER_DB_PORT     | Default port of InfluxDB database                   | 8086           |
| MF_INFLUX_READER_DB_USER     | Default user of InfluxDB database                   | mainflux       |
| MF_INFLUX_READER_DB_PASS     | Default password of InfluxDB user                   | mainflux       |
| MF_INFLUX_READER_DB          | InfluxDB database name                              | messages       |
| MF_INFLUX_READER_CLIENT_TLS  | Flag that indicates if TLS should be turned on      | false          |
| MF_INFLUX_READER_CA_CERTS    | Path to trusted CAs in PEM format                   |                |
| MF_INFLUX_READER_SERVER_CERT | Path to server certificate in pem format            |                |
| MF_INFLUX_READER_SERVER_KEY  | Path to server key in pem format                    |                |
| MF_JAEGER_URL                | Jaeger server URL                                   | localhost:6831 |
| MF_THINGS_AUTH_GRPC_URL      | Things service Auth gRPC URL                        | localhost:8181 |
| MF_THINGS_AUTH_GRPC_TIMEOUT  | Things service Auth gRPC request timeout in seconds | 1s             |

## Deployment

The service itself is distributed as Docker container. Check the [`influxdb-reader`](https://github.com/mainflux/mainflux/blob/master/docker/addons/influxdb-reader/docker-compose.yml#L17-L40) service section in 
docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/mainflux/mainflux

cd mainflux

# compile the influxdb-reader
make influxdb-reader

# copy binary to bin
make install

# Set the environment variables and run the service
MF_INFLUX_READER_PORT=[Service HTTP port] \
MF_INFLUX_READER_DB=[InfluxDB database name] \
MF_INFLUX_READER_DB_HOST=[InfluxDB database host] \
MF_INFLUX_READER_DB_PORT=[InfluxDB database port] \
MF_INFLUX_READER_DB_USER=[InfluxDB admin user] \
MF_INFLUX_READER_DB_PASS=[InfluxDB admin password] \
MF_INFLUX_READER_CLIENT_TLS=[Flag that indicates if TLS should be turned on] \
MF_INFLUX_READER_CA_CERTS=[Path to trusted CAs in PEM format] \
MF_INFLUX_READER_SERVER_CERT=[Path to server pem certificate file] \
MF_INFLUX_READER_SERVER_KEY=[Path to server pem key file] \
MF_JAEGER_URL=[Jaeger server URL] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MF_THINGS_AURH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout in seconds] \
$GOBIN/mainflux-influxdb

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

[doc]: ../openapi.yml
