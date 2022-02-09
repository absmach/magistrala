# Cassandra reader

Cassandra reader provides message repository implementation for Cassandra.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                        | Description                                         | Default        |
|---------------------------------|-----------------------------------------------------|----------------|
| MF_CASSANDRA_READER_PORT        | Service HTTP port                                   | 8180           |
| MF_CASSANDRA_READER_DB_CLUSTER  | Cassandra cluster comma separated addresses         | 127.0.0.1      |
| MF_CASSANDRA_READER_DB_USER     | Cassandra DB username                               |                |
| MF_CASSANDRA_READER_DB_PASS     | Cassandra DB password                               |                |
| MF_CASSANDRA_READER_DB_KEYSPACE | Cassandra keyspace name                             | messages       |
| MF_CASSANDRA_READER_DB_PORT     | Cassandra DB port                                   | 9042           |
| MF_CASSANDRA_READER_CLIENT_TLS  | Flag that indicates if TLS should be turned on      | false          |
| MF_CASSANDRA_READER_CA_CERTS    | Path to trusted CAs in PEM format                   |                |
| MF_CASSANDRA_READER_SERVER_CERT | Path to server certificate in pem format            |                |
| MF_CASSANDRA_READER_SERVER_KEY  | Path to server key in pem format                    |                |
| MF_JAEGER_URL                   | Jaeger server URL                                   | localhost:6831 |
| MF_THINGS_AUTH_GRPC_URL         | Things service Auth gRPC URL                        | localhost:8183 |
| MF_THINGS_AUTH_GRPC_TIMEOUT     | Things service Auth gRPC request timeout in seconds | 1              |
| MF_AUTH_GRPC_URL                | Auth service gRPC URL                               | localhost:8181 |
| MF_AUTH_GRPC_TIMEOUT            | Auth service gRPC request timeout in seconds        | 1s             |


## Deployment

The service itself is distributed as Docker container. Check the [`cassandra-reader`](https://github.com/mainflux/mainflux/blob/master/docker/addons/cassandra-reader/docker-compose.yml#L15-L35) service section in 
docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/mainflux/mainflux

cd mainflux

# compile the cassandra
make cassandra-reader

# copy binary to bin
make install

# Set the environment variables and run the service
MF_CASSANDRA_READER_PORT=[Service HTTP port] \
MF_CASSANDRA_READER_DB_CLUSTER=[Cassandra cluster comma separated addresses] \
MF_CASSANDRA_READER_DB_KEYSPACE=[Cassandra keyspace name] \
MF_CASSANDRA_READER_DB_USER=[Cassandra DB username] \
MF_CASSANDRA_READER_DB_PASS=[Cassandra DB password] \
MF_CASSANDRA_READER_DB_PORT=[Cassandra DB port] \
MF_CASSANDRA_READER_CLIENT_TLS=[Flag that indicates if TLS should be turned on] \
MF_CASSANDRA_READER_CA_CERTS=[Path to trusted CAs in PEM format] \
MF_CASSANDRA_READER_SERVER_CERT=[Path to server pem certificate file] \
MF_CASSANDRA_READER_SERVER_KEY=[Path to server pem key file] \
MF_JAEGER_URL=[Jaeger server URL] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout in seconds] \
$GOBIN/mainflux-cassandra-reader

```

### Using docker-compose

This service can be deployed using docker containers. Docker compose file is
available in `<project_root>/docker/addons/cassandra-reader/docker-compose.yml`.
In order to run all Mainflux core services, as well as mentioned optional ones,
execute following command:

```bash
docker-compose -f docker/docker-compose.yml up -d
./docker/addons/cassandra-writer/init.sh
docker-compose -f docker/addons/casandra-reader/docker-compose.yml up -d
```

## Usage

Service exposes [HTTP API](https://api.mainflux.io/?urls.primaryName=readers-openapi.yml) for fetching messages.

[doc]: https://docs.mainflux.io
