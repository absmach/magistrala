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

| Variable                   | Description                                                         | Default                          |
| -------------------------- | ------------------------------------------------------------------- | -------------------------------- |
| MG_TWINS_LOG_LEVEL         | Log level for twin service (debug, info, warn, error)               | info                             |
| MG_TWINS_HTTP_PORT         | Twins service HTTP port                                             | 9018                             |
| MG_TWINS_SERVER_CERT       | Path to server certificate in PEM format                            |                                  |
| MG_TWINS_SERVER_KEY        | Path to server key in PEM format                                    |                                  |
| MG_JAEGER_URL              | Jaeger server URL                                                   | <http://jaeger:14268/api/traces> |
| MG_TWINS_DB                | Database name                                                       | magistrala                       |
| MG_TWINS_DB_HOST           | Database host address                                               | localhost                        |
| MG_TWINS_DB_PORT           | Database host port                                                  | 27017                            |
| MG_THINGS_STANDALONE_ID    | User ID for standalone mode (no gRPC communication with users)      |                                  |
| MG_THINGS_STANDALONE_TOKEN | User token for standalone mode that should be passed in auth header |                                  |
| MG_TWINS_CLIENT_TLS        | Flag that indicates if TLS should be turned on                      | false                            |
| MG_TWINS_CA_CERTS          | Path to trusted CAs in PEM format                                   |                                  |
| MG_TWINS_CHANNEL_ID        | Message broker notifications channel ID                             |                                  |
| MG_MESSAGE_BROKER_URL      | Magistrala Message broker URL                                       | <nats://localhost:4222>          |
| MG_AUTH_GRPC_URL           | Auth service gRPC URL                                               | <localhost:7001>                 |
| MG_AUTH_GRPC_TIMEOUT       | Auth service gRPC request timeout in seconds                        | 1s                               |
| MG_TWINS_CACHE_URL         | Cache database URL                                                  | <redis://localhost:6379/0>       |
| MG_SEND_TELEMETRY          | Send telemetry to magistrala call home server                       | true                             |

## Deployment

The service itself is distributed as Docker container. Check the [`twins`](https://github.com/absmach/magistrala/blob/master/docker/addons/twins/docker-compose.yml#L35-L58) service section in
docker-compose to see how service is deployed.

To start the service outside of the container, execute the following shell
script:

```bash
# download the latest version of the service
go get github.com/absmach/magistrala

cd $GOPATH/src/github.com/absmach/magistrala

# compile the twins
make twins

# copy binary to bin
make install

# set the environment variables and run the service
MG_TWINS_LOG_LEVEL=[Twins log level] \
MG_TWINS_HTTP_PORT=[Service HTTP port] \
MG_TWINS_SERVER_CERT=[String path to server cert in pem format] \
MG_TWINS_SERVER_KEY=[String path to server key in pem format] \
MG_JAEGER_URL=[Jaeger server URL] \
MG_TWINS_DB=[Database name] \
MG_TWINS_DB_HOST=[Database host address] \
MG_TWINS_DB_PORT=[Database host port] \
MG_THINGS_STANDALONE_EMAIL=[User email for standalone mode (no gRPC communication with auth)] \
MG_THINGS_STANDALONE_TOKEN=[User token for standalone mode that should be passed in auth header] \
MG_TWINS_CLIENT_TLS=[Flag that indicates if TLS should be turned on] \
MG_TWINS_CA_CERTS=[Path to trusted CAs in PEM format] \
MG_TWINS_CHANNEL_ID=[Message broker notifications channel ID] \
MG_MESSAGE_BROKER_URL=[Magistrala Message broker URL] \
MG_AUTH_GRPC_URL=[Auth service gRPC URL] \
MG_AUTH_GRPC_TIMEOUT=[Auth service gRPC request timeout in seconds] \
MG_TWINS_CACHE_URL=[Cache database URL] \
$GOBIN/magistrala-twins
```

## Usage

### Starting twins service

The twins service publishes notifications on a Message broker subject of the format
`channels.<MG_TWINS_CHANNEL_ID>.messages.<twinID>.<crudOp>`, where `crudOp`
stands for the crud operation done on twin - create, update, delete or
retrieve - or state - save state. In order to use twin service notifications,
one must inform it - via environment variables - about the Magistrala channel used
for notification publishing. You must use an already existing channel, since you
cannot know in advance or set the channel ID (Magistrala does it automatically).

To set the environment variable, please go to `.env` file and set the following
variable:

```
MG_TWINS_CHANNEL_ID=
```

with the corresponding values of the desired channel. If you are running
magistrala natively, than do the same thing in the corresponding console
environment.

For more information about service capabilities and its usage, please check out
the [API documentation](https://api.mainflux.io/?urls.primaryName=twins-openapi.yml).

[doc]: https://docs.mainflux.io
