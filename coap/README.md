# Magistrala CoAP Adapter

Magistrala CoAP adapter provides an [CoAP](http://coap.technology/) API for sending messages through the platform.

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                         | Description                                                                        | Default                             |
| -------------------------------- | ---------------------------------------------------------------------------------- | ----------------------------------- |
| MG_COAP_ADAPTER_LOG_LEVEL        | Log level for the CoAP Adapter (debug, info, warn, error)                          | info                                |
| MG_COAP_ADAPTER_HOST             | CoAP service listening host                                                        | ""                                  |
| MG_COAP_ADAPTER_PORT             | CoAP service listening port                                                        | 5683                                |
| MG_COAP_ADAPTER_SERVER_CERT      | CoAP service server certificate                                                    | ""                                  |
| MG_COAP_ADAPTER_SERVER_KEY       | CoAP service server key                                                            | ""                                  |
| MG_COAP_ADAPTER_HTTP_HOST        | Service HTTP listening host                                                        | ""                                  |
| MG_COAP_ADAPTER_HTTP_PORT        | Service listening port                                                             | 5683                                |
| MG_COAP_ADAPTER_HTTP_SERVER_CERT | Service server certificate                                                         | ""                                  |
| MG_COAP_ADAPTER_HTTP_SERVER_KEY  | Service server key                                                                 | ""                                  |
| MG_THINGS_AUTH_GRPC_URL          | Things service Auth gRPC URL                                                       | <localhost:7000>                    |
| MG_THINGS_AUTH_GRPC_TIMEOUT      | Things service Auth gRPC request timeout in seconds                                | 1s                                  |
| MG_THINGS_AUTH_GRPC_CLIENT_CERT  | Path to the PEM encoded things service Auth gRPC client certificate file           | ""                                  |
| MG_THINGS_AUTH_GRPC_CLIENT_KEY   | Path to the PEM encoded things service Auth gRPC client key file                   | ""                                  |
| MG_THINGS_AUTH_GRPC_SERVER_CERTS | Path to the PEM encoded things server Auth gRPC server trusted CA certificate file | ""                                  |
| MG_MESSAGE_BROKER_URL            | Message broker instance URL                                                        | <nats://localhost:4222>             |
| MG_JAEGER_URL                    | Jaeger server URL                                                                  | <http://localhost:14268/api/traces> |
| MG_JAEGER_TRACE_RATIO            | Jaeger sampling ratio                                                              | 1.0                                 |
| MG_SEND_TELEMETRY                | Send telemetry to magistrala call home server                                      | true                                |
| MG_COAP_ADAPTER_INSTANCE_ID      | CoAP adapter instance ID                                                           | ""                                  |

## Deployment

The service itself is distributed as Docker container. Check the [`coap-adapter`](https://github.com/absmach/magistrala/blob/main/docker/docker-compose.yml) service section in docker-compose to see how service is deployed.

Running this service outside of container requires working instance of the message broker service, things service and Jaeger server.
To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/magistrala

cd magistrala

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
MG_THINGS_AUTH_GRPC_URL=localhost:7000 \
MG_THINGS_AUTH_GRPC_TIMEOUT=1s \
MG_THINGS_AUTH_GRPC_CLIENT_CERT="" \
MG_THINGS_AUTH_GRPC_CLIENT_KEY="" \
MG_THINGS_AUTH_GRPC_SERVER_CERTS="" \
MG_MESSAGE_BROKER_URL=nats://localhost:4222 \
MG_JAEGER_URL=http://localhost:14268/api/traces \
MG_JAEGER_TRACE_RATIO=1.0 \
MG_SEND_TELEMETRY=true \
MG_COAP_ADAPTER_INSTANCE_ID="" \
$GOBIN/magistrala-coap
```

Setting `MG_COAP_ADAPTER_SERVER_CERT` and `MG_COAP_ADAPTER_SERVER_KEY` will enable TLS against the service. The service expects a file in PEM format for both the certificate and the key. Setting `MG_COAP_ADAPTER_HTTP_SERVER_CERT` and `MG_COAP_ADAPTER_HTTP_SERVER_KEY` will enable TLS against the service. The service expects a file in PEM format for both the certificate and the key.

Setting `MG_THINGS_AUTH_GRPC_CLIENT_CERT` and `MG_THINGS_AUTH_GRPC_CLIENT_KEY` will enable TLS against the things service. The service expects a file in PEM format for both the certificate and the key. Setting `MG_THINGS_AUTH_GRPC_SERVER_CERTS` will enable TLS against the things service trusting only those CAs that are provided. The service expects a file in PEM format of trusted CAs.

## Usage

If CoAP adapter is running locally (on default 5683 port), a valid URL would be: `coap://localhost/channels/<channel_id>/messages?auth=<thing_auth_key>`.
Since CoAP protocol does not support `Authorization` header (option) and options have limited size, in order to send CoAP messages, valid `auth` value (a valid Thing key) must be present in `Uri-Query` option.
