# MQTT adapter

MQTT adapter provides an MQTT API for sending messages through the platform.
MQTT adapter uses [mProxy](https://github.com/mainflux/mproxy) for proxying
traffic between client and MQTT broker.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                                 | Description                                            | Default                          |
| ---------------------------------------- | ------------------------------------------------------ | -------------------------------- |
| MG_MQTT_ADAPTER_LOG_LEVEL                | mProxy Log level                                       | info                             |
| MG_MQTT_ADAPTER_MQTT_PORT                | mProxy port                                            | 1883                             |
| MG_MQTT_ADAPTER_MQTT_TARGET_HOST         | MQTT broker host                                       | 0.0.0.0                          |
| MG_MQTT_ADAPTER_MQTT_TARGET_PORT         | MQTT broker port                                       | 1883                             |
| MG_MQTT_ADAPTER_MQTT_QOS                 | MQTT broker QoS                                        | 1                                |
| MG_MQTT_ADAPTER_FORWARDER_TIMEOUT        | MQTT forwarder for multiprotocol communication timeout | 30s                              |
| MG_MQTT_ADAPTER_MQTT_TARGET_HEALTH_CHECK | URL of broker health check                             | ""                               |
| MG_MQTT_ADAPTER_WS_PORT                  | mProxy MQTT over WS port                               | 8080                             |
| MG_MQTT_ADAPTER_WS_TARGET_HOST           | MQTT broker host for MQTT over WS                      | localhost                        |
| MG_MQTT_ADAPTER_WS_TARGET_PORT           | MQTT broker port for MQTT over WS                      | 8080                             |
| MG_MQTT_ADAPTER_WS_TARGET_PATH           | MQTT broker MQTT over WS path                          | /mqtt                            |
| MG_MQTT_ADAPTER_INSTANCE                 | Instance name for event sourcing                       | ""                               |
| MG_THINGS_AUTH_GRPC_URL                  | Things gRPC endpoint URL                               | localhost:7000                   |
| MG_THINGS_AUTH_GRPC_TIMEOUT              | Timeout in seconds for Things service gRPC calls       | 1s                               |
| MG_THINGS_AUTH_GRPC_CLIENT_TLS           | Enable TLS for Things service gRPC calls               | false                            |
| MG_THINGS_AUTH_GRPC_CA_CERTS             | CA certs for Things service gRPC calls                 | ""                               |
| MG_MQTT_ADAPTER_ES_URL                   | Event sourcing URL                                     | localhost:6379                   |
| MG_MQTT_ADAPTER_ES_PASS                  | Event sourcing password                                | ""                               |
| MG_MQTT_ADAPTER_ES_DB                    | Event sourcing database                                | "0"                              |
| MG_MESSAGE_BROKER_URL                    | Message broker broker URL                              | nats://127.0.0.1:4222            |
| MG_JAEGER_URL                            | URL of Jaeger tracing service                          | "http://jaeger:14268/api/traces" |
| MG_SEND_TELEMETRY                        | Send telemetry to magistrala call home server          | true                             |
| MG_MQTT_ADAPTER_INSTANCE_ID              | Instance ID for telemetry                              | ""                               |

## Deployment

The service itself is distributed as Docker container. Check the [`mqtt-adapter`](https://github.com/absmach/magistrala/blob/master/docker/docker-compose.yml#L219-L243) service section in
docker-compose to see how service is deployed.

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/magistrala

cd magistrala

# compile the mqtt
make mqtt

# copy binary to bin
make install

# set the environment variables and run the service
MG_MQTT_ADAPTER_LOG_LEVEL=[MQTT Adapter Log Level] \
MG_MQTT_ADAPTER_MQTT_PORT=[MQTT adapter MQTT port]
MG_MQTT_ADAPTER_MQTT_TARGET_HOST=[MQTT broker host] \
MG_MQTT_ADAPTER_MQTT_TARGET_PORT=[MQTT broker MQTT port]] \
MG_MQTT_ADAPTER_FORWARDER_TIMEOUT=[MQTT forwarder for multiprotocol support timeout] \
MG_MQTT_ADAPTER_MQTT_TARGET_HEALTH_CHECK=[MQTT health check URL] \
MG_MQTT_ADAPTER_MQTT_QOS=[MQTT broker QoS] \
MG_MQTT_ADAPTER_WS_PORT=[MQTT adapter WS port] \
MG_MQTT_ADAPTER_WS_TARGET_HOST=[MQTT broker for MQTT over WS host] \
MG_MQTT_ADAPTER_WS_TARGET_PORT=[MQTT broker for MQTT over WS port]] \
MG_MQTT_ADAPTER_WS_TARGET_PATH=[MQTT adapter WS path] \
MG_MQTT_ADAPTER_INSTANCE=[Instance for event sourcing] \
MG_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MG_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout in seconds] \
MG_THINGS_AUTH_GRPC_CLIENT_TLS=[gRPC client TLS] \
MG_THINGS_AUTH_GRPC_CA_CERTS=[CA certs for gRPC client] \
MG_MQTT_ADAPTER_CLIENT_TLS=[gRPC client TLS] \
MG_MQTT_ADAPTER_CA_CERTS=[CA certs for gRPC client] \
MG_MQTT_ADAPTER_ES_URL=[Event sourcing URL] \
MG_MQTT_ADAPTER_ES_PASS=[Event sourcing pass] \
MG_MQTT_ADAPTER_ES_DB=[Event sourcing database] \
MG_MESSAGE_BROKER_URL=[Message broker instance URL] \
MG_JAEGER_URL=[Jaeger service URL] \
MG_SEND_TELEMETRY=[Send telemetry to magistrala call home server] \
MG_MQTT_ADAPTER_INSTANCE_ID=[Instance ID] \
$GOBIN/magistrala-mqtt
```

For more information about service capabilities and its usage, please check out the API documentation [API](https://github.com/absmach/magistrala/blob/master/api/mqtt.yml).
