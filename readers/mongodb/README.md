# MongoDB reader

MongoDB reader provides message repository implementation for MongoDB.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                         | Description                                         | Default                        |
| -------------------------------- | --------------------------------------------------- | ------------------------------ |
| MG_MONGO_READER_LOG_LEVEL        | Service log level                                   | info                           |
| MG_MONGO_READER_HTTP_HOST        | Service HTTP host                                   | localhost                      |
| MG_MONGO_READER_HTTP_PORT        | Service HTTP port                                   | 9007                           |
| MG_MONGO_READER_HTTP_SERVER_CERT | Service HTTP server cert                            | ""                             |
| MG_MONGO_READER_HTTP_SERVER_KEY  | Service HTTP server key                             | ""                             |
| MG_MONGO_NAME                    | MongoDB database name                               | messages                       |
| MG_MONGO_HOST                    | MongoDB database host                               | localhost                      |
| MG_MONGO_PORT                    | MongoDB database port                               | 27017                          |
| MG_THINGS_AUTH_GRPC_URL          | Things service Auth gRPC URL                        | localhost:7000                 |
| MG_THINGS_AUTH_GRPC_TIMEOUT      | Things service Auth gRPC request timeout in seconds | 1s                             |
| MG_THINGS_AUTH_GRPC_CLIENT_TLS   | Flag that indicates if TLS should be turned on      | false                          |
| MG_THINGS_AUTH_GRPC_CA_CERTS     | Path to trusted CAs in PEM format                   | ""                             |
| MG_AUTH_GRPC_URL                 | Auth service gRPC URL                               | localhost:7001                 |
| MG_AUTH_GRPC_TIMEOUT             | Auth service gRPC request timeout in seconds        | 1s                             |
| MG_AUTH_GRPC_CLIENT_TLS          | Flag that indicates if TLS should be turned on      | false                          |
| MG_AUTH_GRPC_CA_CERT             | Path to trusted CAs in PEM format                   | ""                             |
| MG_JAEGER_URL                    | Jaeger server URL                                   | http://jaeger:14268/api/traces |
| MG_SEND_TELEMETRY                | Send telemetry to magistrala call home server       | true                           |
| MG_MONGO_READER_INSTANCE_ID      | Service instance ID                                 | ""                             |

## Deployment

The service itself is distributed as Docker container. Check the [`mongodb-reader`](https://github.com/absmach/magistrala/blob/master/docker/addons/mongodb-reader/docker-compose.yml#L16-L37) service section in
docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/magistrala

cd magistrala

# compile the mongodb reader
make mongodb-reader

# copy binary to bin
make install

# Set the environment variables and run the service
MG_MONGO_READER_LOG_LEVEL=[Service log level] \
MG_MONGO_READER_HTTP_HOST=[Service HTTP host] \
MG_MONGO_READER_HTTP_PORT=[Service HTTP port] \
MG_MONGO_READER_HTTP_SERVER_CERT=[Path to server pem certificate file] \
MG_MONGO_READER_HTTP_SERVER_KEY=[Path to server pem key file] \
MG_MONGO_NAME=[MongoDB database name] \
MG_MONGO_HOST=[MongoDB database host] \
MG_MONGO_PORT=[MongoDB database port] \
MG_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MG_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout in seconds] \
MG_THINGS_AUTH_GRPC_CLIENT_TLS=[Flag that indicates if TLS should be turned on] \
MG_THINGS_AUTH_GRPC_CA_CERTS=[Path to trusted CAs in PEM format] \
MG_AUTH_GRPC_URL=[Auth service gRPC URL] \
MG_AUTH_GRPC_TIMEOUT=[Auth service gRPC request timeout in seconds] \
MG_AUTH_GRPC_CLIENT_TLS=[Flag that indicates if TLS should be turned on] \
MG_AUTH_GRPC_CA_CERT=[Path to trusted CAs in PEM format] \
MG_JAEGER_URL=[Jaeger server URL] \
MG_SEND_TELEMETRY=[Send telemetry to magistrala call home server] \
MG_MONGO_READER_INSTANCE_ID=[Service instance ID] \
$GOBIN/magistrala-mongodb-reader

```

### Using docker-compose

This service can be deployed using docker containers. Docker compose file is
available in `<project_root>/docker/addons/mongodb-reader/docker-compose.yml`.
In order to run all Magistrala core services, as well as mentioned optional ones,
execute following command:

```bash
docker-compose -f docker/docker-compose.yml up -d
docker-compose -f docker/addons/mongodb-reader/docker-compose.yml up -d
```

## Usage

Service exposes [HTTP API](https://api.mainflux.io/?urls.primaryName=readers-openapi.yml) for fetching messages.

```
Note: MongoDB Reader doesn't support searching substrings from string_value, due to inefficient searching as the current data model is not suitable for this type of queries.
```

[doc]: https://docs.mainflux.io
