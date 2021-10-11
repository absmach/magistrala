# Twins

Service twins is used for CRUD and update of digital twins. Twin is a semantic
representation of a real world data system consisting of data producers and
consumers. It stores the sequence of attribute based definitions of a system and
refers to a time series of definition based states that store the system
historical data.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                   | Description                                                          | Default               |
|----------------------------|----------------------------------------------------------------------|-----------------------|
| MF_TWINS_LOG_LEVEL         | Log level for twin service (debug, info, warn, error)                | error                 |
| MF_TWINS_HTTP_PORT         | Twins service HTTP port                                              | 9021                  |
| MF_TWINS_SERVER_CERT       | Path to server certificate in PEM format                             |                       |
| MF_TWINS_SERVER_KEY        | Path to server key in PEM format                                     |                       |
| MF_JAEGER_URL              | Jaeger server URL                                                    |                       |
| MF_TWINS_DB                | Database name                                                        | mainflux              |
| MF_TWINS_DB_HOST           | Database host address                                                | localhost             |
| MF_TWINS_DB_PORT           | Database host port                                                   | 27017                 |
| MF_THINGS_STANDALONE_EMAIL | User email for standalone mode (no gRPC communication with users)       |                |
| MF_THINGS_STANDALONE_TOKEN | User token for standalone mode that should be passed in auth header     |                |
| MF_TWINS_CLIENT_TLS        | Flag that indicates if TLS should be turned on                       | false                 |
| MF_TWINS_CA_CERTS          | Path to trusted CAs in PEM format                                    |                       |
| MF_TWINS_CHANNEL_ID        | NATS notifications channel ID                                        |                       |
| MF_NATS_URL                | Mainflux NATS broker URL                                             | nats://localhost:4222 |
| MF_AUTH_GRPC_URL           | Auth service gRPC URL                                                | localhost:8181        |
| MF_AUTH_GRPC_TIMEOUT       | Auth service gRPC request timeout in seconds                         | 1s                    |
| MF_TWINS_CACHE_URL         | Cache database URL                                                   | localhost:6379        |
| MF_TWINS_CACHE_PASS        | Cache database password                                              |                       |
| MF_TWINS_CACHE_DB          | Cache instance name                                                  | 0                     |


## Deployment

The service itself is distributed as Docker container. Check the [`twins`](https://github.com/mainflux/mainflux/blob/master/docker/addons/twins/docker-compose.yml#L35-L58) service section in 
docker-compose to see how service is deployed.

To start the service outside of the container, execute the following shell
script:

```bash
# download the latest version of the service
go get github.com/mainflux/mainflux

cd $GOPATH/src/github.com/mainflux/mainflux

# compile the twins
make twins

# copy binary to bin
make install

# set the environment variables and run the service
MF_TWINS_LOG_LEVEL: [Twins log level] \
MF_TWINS_HTTP_PORT: [Service HTTP port] \
MF_TWINS_SERVER_CERT: [String path to server cert in pem format] \
MF_TWINS_SERVER_KEY: [String path to server key in pem format] \
MF_JAEGER_URL: [Jaeger server URL] MF_TWINS_DB: [Database name] \
MF_TWINS_DB_HOST: [Database host address] \
MF_TWINS_DB_PORT: [Database host port] \
MF_THINGS_STANDALONE_EMAIL=[User email for standalone mode (no gRPC communication with auth)] \
MF_THINGS_STANDALONE_TOKEN=[User token for standalone mode that should be passed in auth header] \
MF_TWINS_CLIENT_TLS: [Flag that indicates if TLS should be turned on] \
MF_TWINS_CA_CERTS: [Path to trusted CAs in PEM format] \
MF_TWINS_CHANNEL_ID: [NATS notifications channel ID] \
MF_NATS_URL: [Mainflux NATS broker URL] \
MF_AUTH_GRPC_URL: [Auth service gRPC URL] \
MF_AUTH_GRPC_TIMEOUT: [Auth service gRPC request timeout in seconds] \
$GOBIN/mainflux-twins
```

## Usage

### Starting twins service

The twins service publishes notifications on a NATS subject of the format
`channels.<MF_TWINS_CHANNEL_ID>.messages.<twinID>.<crudOp>`, where `crudOp`
stands for the crud operation done on twin - create, update, delete or
retrieve - or state - save state. In order to use twin service notifications,
one must inform it - via environment variables - about the Mainflux channel used
for notification publishing. You must use an already existing channel, since you
cannot know in advance or set the channel ID (Mainflux does it automatically).

To set the environment variable, please go to `.env` file and set the following
variable:

```
MF_TWINS_CHANNEL_ID=
```

with the corresponding values of the desired channel. If you are running
mainflux natively, than do the same thing in the corresponding console
environment.

For more information about service capabilities and its usage, please check out
the [API documentation](https://api.mainflux.io/?urls.primaryName=twins-openapi.yml).

[doc]: https://docs.mainflux.io
