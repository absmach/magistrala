# HTTP adapter

HTTP adapter provides an HTTP API for sending messages through the platform.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                       | Description                                         | Default                        |
| ------------------------------ | --------------------------------------------------- | ------------------------------ |
| MG_HTTP_ADAPTER_LOG_LEVEL      | Log level for the HTTP Adapter                      | debug                          |
| MG_HTTP_ADAPTER_HOST           | HTTP adapter listening host                         |                                |
| MG_HTTP_ADAPTER_PORT           | Service HTTP port                                   | 80                             |
| MG_HTTP_ADAPTER_SERVER_CERT    | Service server certificate                          |                                |
| MG_HTTP_ADAPTER_SERVER_KEY     | Service server key                                  |                                |
| MG_THINGS_AUTH_GRPC_URL        | Things service Auth gRPC URL                        | localhost:7000                 |
| MG_THINGS_AUTH_GRPC_TIMEOUT    | Things service Auth gRPC request timeout in seconds | 1s                             |
| MG_THINGS_AUTH_GRPC_CLIENT_TLS | Flag that indicates if TLS should be turned on      | false                          |
| MG_THINGS_AUTH_GRPC_CA_CERTS   | Path to trusted CAs in PEM format                   |                                |
| MG_MESSAGE_BROKER_URL          | Message broker instance URL                         | nats://localhost:4222          |
| MG_JAEGER_URL                  | Jaeger server URL                                   | http://jaeger:14268/api/traces |
| MG_SEND_TELEMETRY              | Send telemetry to magistrala call home server       | true                           |
| MG_HTTP_ADAPTER_INSTANCE_ID    | HTTP Adapter instance ID                            |                                |

## Deployment

The service itself is distributed as Docker container. Check the [`http-adapter`](https://github.com/absmach/magistrala/blob/master/docker/docker-compose.yml#L245-L262) service section in
docker-compose to see how service is deployed.

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
MG_HTTP_ADAPTER_LOG_LEVEL=[HTTP Adapter Log Level] \
MG_HTTP_ADAPTER_HOST=[Service HTTP host] \
MG_HTTP_ADAPTER_PORT=[Service HTTP port] \
MG_HTTP_ADAPTER_SERVER_CERT=[Path to server certificate] \
MG_HTTP_ADAPTER_SERVER_KEY=[Path to server key] \
MG_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MG_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout in seconds] \
MG_THINGS_AUTH_GRPC_CLIENT_TLS=[Flag that indicates if TLS should be turned on] \
MG_THINGS_AUTH_GRPC_CA_CERTS=[Path to trusted CAs in PEM format] \
MG_MESSAGE_BROKER_URL=[Message broker instance URL] \
MG_JAEGER_URL=[Jaeger server URL] \
MG_SEND_TELEMETRY=[Send telemetry to magistrala call home server] \
MG_HTTP_ADAPTER_INSTANCE_ID=[HTTP Adapter instance ID] \
$GOBIN/magistrala-http
```

Setting `MG_HTTP_ADAPTER_CA_CERTS` expects a file in PEM format of trusted CAs. This will enable TLS against the Things gRPC endpoint trusting only those CAs that are provided.

## Usage

HTTP Authorization request header contains the credentials to authenticate a Thing. The authorization header can be a plain Thing key
or a Thing key encoded as a password for Basic Authentication. In case the Basic Authentication schema is used, the username is ignored.
For more information about service capabilities and its usage, please check out
the [API documentation](https://api.mainflux.io/?urls.primaryName=http.yml).

[doc]: https://docs.mainflux.io
