# LoRa Adapter

Adapter between Magistrala IoT system and [LoRa Server](https://github.com/brocaar/chirpstack-network-server).

This adapter sits between Magistrala and LoRa Server and just forwards the messages from one system to another via MQTT protocol, using the adequate MQTT topics and in the good message format (JSON and SenML), i.e. respecting the APIs of both systems.

LoRa Server is used for connectivity layer and data is pushed via this adapter service to Magistrala, where it is persisted and routed to other protocols via Magistrala multi-protocol message broker. Magistrala adds user accounts, application management and security in order to obtain the overall end-to-end LoRa solution.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                         | Description                                 | Default                         |
| -------------------------------- | ------------------------------------------- | ------------------------------- |
| MG_LORA_ADAPTER_HTTP_PORT        | Service HTTP port                           | 9017                            |
| MG_LORA_ADAPTER_LOG_LEVEL        | Service Log level                           | info                            |
| MG_MESSAGE_BROKER_URL            | Message broker instance URL                 | nats://localhost:4222           |
| MG_LORA_ADAPTER_MESSAGES_URL     | LoRa adapter MQTT broker URL                | tcp://localhost:1883            |
| MG_LORA_ADAPTER_MESSAGES_TOPIC   | LoRa adapter MQTT subscriber Topic          | application/+/device/+/event/up |
| MG_LORA_ADAPTER_MESSAGES_USER    | LoRa adapter MQTT subscriber Username       |                                 |
| MG_LORA_ADAPTER_MESSAGES_PASS    | LoRa adapter MQTT subscriber Password       |                                 |
| MG_LORA_ADAPTER_MESSAGES_TIMEOUT | LoRa adapter MQTT subscriber Timeout        | 30s                             |
| MG_LORA_ADAPTER_ROUTE_MAP_URL    | Route-map database URL                      | redis://localhost:6379          |
| MG_THINGS_ES_URL                 | Things service event source URL             | localhost:6379                  |
| MG_THINGS_ES_PASS                | Things service event source password        |                                 |
| MG_THINGS_ES_DB                  | Things service event source DB              | 0                               |
| MG_LORA_ADAPTER_EVENT_CONSUMER   | Service event consumer name                 | lora                            |
| MG_JAEGER_URL                    | Jaeger server URL                           | http://jaeger:14268/api/traces  |
| MG_SEND_TELEMETRY                | Send telemetry to magistrala call home server | true                            |

## Deployment

The service itself is distributed as Docker container. Check the [`lora-adapter`](https://github.com/absmach/magistrala/blob/master/docker/addons/lora-adapter/docker-compose.yml#L23-L37) service section in
docker-compose to see how service is deployed.

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
MG_LORA_ADAPTER_LOG_LEVEL=[Lora Adapter Log Level] \
MG_MESSAGE_BROKER_URL=[Message broker instance URL] \
MG_LORA_ADAPTER_MESSAGES_URL=[LoRa adapter MQTT broker URL] \
MG_LORA_ADAPTER_MESSAGES_TOPIC=[LoRa adapter MQTT subscriber Topic] \
MG_LORA_ADAPTER_MESSAGES_USER=[LoRa adapter MQTT subscriber Username] \
MG_LORA_ADAPTER_MESSAGES_PASS=[LoRa adapter MQTT subscriber Password] \
MG_LORA_ADAPTER_MESSAGES_TIMEOUT=[LoRa adapter MQTT subscriber Timeout]
MG_LORA_ADAPTER_ROUTE_MAP_URL=[Lora adapter routemap URL] \
MG_THINGS_ES_URL=[Things service event source URL] \
MG_THINGS_ES_PASS=[Things service event source password] \
MG_THINGS_ES_DB=[Things service event source password] \
MG_OPCUA_ADAPTER_EVENT_CONSUMER=[LoRa adapter instance name] \
$GOBIN/magistrala-lora
```

### Using docker-compose

This service can be deployed using docker containers.
Docker compose file is available in `<project_root>/docker/addons/lora-adapter/docker-compose.yml`. In order to run Magistrala lora-adapter, execute the following command:

```bash
docker-compose -f docker/addons/lora-adapter/docker-compose.yml up -d
```

## Usage

For more information about service capabilities and its usage, please check out
the [Magistrala documentation](https://docs.mainflux.io/lora).
