# MongoDB writer

MongoDB writer provides message repository implementation for MongoDB.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                        | Description                                | Default               |
|---------------------------------|--------------------------------------------|-----------------------|
| MF_NATS_URL                     | NATS instance URL                          | nats://localhost:4222 |
| MF_MONGO_WRITER_LOG_LEVEL       | Log level for MongoDB writer               | error                 |
| MF_MONGO_WRITER_PORT            | Service HTTP port                          | 8180                  |
| MF_MONGO_WRITER_DB_NAME         | Default MongoDB database name              | mainflux              |
| MF_MONGO_WRITER_DB_HOST         | Default MongoDB database host              | localhost             |
| MF_MONGO_WRITER_DB_PORT         | Default MongoDB database port              | 27017                 |
| MF_MONGO_WRITER_CHANNELS_CONFIG | Configuration file path with channels list | /config/channels.toml |

## Deployment

```yaml
  mongodb-writer:
    image: mainflux/mongodb-writer:[version]
    container_name: [instance name]
    depends_on:
      - mongodb
      - nats
    expose:
      - [Service HTTP port]
    restart: on-failure
    environment:
      MF_NATS_URL: [NATS instance URL]
      MF_MONGO_WRITER_LOG_LEVEL: [MongoDB writer log level]
      MF_MONGO_WRITER_PORT: [Service HTTP port]
      MF_MONGO_WRITER_DB_NAME: [MongoDB name]
      MF_MONGO_WRITER_DB_HOST: [MongoDB host]
      MF_MONGO_WRITER_DB_PORT: [MongoDB port]
      MF_MONGO_WRITER_CHANNELS_CONFIG: [Configuration file path with channels list]
    ports:
      - [host machine port]:[configured HTTP port]
    volume:
      - ./channels.yaml:/config/channels.yaml
```

To start the service, execute the following shell script:

```bash
# download the latest version of the service
go get github.com/mainflux/mainflux


cd $GOPATH/src/github.com/mainflux/mainflux

# compile the mongodb writer
make mongodb-writer

# copy binary to bin
make install

# Set the environment variables and run the service
MF_NATS_URL=[NATS instance URL] MF_MONGO_WRITER_LOG_LEVEL=[MongoDB writer log level] MF_MONGO_WRITER_PORT=[Service HTTP port] MF_MONGO_WRITER_DB_NAME=[MongoDB database name] MF_MONGO_WRITER_DB_HOST=[MongoDB database host] MF_MONGO_WRITER_DB_PORT=[MongoDB database port] MF_MONGO_WRITER_CHANNELS_CONFIG=[Configuration file path with channels list] $GOBIN/mainflux-mongodb-writer
```

## Usage

Starting service will start consuming normalized messages in SenML format.
