# Magistrala CoAP Adapter

Magistrala CoAP adapter provides an [CoAP](http://coap.technology/) API for sending messages through the
platform.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                         | Description                                         | Default                        |
| -------------------------------- | --------------------------------------------------- | ------------------------------ |
| MG_COAP_ADAPTER_LOG_LEVEL        | Service log level                                   | info                           |
| MG_COAP_ADAPTER_HOST             | CoAP service listening host                         |                                |
| MG_COAP_ADAPTER_PORT             | CoAP service listening port                         | 5683                           |
| MG_COAP_ADAPTER_SERVER_CERT      | CoAP service server certificate                     |                                |
| MG_COAP_ADAPTER_SERVER_KEY       | CoAP service server key                             |                                |
| MG_COAP_ADAPTER_HTTP_HOST        | Service HTTP listening host                         |                                |
| MG_COAP_ADAPTER_HTTP_PORT        | Service listening port                              | 5683                           |
| MG_COAP_ADAPTER_HTTP_SERVER_CERT | Service server certificate                          |                                |
| MG_COAP_ADAPTER_HTTP_SERVER_KEY  | Service server key                                  |                                |
| MG_THINGS_AUTH_GRPC_URL          | Things service Auth gRPC URL                        | localhost:7000                 |
| MG_THINGS_AUTH_GRPC_TIMEOUT      | Things service Auth gRPC request timeout in seconds | 1s                             |
| MG_THINGS_AUTH_GRPC_CLIENT_TLS   | Flag that indicates if TLS should be turned on      | false                          |
| MG_THINGS_AUTH_GRPC_CA_CERTS     | Path to trusted CAs in PEM format                   |                                |
| MG_MESSAGE_BROKER_URL            | Message broker instance URL                         | nats://localhost:4222          |
| MG_JAEGER_URL                    | Jaeger server URL                                   | http://jaeger:14268/api/traces |
| MG_SEND_TELEMETRY                | Send telemetry to magistrala call home server       | true                           |
| MG_COAP_ADAPTER_INSTANCE_ID      | CoAP adapter instance ID                            |                                |

## Deployment

The service itself is distributed as Docker container. Check the [`coap-adapter`](https://github.com/absmach/magistrala/blob/master/docker/docker-compose.yml#L273-L291) service section in
docker-compose to see how service is deployed.

Running this service outside of container requires working instance of the message broker service.
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
MG_COAP_ADAPTER_LOG_LEVEL=[Service log level] \
MG_COAP_ADAPTER_HOST=[CoAP service host] \
MG_COAP_ADAPTER_PORT=[CoAP service port] \
MG_COAP_ADAPTER_SERVER_CERT=[Path to CoAP server certificate] \
MG_COAP_ADAPTER_SERVER_KEY=[Path to CoAP server key] \
MG_COAP_ADAPTER_HTTP_HOST=[CoAP HTTP service host] \
MG_COAP_ADAPTER_HTTP_PORT=[CoAP HTTP service port] \
MG_COAP_ADAPTER_HTTP_SERVER_CERT=[Path to HTTP server certificate] \
MG_COAP_ADAPTER_HTTP_SERVER_KEY=[Path to HTTP server key] \
MG_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MG_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout in seconds] \
MG_THINGS_AUTH_GRPC_CLIENT_TLS=[Flag that indicates if TLS should be turned on] \
MG_THINGS_AUTH_GRPC_CA_CERTS=[Path to trusted CAs in PEM format] \
MG_MESSAGE_BROKER_URL=[Message broker instance URL] \
MG_JAEGER_URL=[Jaeger server URL] \
MG_SEND_TELEMETRY=[Send telemetry to magistrala call home server] \
MG_COAP_ADAPTER_INSTANCE_ID=[CoAP adapter instance ID] \
$GOBIN/magistrala-coap
```

## Usage

If CoAP adapter is running locally (on default 5683 port), a valid URL would be: `coap://localhost/channels/<channel_id>/messages?auth=<thing_auth_key>`.
Since CoAP protocol does not support `Authorization` header (option) and options have limited size, in order to send CoAP messages, valid `auth` value (a valid Thing key) must be present in `Uri-Query` option.
