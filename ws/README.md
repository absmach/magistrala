# WebSocket adapter

WebSocket adapter provides an WebSocket API for sending and receiving messages through the platform.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                     | Description                                    | Default               |
|------------------------------|------------------------------------------------|-----------------------|
| MF_WS_ADAPTER_CLIENT_TLS     | Flag that indicates if TLS should be turned on | false                 |
| MF_WS_ADAPTER_CA_CERTS       | Path to trusted CAs in PEM format              |                       |
| MF_WS_ADAPTER_LOG_LEVEL      | Log level for the WS Adapter                   | error                 |
| MF_WS_ADAPTER_PORT           | Service WS port                                | 8180                  |
| MF_NATS_URL                  | NATS instance URL                              | nats://localhost:4222 |
| MF_THINGS_URL                | Things service URL                             | localhost:8181        |
| MF_JAEGER_URL                | Jaeger server URL                              | localhost:6831        |
| MF_WS_ADAPTER_THINGS_TIMEOUT | Things gRPC request timeout in seconds         | 1                     |

## Deployment

The service is distributed as Docker container. The following snippet provides
a compose file template that can be used to deploy the service container locally:

```yaml
version: "2"
services:
  ws:
    image: mainflux/ws:[version]
    container_name: [instance name]
    ports:
      - [host machine port]:[configured port]
    environment:
      MF_THINGS_URL: [Things service URL]
      MF_NATS_URL: [NATS instance URL]
      MF_WS_ADAPTER_PORT: [Service WS port]
      MF_WS_ADAPTER_LOG_LEVEL: [WS adapter log level]
      MF_WS_ADAPTER_CLIENT_TLS: [Flag that indicates if TLS should be turned on]
      MF_WS_ADAPTER_CA_CERTS: [Path to trusted CAs in PEM format]
      MF_JAEGER_URL: [Jaeger server URL]
      MF_WS_ADAPTER_THINGS_TIMEOUT: [Things gRPC request timeout in seconds]
```

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
go get github.com/mainflux/mainflux

cd $GOPATH/src/github.com/mainflux/mainflux

# compile the ws
make ws

# copy binary to bin
make install

# set the environment variables and run the service
MF_THINGS_URL=[Things service URL] MF_NATS_URL=[NATS instance URL] MF_WS_ADAPTER_PORT=[Service WS port] MF_WS_ADAPTER_LOG_LEVEL=[WS adapter log level] MF_WS_ADAPTER_CLIENT_TLS=[Flag that indicates if TLS should be turned on] MF_WS_ADAPTER_CA_CERTS=[Path to trusted CAs in PEM format] MF_JAEGER_URL=[Jaeger server URL] MF_WS_ADAPTER_THINGS_TIMEOUT=[Things gRPC request timeout in seconds] $GOBIN/mainflux-ws
```

## Usage

For more information about service capabilities and its usage, please check out
the [WebSocket paragraph](https://mainflux.readthedocs.io/en/latest/messaging/#websocket) in the Getting Started guide.
