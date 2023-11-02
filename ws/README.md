# WebSocket adapter

WebSocket adapter provides an [WebSocket](https://en.wikipedia.org/wiki/WebSocket#:~:text=WebSocket%20is%20a%20computer%20communications,protocol%20is%20known%20as%20WebSockets.) API for sending and receiving messages through the platform.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                       | Description                                         | Default                          |
| ------------------------------ | --------------------------------------------------- | -------------------------------- |
| MG_WS_ADAPTER_LOG_LEVEL        | Log level for the WS Adapter                        | info                             |
| MG_WS_ADAPTER_HTTP_HOST        | Service WS host                                     |                                  |
| MG_WS_ADAPTER_HTTP_PORT        | Service WS port                                     | 8190                             |
| MG_WS_ADAPTER_HTTP_SERVER_CERT | Service WS server certificate                       |                                  |
| MG_WS_ADAPTER_HTTP_SERVER_KEY  | Service WS server key                               |                                  |
| MG_THINGS_AUTH_GRPC_URL        | Things service Auth gRPC URL                        | <localhost:7000>                 |
| MG_THINGS_AUTH_GRPC_TIMEOUT    | Things service Auth gRPC request timeout in seconds | 1s                               |
| MG_THINGS_AUTH_GRPC_CLIENT_TLS | Flag that indicates if TLS should be turned on      | false                            |
| MG_THINGS_AUTH_GRPC_CA_CERTS   | Path to trusted CAs in PEM format                   |                                  |
| MG_MESSAGE_BROKER_URL          | Message broker instance URL                         | <nats://localhost:4222>          |
| MG_JAEGER_URL                  | Jaeger server URL                                   | <http://jaeger:14268/api/traces> |
| MG_SEND_TELEMETRY              | Send telemetry to magistrala call home server       | true                             |
| MG_WS_ADAPTER_INSTANCE_ID      | Service instance ID                                 | ""                               |

## Deployment

The service is distributed as Docker container. Check the [`ws-adapter`](https://github.com/absmach/magistrala/blob/master/docker/docker-compose.yml#L350-L368) service section in docker-compose to see how the service is deployed.

Running this service outside of container requires working instance of the message broker service.
To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/magistrala

cd magistrala

# compile the ws
make ws

# copy binary to bin
make install

# set the environment variables and run the service
MG_WS_ADAPTER_LOG_LEVEL=[WS adapter log level] \
MG_WS_ADAPTER_HTTP_HOST=[Service WS host] \
MG_WS_ADAPTER_HTTP_PORT=[Service WS port] \
MG_WS_ADAPTER_HTTP_SERVER_CERT=[Service WS server certificate] \
MG_WS_ADAPTER_HTTP_SERVER_KEY=[Service WS server key] \
MG_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MG_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout in seconds] \
MG_THINGS_AUTH_GRPC_CLIENT_TLS=[Flag that indicates if TLS should be turned on] \
MG_THINGS_AUTH_GRPC_CA_CERTS=[Path to trusted CAs in PEM format] \
MG_MESSAGE_BROKER_URL=[Message broker instance URL] \
MG_JAEGER_URL=[Jaeger server URL] \
MG_SEND_TELEMETRY=[Send telemetry to magistrala call home server] \
MG_WS_ADAPTER_INSTANCE_ID=[Service instance ID] \
$GOBIN/magistrala-ws
```

## Usage

For more information about service capabilities and its usage, please check out
the [WebSocket paragraph](https://mainflux.readthedocs.io/en/latest/messaging/#websocket) in the Getting Started guide.
