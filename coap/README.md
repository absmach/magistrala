# Mainflux CoAP Adapter

Mainflux CoAP adapter provides an [CoAP](http://coap.technology/) API for sending messages through the
platform.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                       | Description                                            | Default               |
|--------------------------------|--------------------------------------------------------|-----------------------|
| MF_COAP_ADAPTER_PORT           | Service listening port                                 | 5683                  |
| MF_BROKER_URL                  | Message broker instance URL                            | nats://localhost:4222 |
| MF_COAP_ADAPTER_LOG_LEVEL      | Service log level                                      | info                  |
| MF_COAP_ADAPTER_CLIENT_TLS     | Flag that indicates if TLS should be turned on         | false                 |
| MF_COAP_ADAPTER_CA_CERTS       | Path to trusted CAs in PEM format                      |                       |
| MF_COAP_ADAPTER_PING_PERIOD    | Hours between 1 and 24 to ping client with ACK message | 12                    |
| MF_JAEGER_URL                  | Jaeger server URL                                      | localhost:6831        |
| MF_THINGS_AUTH_GRPC_URL        | Things service Auth gRPC URL                           | localhost:8181        |
| MF_THINGS_AUTH_GRPC_TIMEOUT    | Things service Auth gRPC request timeout in seconds    | 1s                    |

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
MF_BROKER_URL=[Message broker instance URL] \
MF_COAP_ADAPTER_PORT=[Service HTTP port] \
MF_COAP_ADAPTER_LOG_LEVEL=[Service log level] \
MF_COAP_ADAPTER_CLIENT_TLS=[Flag that indicates if TLS should be turned on] \
MF_COAP_ADAPTER_CA_CERTS=[Path to trusted CAs in PEM format] \
MF_COAP_ADAPTER_PING_PERIOD: [Hours between 1 and 24 to ping client with ACK message] \
MF_JAEGER_URL=[Jaeger server URL] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout in seconds] \
$GOBIN/mainflux-coap
```

## Usage

If CoAP adapter is running locally (on default 5683 port), a valid URL would be: `coap://localhost/channels/<channel_id>/messages?auth=<thing_auth_key>`.
Since CoAP protocol does not support `Authorization` header (option) and options have limited size, in order to send CoAP messages, valid `auth` value (a valid Thing key) must be present in `Uri-Query` option.
