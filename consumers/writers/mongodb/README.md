# MongoDB writer

MongoDB writer provides message repository implementation for MongoDB.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                         | Description                                                                       | Default                        |
| -------------------------------- | --------------------------------------------------------------------------------- | ------------------------------ |
| MG_MONGO_WRITER_LOG_LEVEL        | Log level for MongoDB writer                                                      | info                           |
| MG_MONGO_WRITER_CONFIG_PATH      | Config file path with Message broker subjects list, payload type and content-type | /config.toml                   |
| MG_MONGO_WRITER_HTTP_HOST        | Service HTTP host                                                                 | localhost                      |
| MG_MONGO_WRITER_HTTP_PORT        | Service HTTP port                                                                 | 9010                           |
| MG_MONGO_WRITER_HTTP_SERVER_CERT | Service HTTP server certificate path                                              | ""                             |
| MG_MONGO_WRITER_HTTP_SERVER_KEY  | Service HTTP server key                                                           | ""                             |
| MG_MONGO_NAME                    | Default MongoDB database name                                                     | messages                       |
| MG_MONGO_HOST                    | Default MongoDB database host                                                     | localhost                      |
| MG_MONGO_PORT                    | Default MongoDB database port                                                     | 27017                          |
| MG_MESSAGE_BROKER_URL            | Message broker instance URL                                                       | nats://localhost:4222          |
| MG_JAEGER_URL                    | Jaeger server URL                                                                 | http://jaeger:14268/api/traces |
| MG_SEND_TELEMETRY                | Send telemetry to magistrala call home server                                     | true                           |
| MG_MONGO_WRITER_INSTANCE_ID      | MongoDB writer instance ID                                                        | ""                             |

## Deployment

The service itself is distributed as Docker container. Check the [`mongodb-writer`](https://github.com/absmach/magistrala/blob/master/docker/addons/mongodb-writer/docker-compose.yml#L36-L55) service section in docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/magistrala

cd magistrala

# compile the mongodb writer
make mongodb-writer

# copy binary to bin
make install

# Set the environment variables and run the service
MG_MONGO_WRITER_LOG_LEVEL=[MongoDB writer log level] \
MG_MONGO_WRITER_CONFIG_PATH=[Configuration file path with Message broker subjects list] \
MG_MONGO_WRITER_HTTP_HOST=[Service HTTP host] \
MG_MONGO_WRITER_HTTP_PORT=[Service HTTP port] \
MG_MONGO_WRITER_HTTP_SERVER_CERT=[Service HTTP server certificate] \
MG_MONGO_WRITER_HTTP_SERVER_KEY=[Service HTTP server key] \
MG_MONGO_NAME=[MongoDB database name] \
MG_MONGO_HOST=[MongoDB database host] \
MG_MONGO_PORT=[MongoDB database port] \
MG_MESSAGE_BROKER_URL=[Message broker instance URL] \
MG_JAEGER_URL=[Jaeger server URL] \
MG_SEND_TELEMETRY=[Send telemetry to magistrala call home server] \
MG_MONGO_WRITER_INSTANCE_ID=[MongoDB writer instance ID] \

$GOBIN/magistrala-mongodb-writer
```

## Usage

Starting service will start consuming normalized messages in SenML format.
