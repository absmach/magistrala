# MongoDB reader

MongoDB reader provides message repository implementation for MongoDB.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                    | Description                                         | Default        |
|-----------------------------|-----------------------------------------------------|----------------|
| MF_MONGO_READER_PORT        | Service HTTP port                                   | 8180           |
| MF_MONGO_READER_DB          | MongoDB database name                               | messages       |
| MF_MONGO_READER_DB_HOST     | MongoDB database host                               | localhost      |
| MF_MONGO_READER_DB_PORT     | MongoDB database port                               | 27017          |
| MF_MONGO_READER_CLIENT_TLS  | Flag that indicates if TLS should be turned on      | false          |
| MF_MONGO_READER_CA_CERTS    | Path to trusted CAs in PEM format                   |                |
| MF_MONGO_SERVER_CERT        | Path to server certificate in pem format            |                |
| MF_MONGO_SERVER_KEY         | Path to server key in pem format                    |                |
| MF_JAEGER_URL               | Jaeger server URL                                   | localhost:6831 |
| MF_THINGS_AUTH_GRPC_URL     | Things service Auth gRPC URL                        | localhost:8183 |
| MF_THINGS_AUTH_GRPC_TIMEOUT | Things service Auth gRPC request timeout in seconds | 1s             |
| MF_AUTH_GRPC_URL            | Auth service gRPC URL                               | localhost:8181 |
| MF_AUTH_GRPC_TIMEOUT        | Auth service gRPC request timeout in seconds        | 1s             |


## Deployment

The service itself is distributed as Docker container. Check the [`mongodb-reader`](https://github.com/mainflux/mainflux/blob/master/docker/addons/mongodb-reader/docker-compose.yml#L16-L37) service section in 
docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/mainflux/mainflux

cd mainflux

# compile the mongodb reader
make mongodb-reader

# copy binary to bin
make install

# Set the environment variables and run the service
MF_MONGO_READER_PORT=[Service HTTP port] \
MF_MONGO_READER_DB=[MongoDB database name] \
MF_MONGO_READER_DB_HOST=[MongoDB database host] \
MF_MONGO_READER_DB_PORT=[MongoDB database port] \
MF_MONGO_READER_CLIENT_TLS=[Flag that indicates if TLS should be turned on] \
MF_MONGO_READER_CA_CERTS=[Path to trusted CAs in PEM format] \
MF_JAEGER_URL=[Jaeger server URL] \
MF_MONGO_READER_SERVER_CERT=[Path to server pem certificate file] \
MF_MONGO_READER_SERVER_KEY=[Path to server pem key file] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout in seconds] \
$GOBIN/mainflux-mongodb-reader

```

### Using docker-compose

This service can be deployed using docker containers. Docker compose file is
available in `<project_root>/docker/addons/mongodb-reader/docker-compose.yml`.
In order to run all Mainflux core services, as well as mentioned optional ones,
execute following command:

```bash
docker-compose -f docker/docker-compose.yml up -d
docker-compose -f docker/addons/mongodb-reader/docker-compose.yml up -d
```

## Usage

Service exposes [HTTP API](https://api.mainflux.io/?urls.primaryName=readers-openapi.yml) for fetching messages.

[doc]: https://docs.mainflux.io
