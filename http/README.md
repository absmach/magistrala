# HTTP adapter

HTTP adapter provides an HTTP API for sending messages through the platform.

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                          | Description                                                                         | Default                           |
| --------------------------------- | ----------------------------------------------------------------------------------- | --------------------------------- |
| MG_HTTP_ADAPTER_LOG_LEVEL         | Log level for the HTTP Adapter (debug, info, warn, error)                           | info                              |
| MG_HTTP_ADAPTER_HOST              | Service HTTP host                                                                   | ""                                |
| MG_HTTP_ADAPTER_PORT              | Service HTTP port                                                                   | 80                                |
| MG_HTTP_ADAPTER_SERVER_CERT       | Path to the PEM encoded server certificate file                                     | ""                                |
| MG_HTTP_ADAPTER_SERVER_KEY        | Path to the PEM encoded server key file                                             | ""                                |
| MG_CLIENTS_AUTH_GRPC_URL          | Clients service Auth gRPC URL                                                       | <localhost:7000>                  |
| MG_CLIENTS_AUTH_GRPC_TIMEOUT      | Clients service Auth gRPC request timeout in seconds                                | 1s                                |
| MG_CLIENTS_AUTH_GRPC_CLIENT_CERT  | Path to the PEM encoded clients service Auth gRPC client certificate file           | ""                                |
| MG_CLIENTS_AUTH_GRPC_CLIENT_KEY   | Path to the PEM encoded clients service Auth gRPC client key file                   | ""                                |
| MG_CLIENTS_AUTH_GRPC_SERVER_CERTS | Path to the PEM encoded clients server Auth gRPC server trusted CA certificate file | ""                                |
| MG_MESSAGE_BROKER_URL             | Message broker instance URL                                                         | <nats://localhost:4222>           |
| MG_JAEGER_URL                     | Jaeger server URL                                                                   | <http://localhost:4318/v1/traces> |
| MG_JAEGER_TRACE_RATIO             | Jaeger sampling ratio                                                               | 1.0                               |
| MG_SEND_TELEMETRY                 | Send telemetry to magistrala call home server                                       | true                              |
| MG_HTTP_ADAPTER_INSTANCE_ID       | Service instance ID                                                                 | ""                                |

## Deployment

The service itself is distributed as Docker container. Check the [`http-adapter`](https://github.com/absmach/magistrala/blob/main/docker/docker-compose.yml) service section in docker-compose file to see how service is deployed.

Running this service outside of container requires working instance of the message broker service, clients service and Jaeger server.
To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/magistrala

cd magistrala

# compile the http
make http

# copy binary to bin
make install

# set the environment variables and run the service
MG_HTTP_ADAPTER_LOG_LEVEL=info \
MG_HTTP_ADAPTER_HOST=localhost \
MG_HTTP_ADAPTER_PORT=80 \
MG_HTTP_ADAPTER_SERVER_CERT="" \
MG_HTTP_ADAPTER_SERVER_KEY="" \
MG_CLIENTS_AUTH_GRPC_URL=localhost:7000 \
MG_CLIENTS_AUTH_GRPC_TIMEOUT=1s \
MG_CLIENTS_AUTH_GRPC_CLIENT_CERT="" \
MG_CLIENTS_AUTH_GRPC_CLIENT_KEY="" \
MG_CLIENTS_AUTH_GRPC_SERVER_CERTS="" \
MG_MESSAGE_BROKER_URL=nats://localhost:4222 \
MG_JAEGER_URL=http://localhost:14268/api/traces \
MG_JAEGER_TRACE_RATIO=1.0 \
MG_SEND_TELEMETRY=true \
MG_HTTP_ADAPTER_INSTANCE_ID="" \
$GOBIN/magistrala-http
```

Setting `MG_HTTP_ADAPTER_SERVER_CERT` and `MG_HTTP_ADAPTER_SERVER_KEY` will enable TLS against the service. The service expects a file in PEM format for both the certificate and the key.

Setting `MG_CLIENTS_AUTH_GRPC_CLIENT_CERT` and `MG_CLIENTS_AUTH_GRPC_CLIENT_KEY` will enable TLS against the clients service. The service expects a file in PEM format for both the certificate and the key. Setting `MG_CLIENTS_AUTH_GRPC_SERVER_CERTS` will enable TLS against the clients service trusting only those CAs that are provided. The service expects a file in PEM format of trusted CAs.

## Usage

HTTP Authorization request header contains the credentials to authenticate a Client. The authorization header can be a plain Client key or a Client key encoded as a password for Basic Authentication. In case the Basic Authentication schema is used, the username is ignored. For more information about service capabilities and its usage, please check out the [API documentation](https://docs.api.magistrala.abstractmachines.fr/?urls.primaryName=http.yml).
