# SuperMQ CoAP Adapter

SuperMQ CoAP adapter provides an [CoAP](http://coap.technology/) API for sending messages through the platform.

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                          | Description                                                                         | Default                             |
| --------------------------------- | ----------------------------------------------------------------------------------- | ----------------------------------- |
| SMQ_COAP_ADAPTER_LOG_LEVEL        | Log level for the CoAP Adapter (debug, info, warn, error)                           | info                                |
| SMQ_COAP_ADAPTER_HOST             | CoAP service listening host                                                         | ""                                  |
| SMQ_COAP_ADAPTER_PORT             | CoAP service listening port                                                         | 5683                                |
| SMQ_COAP_ADAPTER_SERVER_CERT      | CoAP service server certificate                                                     | ""                                  |
| SMQ_COAP_ADAPTER_SERVER_KEY       | CoAP service server key                                                             | ""                                  |
| SMQ_COAP_ADAPTER_HTTP_HOST        | Service HTTP listening host                                                         | ""                                  |
| SMQ_COAP_ADAPTER_HTTP_PORT        | Service listening port                                                              | 5683                                |
| SMQ_COAP_ADAPTER_HTTP_SERVER_CERT | Service server certificate                                                          | ""                                  |
| SMQ_COAP_ADAPTER_HTTP_SERVER_KEY  | Service server key                                                                  | ""                                  |
| SMQ_CLIENTS_GRPC_URL              | Clients service Auth gRPC URL                                                       | <localhost:7000>                    |
| SMQ_CLIENTS_GRPC_TIMEOUT          | Clients service Auth gRPC request timeout in seconds                                | 1s                                  |
| SMQ_CLIENTS_GRPC_CLIENT_CERT      | Path to the PEM encoded clients service Auth gRPC client certificate file           | ""                                  |
| SMQ_CLIENTS_GRPC_CLIENT_KEY       | Path to the PEM encoded clients service Auth gRPC client key file                   | ""                                  |
| SMQ_CLIENTS_GRPC_SERVER_CERTS     | Path to the PEM encoded clients server Auth gRPC server trusted CA certificate file | ""                                  |
| SMQ_MESSAGE_BROKER_URL            | Message broker instance URL                                                         | <amqp://guest:guest@rabbitmq:5672/> |
| SMQ_JAEGER_URL                    | Jaeger server URL                                                                   | <http://localhost:4318/v1/traces>   |
| SMQ_JAEGER_TRACE_RATIO            | Jaeger sampling ratio                                                               | 1.0                                 |
| SMQ_SEND_TELEMETRY                | Send telemetry to magistrala call home server                                       | true                                |
| SMQ_COAP_ADAPTER_INSTANCE_ID      | CoAP adapter instance ID                                                            | ""                                  |

## Deployment

The service itself is distributed as Docker container. Check the [`coap-adapter`](https://github.com/absmach/supermq/blob/main/docker/docker-compose.yml) service section in docker-compose file to see how service is deployed.

Running this service outside of container requires working instance of the message broker service, clients service and Jaeger server.
To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/supermq

cd magistrala

# compile the http
make coap

# copy binary to bin
make install

# set the environment variables and run the service
SMQ_COAP_ADAPTER_LOG_LEVEL=info \
SMQ_COAP_ADAPTER_HOST=localhost \
SMQ_COAP_ADAPTER_PORT=5683 \
SMQ_COAP_ADAPTER_SERVER_CERT="" \
SMQ_COAP_ADAPTER_SERVER_KEY="" \
SMQ_COAP_ADAPTER_HTTP_HOST=localhost \
SMQ_COAP_ADAPTER_HTTP_PORT=5683 \
SMQ_COAP_ADAPTER_HTTP_SERVER_CERT="" \
SMQ_COAP_ADAPTER_HTTP_SERVER_KEY="" \
SMQ_CLIENTS_GRPC_URL=localhost:7000 \
SMQ_CLIENTS_GRPC_TIMEOUT=1s \
SMQ_CLIENTS_GRPC_CLIENT_CERT="" \
SMQ_CLIENTS_GRPC_CLIENT_KEY="" \
SMQ_CLIENTS_GRPC_SERVER_CERTS="" \
SMQ_MESSAGE_BROKER_URL=amqp://guest:guest@rabbitmq:5672/ \
SMQ_JAEGER_URL=http://localhost:14268/api/traces \
SMQ_JAEGER_TRACE_RATIO=1.0 \
SMQ_SEND_TELEMETRY=true \
SMQ_COAP_ADAPTER_INSTANCE_ID="" \
$GOBIN/supermq-coap
```

Setting `SMQ_COAP_ADAPTER_SERVER_CERT` and `SMQ_COAP_ADAPTER_SERVER_KEY` will enable TLS against the service. The service expects a file in PEM format for both the certificate and the key. Setting `SMQ_COAP_ADAPTER_HTTP_SERVER_CERT` and `SMQ_COAP_ADAPTER_HTTP_SERVER_KEY` will enable TLS against the service. The service expects a file in PEM format for both the certificate and the key.

Setting `SMQ_CLIENTS_GRPC_CLIENT_CERT` and `SMQ_CLIENTS_GRPC_CLIENT_KEY` will enable TLS against the clients service. The service expects a file in PEM format for both the certificate and the key. Setting `SMQ_CLIENTS_GRPC_SERVER_CERTS` will enable TLS against the clients service trusting only those CAs that are provided. The service expects a file in PEM format of trusted CAs.

## Usage

If CoAP adapter is running locally (on default 5683 port), a valid URL would be: `coap://localhost/channels/<channel_id>/messages?auth=<client_auth_key>`.
Since CoAP protocol does not support `Authorization` header (option) and options have limited size, in order to send CoAP messages, valid `auth` value (a valid Client key) must be present in `Uri-Query` option.
