# MongoDB reader

MongoDB reader provides message repository implementation for MongoDB.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                       | Description                                    | Default        |
|--------------------------------|------------------------------------------------|----------------|
| MF_THINGS_URL                  | Things service URL                             | localhost:8181 |
| MF_MONGO_READER_PORT           | Service HTTP port                              | 8180           |
| MF_MONGO_READER_DB_NAME        | MongoDB database name                          | mainflux       |
| MF_MONGO_READER_DB_HOST        | MongoDB database host                          | localhost      |
| MF_MONGO_READER_DB_PORT        | MongoDB database port                          | 27017          |
| MF_MONGO_READER_CLIENT_TLS     | Flag that indicates if TLS should be turned on | false          |
| MF_MONGO_READER_CA_CERTS       | Path to trusted CAs in PEM format              |                |
| MF_JAEGER_URL                  | Jaeger server URL                              | localhost:6831 |
| MF_MONGO_READER_THINGS_TIMEOUT | Things gRPC request timeout in seconds         | 1              |

## Deployment

```yaml
  version: "2"
  mongodb-reader:
    image: mainflux/mongodb-reader:[version]
    container_name: [instance name]
    expose:
      - [Service HTTP port]
    restart: on-failure
    environment:
        MF_THINGS_URL: [Things service URL]
        MF_MONGO_READER_PORT: [Service HTTP port]
        MF_MONGO_READER_DB_NAME: [MongoDB name]
        MF_MONGO_READER_DB_HOST: [MongoDB host]
        MF_MONGO_READER_DB_PORT: [MongoDB port]
        MF_MONGO_READER_CLIENT_TLS: [Flag that indicates if TLS should be turned on]
        MF_MONGO_READER_CA_CERTS: [Path to trusted CAs in PEM format]
        MF_JAEGER_URL: [Jaeger server URL]
        MF_MONGO_READER_THINGS_TIMEOUT: [Things gRPC request timeout in seconds]
    ports:
      - [host machine port]:[configured HTTP port]
```

To start the service, execute the following shell script:

```bash
# download the latest version of the service
go get github.com/mainflux/mainflux

cd $GOPATH/src/github.com/mainflux/mainflux

# compile the mongodb reader
make mongodb-reader

# copy binary to bin
make install

# Set the environment variables and run the service
MF_THINGS_URL=[Things service URL] MF_MONGO_READER_PORT=[Service HTTP port] MF_MONGO_READER_DB_NAME=[MongoDB database name] MF_MONGO_READER_DB_HOST=[MongoDB database host] MF_MONGO_READER_DB_PORT=[MongoDB database port] MF_MONGO_READER_CLIENT_TLS=[Flag that indicates if TLS should be turned on] MF_MONGO_READER_CA_CERTS=[Path to trusted CAs in PEM format] MF_JAEGER_URL=[Jaeger server URL] MF_MONGO_READER_THINGS_TIMEOUT=[Things gRPC request timeout in seconds] $GOBIN/mainflux-mongodb-reader

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

Service exposes [HTTP API][doc] for fetching messages.

[doc]: ../swagger.yml
