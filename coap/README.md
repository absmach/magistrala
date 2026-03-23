# SuperMQ CoAP Adapter

SuperMQ CoAP adapter provides an [CoAP](http://coap.technology/) API for sending messages through the platform.

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                              | Description                                                                                  | Default                               |
| ------------------------------------- | -------------------------------------------------------------------------------------------- | ------------------------------------- |
| `MG_COAP_ADAPTER_LOG_LEVEL`          | Log level for the CoAP Adapter (`debug`, `info`, `warn`, `error`)                            | info                                  |
| `MG_COAP_ADAPTER_HOST`               | CoAP service listening host                                                                  | ""                                    |
| `MG_COAP_ADAPTER_PORT`               | CoAP service listening port                                                                  | 5683                                  |
| `MG_COAP_ADAPTER_SERVER_CERT`        | Path to the PEM-encoded CoAP server certificate                                              | ""                                    |
| `MG_COAP_ADAPTER_SERVER_KEY`         | Path to the PEM-encoded CoAP server key                                                      | ""                                    |
| `MG_COAP_ADAPTER_HTTP_HOST`          | Service HTTP listening host                                                                  | ""                                    |
| `MG_COAP_ADAPTER_HTTP_PORT`          | Service HTTP listening port                                                                  | 5683                                  |
| `MG_COAP_ADAPTER_HTTP_SERVER_CERT`   | Path to the PEM-encoded HTTP server certificate                                              | ""                                    |
| `MG_COAP_ADAPTER_HTTP_SERVER_KEY`    | Path to the PEM-encoded HTTP server key                                                      | ""                                    |
| `MG_COAP_ADAPTER_CACHE_NUM_COUNTERS` | Number of cache counters that track topic parsing frequency                                  | 200000                                |
| `MG_COAP_ADAPTER_CACHE_MAX_COST`     | Maximum cache size (bytes)                                                                   | 1048576                               |
| `MG_COAP_ADAPTER_CACHE_BUFFER_ITEMS` | Number of cache `Get` buffer items                                                           | 64                                    |
| `MG_CLIENTS_GRPC_URL`                | Clients service Auth gRPC URL                                                                | <localhost:7000>                      |
| `MG_CLIENTS_GRPC_TIMEOUT`            | Clients service Auth gRPC request timeout                                                    | 1s                                    |
| `MG_CLIENTS_GRPC_CLIENT_CERT`        | Path to the PEM-encoded clients service Auth gRPC client certificate file                    | ""                                    |
| `MG_CLIENTS_GRPC_CLIENT_KEY`         | Path to the PEM-encoded clients service Auth gRPC client key file                            | ""                                    |
| `MG_CLIENTS_GRPC_SERVER_CERTS`       | Path to the PEM-encoded clients server Auth gRPC trusted CA certificate file                 | ""                                    |
| `MG_MESSAGE_BROKER_URL`              | Message broker instance URL                                                                  | <amqp://guest:guest@rabbitmq:5672/>   |
| `MG_JAEGER_URL`                      | Jaeger server URL                                                                            | <http://localhost:4318/v1/traces>     |
| `MG_JAEGER_TRACE_RATIO`              | Jaeger sampling ratio                                                                        | 1.0                                   |
| `MG_SEND_TELEMETRY`                  | Send telemetry to SuperMQ call-home server                                                   | true                                  |
| `MG_COAP_ADAPTER_INSTANCE_ID`        | CoAP adapter instance ID                                                                     | ""                                    |

## Deployment

The service itself is distributed as Docker container. Check the [`coap-adapter`](https://github.com/absmach/supermq/blob/main/docker/docker-compose.yaml) service section in docker-compose file to see how service is deployed.

Running this service outside of container requires working instance of the message broker service, clients service and Jaeger server.
To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/supermq

cd supermq

# compile the http
make coap

# copy binary to bin
make install

# set the environment variables and run the service
MG_COAP_ADAPTER_LOG_LEVEL=info \
MG_COAP_ADAPTER_HOST=localhost \
MG_COAP_ADAPTER_PORT=5683 \
MG_COAP_ADAPTER_SERVER_CERT="" \
MG_COAP_ADAPTER_SERVER_KEY="" \
MG_COAP_ADAPTER_HTTP_HOST=localhost \
MG_COAP_ADAPTER_HTTP_PORT=5683 \
MG_COAP_ADAPTER_HTTP_SERVER_CERT="" \
MG_COAP_ADAPTER_HTTP_SERVER_KEY="" \
MG_COAP_ADAPTER_CACHE_NUM_COUNTERS=200000 \
MG_COAP_ADAPTER_CACHE_MAX_COST=1048576 \
MG_COAP_ADAPTER_CACHE_BUFFER_ITEMS=64 \
MG_CLIENTS_GRPC_URL=localhost:7000 \
MG_CLIENTS_GRPC_TIMEOUT=1s \
MG_CLIENTS_GRPC_CLIENT_CERT="" \
MG_CLIENTS_GRPC_CLIENT_KEY="" \
MG_CLIENTS_GRPC_SERVER_CERTS="" \
MG_MESSAGE_BROKER_URL=amqp://guest:guest@rabbitmq:5672/ \
MG_JAEGER_URL=http://localhost:14268/api/traces \
MG_JAEGER_TRACE_RATIO=1.0 \
MG_SEND_TELEMETRY=true \
MG_COAP_ADAPTER_INSTANCE_ID="" \
$GOBIN/supermq-coap
```

Setting `MG_COAP_ADAPTER_SERVER_CERT` and `MG_COAP_ADAPTER_SERVER_KEY` will enable TLS against the service. The service expects a file in PEM format for both the certificate and the key. Setting `MG_COAP_ADAPTER_HTTP_SERVER_CERT` and `MG_COAP_ADAPTER_HTTP_SERVER_KEY` will enable TLS against the service. The service expects a file in PEM format for both the certificate and the key.

Setting `MG_CLIENTS_GRPC_CLIENT_CERT` and `MG_CLIENTS_GRPC_CLIENT_KEY` will enable TLS against the clients service. The service expects a file in PEM format for both the certificate and the key. Setting `MG_CLIENTS_GRPC_SERVER_CERTS` will enable TLS against the clients service trusting only those CAs that are provided. The service expects a file in PEM format of trusted CAs.

## Usage

If CoAP adapter is running locally (on default 5683 port), a valid URL would be: `coap://localhost/m/<domain_id>/c/<channel_id>/<subtopic>?auth=<client_auth_key>`.
Since CoAP protocol does not support `Authorization` header (option) and options have limited size, in order to send CoAP messages, valid `auth` value (a valid Client key) must be present in `Uri-Query` option.

## Best Practices

- Use distinct client auth keys and rotate them frequently for better security.

- Use meaningful channel IDs and subtopics so you know exactly where your messages go.

- Leverage metadata/tags in channels and clients (via clients service) to filter and manage messaging paths.

- Ensure the auth query parameter is not exposed publicly (use secure networks or DTLS if available).

- Monitor message broker load and usage patterns — CoAP traffic can burst.

- Use the /health endpoint (if exposed) to monitor service status and integrate with your observability stack.

## Versioning and  Health Check

If the service exposes a /health endpoint, you can use it for monitoring and version readiness checks.

```bash
curl -X GET coap://localhost/health \
  -H "accept: application/health+json"
```

The expected response is:

```bash
{
  "status": "pass",
  "version": "0.xx.x",
  "commit": "<commit‑hash>",
  "description": "coap‑adapter service",
  "build_time": "YYYY‑MM‑DDT…"
}
```

## CLI

SuperMQ provides a CoAP CLI for testing and interacting with the CoAP Adapter.
To learn more about this visit the [SuperMQ CoAp CLI page](https://github.com/absmach/coap-cli/tree/main).
