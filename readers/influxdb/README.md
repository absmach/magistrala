# InfluxDB reader

InfluxDB reader provides message repository implementation for InfluxDB.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                         | Description                                         | Default                        |
| -------------------------------- | --------------------------------------------------- | ------------------------------ |
| MG_INFLUX_READER_LOG_LEVEL       | Service log level                                   | info                           |
| MG_INFLUX_READER_HTTP_HOST       | Service HTTP host                                   | localhost                      |
| MG_INFLUX_READER_HTTP_PORT       | Service HTTP port                                   | 9005                           |
| MG_INFLUX_READER_SERVER_CERT     | Service HTTP server cert                            | ""                             |
| MG_INFLUX_READER_SERVER_KEY      | Service HTTP server key                             | ""                             |
| MG_INFLUXDB_PROTOCOL             | InfluxDB protocol                                   | http                           |
| MG_INFLUXDB_HOST                 | InfluxDB host name                                  | localhost                      |
| MG_INFLUXDB_PORT                 | Default port of InfluxDB database                   | 8086                           |
| MG_INFLUXDB_ADMIN_USER           | Default user of InfluxDB database                   | magistrala                     |
| MG_INFLUXDB_ADMIN_PASSWORD       | Default password of InfluxDB user                   | magistrala                     |
| MG_INFLUXDB_NAME                 | InfluxDB database name                              | magistrala                     |
| MG_INFLUXDB_BUCKET               | InfluxDB bucket name                                | magistrala-bucket              |
| MG_INFLUXDB_ORG                  | InfluxDB organization name                          | magistrala                     |
| MG_INFLUXDB_TOKEN                | InfluxDB API token                                  | magistrala-token               |
| MG_INFLUXDB_DBURL                | InfluxDB database URL                               | ""                             |
| MG_INFLUXDB_USER_AGENT           | InfluxDB user agent                                 | ""                             |
| MG_INFLUXDB_TIMEOUT              | InfluxDB client connection readiness timeout        | 1s                             |
| MG_INFLUXDB_INSECURE_SKIP_VERIFY | InfluxDB insecure skip verify                       | false                          |
| MG_THINGS_AUTH_GRPC_URL          | Things service Auth gRPC URL                        | localhost:7000                 |
| MG_THINGS_AUTH_GRPC_TIMEOUT      | Things service Auth gRPC request timeout in seconds | 1s                             |
| MG_THINGS_AUTH_GRPC_CLIENT_TLS   | Flag that indicates if TLS should be turned on      | false                          |
| MG_THINGS_AUTH_GRPC_CA_CERTS     | Path to trusted CAs in PEM format                   | ""                             |
| MG_AUTH_GRPC_URL                 | Auth service gRPC URL                               | localhost:7001                 |
| MG_AUTH_GRPC_TIMEOUT             | Auth service gRPC request timeout in seconds        | 1s                             |
| MG_AUTH_GRPC_CLIENT_TLS          | Flag that indicates if TLS should be turned on      | false                          |
| MG_AUTH_GRPC_CA_CERTS            | Path to trusted CAs in PEM format                   | ""                             |
| MG_JAEGER_URL                    | Jaeger server URL                                   | http://jaeger:14268/api/traces |
| MG_SEND_TELEMETRY                | Send telemetry to magistrala call home server       | true                           |
| MG_INFLUX_READER_INSTANCE_ID     | InfluxDB reader instance ID                         |                                |

## Deployment

The service itself is distributed as Docker container. Check the [`influxdb-reader`](https://github.com/absmach/magistrala/blob/master/docker/addons/influxdb-reader/docker-compose.yml#L17-L40) service section in docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/magistrala

cd magistrala

# compile the influxdb-reader
make influxdb-reader

# copy binary to bin
make install

# Set the environment variables and run the service
MG_INFLUX_READER_LOG_LEVEL=[Service log level] \
MG_INFLUX_READER_HTTP_HOST=[Service HTTP host] \
MG_INFLUX_READER_HTTP_PORT=[Service HTTP port] \
MG_INFLUX_READER_HTTP_SERVER_CERT=[Service HTTP server certificate] \
MG_INFLUX_READER_HTTP_SERVER_KEY=[Service HTTP server key] \
MG_INFLUXDB_PROTOCOL=[InfluxDB protocol] \
MG_INFLUXDB_HOST=[InfluxDB database host] \
MG_INFLUXDB_PORT=[InfluxDB database port] \
MG_INFLUXDB_ADMIN_USER=[InfluxDB admin user] \
MG_INFLUXDB_ADMIN_PASSWORD=[InfluxDB admin password] \
MG_INFLUXDB_NAME=[InfluxDB database name] \
MG_INFLUXDB_BUCKET=[InfluxDB bucket] \
MG_INFLUXDB_ORG=[InfluxDB org] \
MG_INFLUXDB_TOKEN=[InfluxDB token] \
MG_INFLUXDB_DBURL=[InfluxDB database URL] \
MG_INFLUXDB_USER_AGENT=[InfluxDB user agent] \
MG_INFLUXDB_TIMEOUT=[InfluxDB timeout] \
MG_INFLUXDB_INSECURE_SKIP_VERIFY=[InfluxDB insecure skip verify] \
MG_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MG_THINGS_AURH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout in seconds] \
MG_THINGS_AUTH_GRPC_CLIENT_TLS=[Flag that indicates if TLS should be turned on] \
MG_THINGS_AUTH_GRPC_CA_CERTS=[Path to trusted CAs in PEM format] \
MG_AUTH_GRPC_URL=[Auth service gRPC URL] \
MG_AUTH_GRPC_TIMEOUT=[Auth service gRPC request timeout in seconds] \
MG_AUTH_GRPC_CLIENT_TLS=[Flag that indicates if TLS should be turned on] \
MG_AUTH_GRPC_CA_CERTS=[Path to trusted CAs in PEM format] \
MG_JAEGER_URL=[Jaeger server URL] \
MG_SEND_TELEMETRY=[Send telemetry to magistrala call home server] \
MG_INFLUX_READER_INSTANCE_ID=[InfluxDB reader instance ID] \
$GOBIN/magistrala-influxdb

```

### Using docker-compose

This service can be deployed using docker containers. Docker compose file is
available in `<project_root>/docker/addons/influxdb-reader/docker-compose.yml`.
In order to run all Magistrala core services, as well as mentioned optional ones,
execute following command:

```bash
docker-compose -f docker/docker-compose.yml up -d
docker-compose -f docker/addons/influxdb-reader/docker-compose.yml up -d
```

And, to use the default .env file, execute the following command:

```bash
docker-compose -f docker/addons/influxdb-reader/docker-compose.yml up --env-file docker/.env -d
```

## Usage

Service exposes [HTTP API](https://api.mainflux.io/?urls.primaryName=readers-openapi.yml) for fetching messages.

Comparator Usage Guide:
| Comparator | Usage | Example |  
|----------------------|-----------------------------------------------------------------------------|------------------------------------|
| eq | Return values that are equal to the query | eq["active"] -> "active" |  
| ge | Return values that are substrings of the query | ge["tiv"] -> "active" and "tiv" |  
| gt | Return values that are substrings of the query and not equal to the query | gt["tiv"] -> "active" |  
| le | Return values that are superstrings of the query | le["active"] -> "tiv" |  
| lt | Return values that are superstrings of the query and not equal to the query | lt["active"] -> "active" and "tiv" |

Official docs can be found [here](https://docs.mainflux.io).
