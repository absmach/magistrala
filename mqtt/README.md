# MQTT adapter

MQTT adapter provides an MQTT API for sending messages through the platform.
MQTT adapter uses [mProxy](https://github.com/mainflux/mproxy) for proxying
traffic between client and MQTT broker.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                                 | Description                                                      | Default               |
|------------------------------------------|------------------------------------------------------------------|-----------------------|
| MF_MQTT_ADAPTER_LOG_LEVEL                | mProxy Log level                                                 | info                  |
| MF_MQTT_ADAPTER_MQTT_PORT                | mProxy port                                                      | 1883                  |
| MF_MQTT_ADAPTER_MQTT_TARGET_HOST         | MQTT broker host                                                 | 0.0.0.0               |
| MF_MQTT_ADAPTER_MQTT_TARGET_PORT         | MQTT broker port                                                 | 1883                  |
| MF_MQTT_ADAPTER_MQTT_TARGET_HEALTH_CHECK | URL of broker health check                                       | ""                    |
| MF_MQTT_ADAPTER_WS_PORT                  | mProxy MQTT over WS port                                         | 8080                  |
| MF_MQTT_ADAPTER_WS_TARGET_HOST           | MQTT broker host for MQTT over WS                                | localhost             |
| MF_MQTT_ADAPTER_WS_TARGET_PORT           | MQTT broker port for MQTT over WS                                | 8080                  |
| MF_MQTT_ADAPTER_WS_TARGET_PATH           | MQTT broker MQTT over WS path                                    | /mqtt                 |
| MF_MQTT_ADAPTER_FORWARDER_TIMEOUT        | MQTT forwarder for multiprotocol communication timeout           | 30s                   |
| MF_BROKER_URL                            | Message broker broker URL                                        | nats://127.0.0.1:4222 |
| MF_THINGS_AUTH_GRPC_URL                  | Things gRPC endpoint URL                                         | localhost:8181        |
| MF_THINGS_AUTH_GRPC_TIMEOUT              | Timeout in seconds for Things service gRPC calls                 | 1s                    |
| MF_JAEGER_URL                            | URL of Jaeger tracing service                                    | ""                    |
| MF_MQTT_ADAPTER_CLIENT_TLS               | gRPC client TLS                                                  | false                 |
| MF_MQTT_ADAPTER_CA_CERTS                 | CA certs for gRPC client TLS                                     | ""                    |
| MF_MQTT_ADAPTER_INSTANCE                 | Instance name for event sourcing                                 | ""                    |
| MF_MQTT_ADAPTER_ES_URL                   | Event sourcing URL                                               | localhost:6379        |
| MF_MQTT_ADAPTER_ES_PASS                  | Event sourcing password                                          | ""                    |
| MF_MQTT_ADAPTER_ES_DB                    | Event sourcing database                                          | "0"                   |
| MF_AUTH_CACHE_URL                        | Auth cache URL                                                   | localhost:6379        |
| MF_AUTH_CACHE_PASS                       | Auth cache password                                              | ""                    |
| MF_AUTH_CACHE_DB                         | Auth cache database                                              | "0"                   |

## Deployment

The service itself is distributed as Docker container. Check the [`mqtt-adapter`](https://github.com/mainflux/mainflux/blob/master/docker/docker-compose.yml#L219-L243) service section in
docker-compose to see how service is deployed.

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/mainflux/mainflux

cd mainflux

# compile the mqtt
make mqtt

# copy binary to bin
make install

# set the environment variables and run the service
MF_MQTT_ADAPTER_LOG_LEVEL=[MQTT Adapter Log Level] \
MF_MQTT_ADAPTER_MQTT_PORT=[MQTT adapter MQTT port]
MF_MQTT_ADAPTER_MQTT_TARGET_HOST=[MQTT broker host] \
MF_MQTT_ADAPTER_MQTT_TARGET_PORT=[MQTT broker MQTT port]] \
MF_MQTT_ADAPTER_MQTT_TARGET_HEALTH_CHECK=[MQTT health check URL] \
MF_MQTT_ADAPTER_WS_PORT=[MQTT adapter WS port] \
MF_MQTT_ADAPTER_WS_TARGET_HOST=[MQTT broker for MQTT over WS host] \
MF_MQTT_ADAPTER_WS_TARGET_PORT=[MQTT broker for MQTT over WS port]] \
MF_MQTT_ADAPTER_WS_TARGET_PATH=[MQTT adapter WS path] \
MF_MQTT_ADAPTER_FORWARDER_TIMEOUT=[MQTT forwarder for multiprotocol support timeout] \
MF_BROKER_URL=[Message broker instance URL] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout in seconds] \
MF_JAEGER_URL=[Jaeger service URL] \
MF_MQTT_ADAPTER_CLIENT_TLS=[gRPC client TLS] \
MF_MQTT_ADAPTER_CA_CERTS=[CA certs for gRPC client] \
MF_MQTT_ADAPTER_INSTANCE=[Instance for event sourcing] \
MF_MQTT_ADAPTER_ES_URL=[Event sourcing URL] \
MF_MQTT_ADAPTER_ES_PASS=[Event sourcing pass] \
MF_MQTT_ADAPTER_ES_DB=[Event sourcing database] \
MF_AUTH_CACHE_URL=[Auth cache URL] \
MF_AUTH_CACHE_PASS=[Auth cache pass] \
MF_AUTH_CACHE_DB=[Auth cache DB name] \
$GOBIN/mainflux-mqtt
```

For more information about service capabilities and its usage, please check out the API documentation [API](https://github.com/mainflux/mainflux/blob/master/api/mqtt.yml).
