# MongoDB writer

MongoDB writer provides message repository implementation for MongoDB.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                     | Description                                                                       | Default                |
| ---------------------------- | --------------------------------------------------------------------------------- | ---------------------- |
| MF_BROKER_URL                | Message broker instance URL                                                       | nats://localhost:4222  |
| MF_MONGO_WRITER_LOG_LEVEL    | Log level for MongoDB writer                                                      | info                  |
| MF_MONGO_WRITER_PORT         | Service HTTP port                                                                 | 8180                   |
| MF_MONGO_WRITER_DB           | Default MongoDB database name                                                     | messages               |
| MF_MONGO_WRITER_DB_HOST      | Default MongoDB database host                                                     | localhost              |
| MF_MONGO_WRITER_DB_PORT      | Default MongoDB database port                                                     | 27017                  |
| MF_MONGO_WRITER_CONFIG_PATH  | Config file path with Message broker subjects list, payload type and content-type | /config.toml           |

## Deployment

The service itself is distributed as Docker container. Check the [`mongodb-writer`](https://github.com/mainflux/mainflux/blob/master/docker/addons/mongodb-writer/docker-compose.yml#L36-L55) service section in docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/mainflux/mainflux

cd mainflux

# compile the mongodb writer
make mongodb-writer

# copy binary to bin
make install

# Set the environment variables and run the service
MF_BROKER_URL=[Message broker instance URL] \
MF_MONGO_WRITER_LOG_LEVEL=[MongoDB writer log level] \
MF_MONGO_WRITER_PORT=[Service HTTP port] \
MF_MONGO_WRITER_DB=[MongoDB database name] \
MF_MONGO_WRITER_DB_HOST=[MongoDB database host] \
MF_MONGO_WRITER_DB_PORT=[MongoDB database port] \
MF_MONGO_WRITER_CONFIG_PATH=[Configuration file path with Message broker subjects list] \
$GOBIN/mainflux-mongodb-writer
```

## Usage

Starting service will start consuming normalized messages in SenML format.
