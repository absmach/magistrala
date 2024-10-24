# WebSocket adapter

WebSocket adapter provides a [WebSocket](https://en.wikipedia.org/wiki/WebSocket#:~:text=WebSocket%20is%20a%20computer%20communications,protocol%20is%20known%20as%20WebSockets.) API for sending and receiving messages through the platform.

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                          | Description                                                                         | Default                           |
| --------------------------------- | ----------------------------------------------------------------------------------- | --------------------------------- |
| MG_WS_ADAPTER_LOG_LEVEL           | Log level for the WS Adapter (debug, info, warn, error)                             | info                              |
| MG_WS_ADAPTER_HTTP_HOST           | Service WS host                                                                     | ""                                |
| MG_WS_ADAPTER_HTTP_PORT           | Service WS port                                                                     | 8190                              |
| MG_WS_ADAPTER_HTTP_SERVER_CERT    | Path to the PEM encoded server certificate file                                     | ""                                |
| MG_WS_ADAPTER_HTTP_SERVER_KEY     | Path to the PEM encoded server key file                                             | ""                                |
| MG_CLIENTS_AUTH_GRPC_URL          | Clients service Auth gRPC URL                                                       | <localhost:7000>                  |
| MG_CLIENTS_AUTH_GRPC_TIMEOUT      | Clients service Auth gRPC request timeout in seconds                                | 1s                                |
| MG_CLIENTS_AUTH_GRPC_CLIENT_CERT  | Path to the PEM encoded clients service Auth gRPC client certificate file           | ""                                |
| MG_CLIENTS_AUTH_GRPC_CLIENT_KEY   | Path to the PEM encoded clients service Auth gRPC client key file                   | ""                                |
| MG_CLIENTS_AUTH_GRPC_SERVER_CERTS | Path to the PEM encoded clients server Auth gRPC server trusted CA certificate file | ""                                |
| MG_MESSAGE_BROKER_URL             | Message broker instance URL                                                         | <nats://localhost:4222>           |
| MG_JAEGER_URL                     | Jaeger server URL                                                                   | <http://localhost:4318/v1/traces> |
| MG_JAEGER_TRACE_RATIO             | Jaeger sampling ratio                                                               | 1.0                               |
| MG_SEND_TELEMETRY                 | Send telemetry to magistrala call home server                                       | true                              |
| MG_WS_ADAPTER_INSTANCE_ID         | Service instance ID                                                                 | ""                                |

## Deployment

The service is distributed as Docker container. Check the [`ws-adapter`](https://github.com/absmach/magistrala/blob/main/docker/docker-compose.yml) service section in docker-compose file to see how the service is deployed.

Running this service outside of container requires working instance of the message broker service, clients service and Jaeger server.
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
MG_WS_ADAPTER_LOG_LEVEL=info \
MG_WS_ADAPTER_HTTP_HOST=localhost \
MG_WS_ADAPTER_HTTP_PORT=8190 \
MG_WS_ADAPTER_HTTP_SERVER_CERT="" \
MG_WS_ADAPTER_HTTP_SERVER_KEY="" \
MG_CLIENTS_AUTH_GRPC_URL=localhost:7000 \
MG_CLIENTS_AUTH_GRPC_TIMEOUT=1s \
MG_CLIENTS_AUTH_GRPC_CLIENT_CERT="" \
MG_CLIENTS_AUTH_GRPC_CLIENT_KEY="" \
MG_CLIENTS_AUTH_GRPC_SERVER_CERTS="" \
MG_MESSAGE_BROKER_URL=nats://localhost:4222 \
MG_JAEGER_URL=http://localhost:14268/api/traces \
MG_JAEGER_TRACE_RATIO=1.0 \
MG_SEND_TELEMETRY=true \
MG_WS_ADAPTER_INSTANCE_ID="" \
$GOBIN/magistrala-ws
```

Setting `MG_WS_ADAPTER_HTTP_SERVER_CERT` and `MG_WS_ADAPTER_HTTP_SERVER_KEY` will enable TLS against the service. The service expects a file in PEM format for both the certificate and the key.

Setting `MG_CLIENTS_AUTH_GRPC_CLIENT_CERT` and `MG_CLIENTS_AUTH_GRPC_CLIENT_KEY` will enable TLS against the clients service. The service expects a file in PEM format for both the certificate and the key. Setting `MG_CLIENTS_AUTH_GRPC_SERVER_CERTS` will enable TLS against the clients service trusting only those CAs that are provided. The service expects a file in PEM format of trusted CAs.

## Usage

For more information about service capabilities and its usage, please check out the [WebSocket section](https://docs.magistrala.abstractmachines.fr/messaging/#websocket).
