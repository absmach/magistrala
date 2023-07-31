# Mainflux CoAP Adapter

Mainflux CoAP adapter provides an [CoAP](http://coap.technology/) API for sending messages through the
platform.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                         | Description                                         | Default                        |
| -------------------------------- | --------------------------------------------------- | ------------------------------ |
| MF_COAP_ADAPTER_LOG_LEVEL        | Service log level                                   | info                           |
| MF_COAP_ADAPTER_HOST             | CoAP service listening host                         |                                |
| MF_COAP_ADAPTER_PORT             | CoAP service listening port                         | 5683                           |
| MF_COAP_ADAPTER_SERVER_CERT      | CoAP service server certificate                     |                                |
| MF_COAP_ADAPTER_SERVER_KEY       | CoAP service server key                             |                                |
| MF_COAP_ADAPTER_HTTP_HOST        | Service HTTP listening host                         |                                |
| MF_COAP_ADAPTER_HTTP_PORT        | Service listening port                              | 5683                           |
| MF_COAP_ADAPTER_HTTP_SERVER_CERT | Service server certificate                          |                                |
| MF_COAP_ADAPTER_HTTP_SERVER_KEY  | Service server key                                  |                                |
| MF_THINGS_AUTH_GRPC_URL          | Things service Auth gRPC URL                        | localhost:7000                 |
| MF_THINGS_AUTH_GRPC_TIMEOUT      | Things service Auth gRPC request timeout in seconds | 1s                             |
| MF_THINGS_AUTH_GRPC_CLIENT_TLS   | Flag that indicates if TLS should be turned on      | false                          |
| MF_THINGS_AUTH_GRPC_CA_CERTS     | Path to trusted CAs in PEM format                   |                                |
| MF_BROKER_URL                    | Message broker instance URL                         | nats://localhost:4222          |
| MF_JAEGER_URL                    | Jaeger server URL                                   | http://jaeger:14268/api/traces |
| MF_SEND_TELEMETRY                | Send telemetry to mainflux call home server         | true                           |
| MF_COAP_ADAPTER_INSTANCE_ID      | CoAP adapter instance ID                            |                                |

## Deployment

The service itself is distributed as Docker container. Check the [`coap-adapter`](https://github.com/mainflux/mainflux/blob/master/docker/docker-compose.yml#L273-L291) service section in
docker-compose to see how service is deployed.

Running this service outside of container requires working instance of the message broker service.
To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/mainflux/mainflux

cd mainflux

# compile the http
make coap

# copy binary to bin
make install

# set the environment variables and run the service
MF_COAP_ADAPTER_LOG_LEVEL=[Service log level] \
MF_COAP_ADAPTER_HOST=[CoAP service host] \
MF_COAP_ADAPTER_PORT=[CoAP service port] \
MF_COAP_ADAPTER_SERVER_CERT=[Path to CoAP server certificate] \
MF_COAP_ADAPTER_SERVER_KEY=[Path to CoAP server key] \
MF_COAP_ADAPTER_HTTP_HOST=[CoAP HTTP service host] \
MF_COAP_ADAPTER_HTTP_PORT=[CoAP HTTP service port] \
MF_COAP_ADAPTER_HTTP_SERVER_CERT=[Path to HTTP server certificate] \
MF_COAP_ADAPTER_HTTP_SERVER_KEY=[Path to HTTP server key] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout in seconds] \
MF_THINGS_AUTH_GRPC_CLIENT_TLS=[Flag that indicates if TLS should be turned on] \
MF_THINGS_AUTH_GRPC_CA_CERTS=[Path to trusted CAs in PEM format] \
MF_BROKER_URL=[Message broker instance URL] \
MF_JAEGER_URL=[Jaeger server URL] \
MF_SEND_TELEMETRY=[Send telemetry to mainflux call home server] \
MF_COAP_ADAPTER_INSTANCE_ID=[CoAP adapter instance ID] \
$GOBIN/mainflux-coap
```

## Usage

If CoAP adapter is running locally (on default 5683 port), a valid URL would be: `coap://localhost/channels/<channel_id>/messages?auth=<thing_auth_key>`.
Since CoAP protocol does not support `Authorization` header (option) and options have limited size, in order to send CoAP messages, valid `auth` value (a valid Thing key) must be present in `Uri-Query` option.
