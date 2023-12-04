# LoRa Adapter

Adapter between Magistrala IoT system and [LoRa Server](https://github.com/brocaar/chirpstack-network-server).

This adapter sits between Magistrala and LoRa Server and just forwards the messages from one system to another via MQTT protocol, using the adequate MQTT topics and in the good message format (JSON and SenML), i.e. respecting the APIs of both systems.

LoRa Server is used for connectivity layer and data is pushed via this adapter service to Magistrala, where it is persisted and routed to other protocols via Magistrala multi-protocol message broker. Magistrala adds user accounts, application management and security in order to obtain the overall end-to-end LoRa solution.

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                         | Description                                               | Default                             |
| -------------------------------- | --------------------------------------------------------- | ----------------------------------- |
| MG_LORA_ADAPTER_LOG_LEVEL        | Log level for the LoRa Adapter (debug, info, warn, error) | info                                |
| MG_LORA_ADAPTER_HTTP_HOST        | Service LoRa host                                         | ""                                  |
| MG_LORA_ADAPTER_HTTP_PORT        | Service LoRa port                                         | 9017                                |
| MG_LORA_ADAPTER_HTTP_SERVER_CERT | Path to the PEM encoded server certificate file           | ""                                  |
| MG_LORA_ADAPTER_HTTP_SERVER_KEY  | Path to the PEM encoded server key file                   | ""                                  |
| MG_LORA_ADAPTER_MESSAGES_URL     | LoRa adapter MQTT broker URL                              | tcp://localhost:1883                |
| MG_LORA_ADAPTER_MESSAGES_TOPIC   | LoRa adapter MQTT subscriber Topic                        | application/+/device/+/event/up     |
| MG_LORA_ADAPTER_MESSAGES_USER    | LoRa adapter MQTT subscriber Username                     | ""                                  |
| MG_LORA_ADAPTER_MESSAGES_PASS    | LoRa adapter MQTT subscriber Password                     | ""                                  |
| MG_LORA_ADAPTER_MESSAGES_TIMEOUT | LoRa adapter MQTT subscriber Timeout                      | 30s                                 |
| MG_LORA_ADAPTER_ROUTE_MAP_URL    | Route-map database URL                                    | redis://localhost:6379              |
| MG_ES_URL                        | Event source URL                                          | <nats://localhost:4222>             |
| MG_LORA_ADAPTER_EVENT_CONSUMER   | Service event consumer name                               | lora-adapter                        |
| MG_MESSAGE_BROKER_URL            | Message broker instance URL                               | <nats://localhost:4222>             |
| MG_JAEGER_URL                    | Jaeger server URL                                         | <http://localhost:14268/api/traces> |
| MG_JAEGER_TRACE_RATIO            | Jaeger sampling ratio                                     | 1.0                                 |
| MG_SEND_TELEMETRY                | Send telemetry to magistrala call home server             | true                                |
| MG_LORA_ADAPTER_INSTANCE_ID      | Service instance ID                                       | ""                                  |

## Deployment

The service itself is distributed as Docker container. Check the [`lora-adapter`](https://github.com/absmach/magistrala/blob/main/docker/addons/lora-adapter/docker-compose.yml) service section in docker-compose to see how service is deployed.

Running this service outside of container requires working instance of the message broker service, LoRa server, things service and Jaeger server.
To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/magistrala

cd magistrala

# compile the lora adapter
make lora

# copy binary to bin
make install

# set the environment variables and run the service
MG_LORA_ADAPTER_LOG_LEVEL=info \
MG_LORA_ADAPTER_HTTP_HOST=localhost \
MG_LORA_ADAPTER_HTTP_PORT=9017 \
MG_LORA_ADAPTER_HTTP_SERVER_CERT="" \
MG_LORA_ADAPTER_HTTP_SERVER_KEY="" \
MG_LORA_ADAPTER_MESSAGES_URL=tcp://localhost:1883 \
MG_LORA_ADAPTER_MESSAGES_TOPIC=application/+/device/+/event/up \
MG_LORA_ADAPTER_MESSAGES_USER="" \
MG_LORA_ADAPTER_MESSAGES_PASS="" \
MG_LORA_ADAPTER_MESSAGES_TIMEOUT=30s \
MG_LORA_ADAPTER_ROUTE_MAP_URL=redis://localhost:6379 \
MG_ES_URL=nats://localhost:4222 \
MG_LORA_ADAPTER_EVENT_CONSUMER=lora-adapter \
MG_MESSAGE_BROKER_URL=nats://localhost:4222 \
MG_JAEGER_URL=http://localhost:14268/api/traces \
MG_JAEGER_TRACE_RATIO=1.0 \
MG_SEND_TELEMETRY=true \
MG_LORA_ADAPTER_INSTANCE_ID="" \
$GOBIN/magistrala-lora
```

Setting `MG_LORA_ADAPTER_HTTP_SERVER_CERT` and `MG_LORA_ADAPTER_HTTP_SERVER_KEY` will enable TLS against the service. The service expects a file in PEM format for both the certificate and the key.

### Using docker-compose

This service can be deployed using docker containers. Docker compose file is available in `<project_root>/docker/addons/lora-adapter/docker-compose.yml`. In order to run Magistrala lora-adapter, execute the following command:

```bash
docker-compose -f docker/addons/lora-adapter/docker-compose.yml up -d
```

## Usage

For more information about service capabilities and its usage, please check out the [Magistrala documentation](https://docs.mainflux.io/lora).
