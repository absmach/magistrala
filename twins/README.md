# Twins

Service twins is used for CRUD and update of digital twins. Twin is a semantic
representation of a real world entity, be it device, application or something
else. It holds the sequence of attribute based definitions of a real world thing
and refers to the time series of definition based states that hold the
historical data about the represented real world thing.

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
| MF_TWINS_SINGLE_USER_EMAIL | User email for single user mode (no gRPC communication with users)   |                       |
| MF_TWINS_SINGLE_USER_TOKEN | User token for single user mode that should be passed in auth header |                       |
| MF_TWINS_CLIENT_TLS        | Flag that indicates if TLS should be turned on                       | false                 |
| MF_TWINS_CA_CERTS          | Path to trusted CAs in PEM format                                    |                       |
| MF_TWINS_MQTT_URL          | Mqtt broker URL for twin CRUD and states update notifications        | tcp://localhost:1883  |
| MF_TWINS_THING_ID          | ID of thing representing twins service & mqtt user                   |                       |
| MF_TWINS_THING_KEY         | Key of thing representing twins service & mqtt pass                  |                       |
| MF_TWINS_CHANNEL_ID        | Mqtt notifications topic                                             |                       |
| MF_NATS_URL                | Mainflux NATS broker URL                                             | nats://localhost:4222 |
| MF_AUTHN_GRPC_URL          | AuthN service gRPC URL                                               | localhost:8181        |
| MF_AUTHN_GRPC_TIMEOUT      | AuthN service gRPC request timeout in seconds                        | 1                     |

## Deployment

The service itself is distributed as Docker container. The following snippet
provides a compose file template that can be used to deploy the service container
locally:

```yaml
version: "3"
services:
  twins:
    image: mainflux/twins:[version]
    container_name: [instance name]
    ports:
      - [host machine port]:[configured HTTP port]
    environment:
      MF_TWINS_LOG_LEVEL: [Twins log level]
      MF_TWINS_HTTP_PORT: [Service HTTP port]
      MF_TWINS_SERVER_CERT: [String path to server cert in pem format]
      MF_TWINS_SERVER_KEY: [String path to server key in pem format]
      MF_JAEGER_URL: [Jaeger server URL]
      MF_TWINS_DB: [Database name]
      MF_TWINS_DB_HOST: [Database host address]
      MF_TWINS_DB_PORT: [Database host port]
      MF_TWINS_SINGLE_USER_EMAIL: [User email for single user mode]
      MF_TWINS_SINGLE_USER_TOKEN: [User token for single user mode]
      MF_TWINS_CLIENT_TLS: [Flag that indicates if TLS should be turned on]
      MF_TWINS_CA_CERTS: [Path to trusted CAs in PEM format]
      MF_TWINS_MQTT_URL: [Mqtt broker URL for twin CRUD and states]
      MF_TWINS_THING_ID: [ID of thing representing twins service]
      MF_TWINS_THING_KEY: [Key of thing representing twins service]
      MF_TWINS_CHANNEL_ID: [Mqtt notifications topic]
      MF_NATS_URL: [Mainflux NATS broker URL]
      MF_AUTHN_GRPC_URL: [AuthN service gRPC URL]
      MF_AUTHN_GRPC_TIMEOUT: [AuthN service gRPC request timeout in seconds]
```

To start the service outside of the container, execute the following shell script:

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
MF_TWINS_SINGLE_USER_EMAIL: [User email for single user mode] \
MF_TWINS_SINGLE_USER_TOKEN: [User token for single user mode] \
MF_TWINS_CLIENT_TLS: [Flag that indicates if TLS should be turned on] \
MF_TWINS_CA_CERTS: [Path to trusted CAs in PEM format] \
MF_TWINS_MQTT_URL: [Mqtt broker URL for twin CRUD and states] \
MF_TWINS_THING_ID: [ID of thing representing twins service] \
MF_TWINS_THING_KEY: [Key of thing representing twins service] \
MF_TWINS_CHANNEL_ID: [Mqtt notifications topic] \
MF_NATS_URL: [Mainflux NATS broker URL] \
MF_AUTHN_GRPC_URL: [AuthN service gRPC URL] \
MF_AUTHN_GRPC_TIMEOUT: [AuthN service gRPC request timeout in seconds] \
$GOBIN/mainflux-twins
```

## Usage

### Starting twins service

The twins service publishes notifications on an mqtt topic of the format
`channels/<MF_TWINS_CHANNEL_ID>/messages/<twinID>/<crudOp>`, where `crudOp`
stands for the crud operation done on twin - create, update, delete or
retrieve - or state - save state. In order to use twin service, one must
inform it - via environment variables - about the Mainflux thing and
channel used for mqtt notification publishing. You can use an already existing
thing and channel - thing has to be connected to channel - or create new ones.

To set the environment variables, please go to `.env` file and set the following
variables:

```
MF_TWINS_THING_ID=
MF_TWINS_THING_KEY=
MF_TWINS_CHANNEL_ID=
```

with the corresponding values of the desired thing and channel. If you are
running mainflux natively, than do the same thing in the corresponding console
environment.

For more information about service capabilities and its usage, please check out
the [API documentation](swagger.yaml).

[doc]: http://mainflux.readthedocs.io
