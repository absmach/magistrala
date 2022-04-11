# LoRa Adapter
Adapter between Mainflux IoT system and [LoRa Server](https://github.com/brocaar/chirpstack-network-server).

This adapter sits between Mainflux and LoRa Server and just forwards the messages from one system to another via MQTT protocol, using the adequate MQTT topics and in the good message format (JSON and SenML), i.e. respecting the APIs of both systems.

LoRa Server is used for connectivity layer and data is pushed via this adapter service to Mainflux, where it is persisted and routed to other protocols via Mainflux multi-protocol message broker. Mainflux adds user accounts, application management and security in order to obtain the overall end-to-end LoRa solution.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                         | Description                           | Default                         |
|----------------------------------|---------------------------------------|---------------------------------|
| MF_LORA_ADAPTER_HTTP_PORT        | Service HTTP port                     | 8180                            |
| MF_LORA_ADAPTER_LOG_LEVEL        | Service Log level                     | error                           |
| MF_NATS_URL                      | NATS instance URL                     | nats://localhost:4222           |
| MF_LORA_ADAPTER_MESSAGES_URL     | LoRa adapter MQTT broker URL          | tcp://localhost:1883            |
| MF_LORA_ADAPTER_MESSAGES_TOPIC   | LoRa adapter MQTT subscriber Topic    | application/+/device/+/event/up |
| MF_LORA_ADAPTER_MESSAGES_USER    | LoRa adapter MQTT subscriber Username |                                 |
| MF_LORA_ADAPTER_MESSAGES_PASS    | LoRa adapter MQTT subscriber Password |                                 |
| MF_LORA_ADAPTER_MESSAGES_TIMEOUT | LoRa adapter MQTT subscriber Timeout  | 30s                             |
| MF_LORA_ADAPTER_ROUTE_MAP_URL    | Route-map database URL                | localhost:6379                  |
| MF_LORA_ADAPTER_ROUTE_MAP_PASS   | Route-map database password           |                                 |
| MF_LORA_ADAPTER_ROUTE_MAP_DB     | Route-map instance                    | 0                               |
| MF_THINGS_ES_URL                 | Things service event source URL       | localhost:6379                  |
| MF_THINGS_ES_PASS                | Things service event source password  |                                 |
| MF_THINGS_ES_DB                  | Things service event source DB        | 0                               |
| MF_LORA_ADAPTER_EVENT_CONSUMER   | Service event consumer name           | lora                            |

## Deployment

The service itself is distributed as Docker container. Check the [`lora-adapter`](https://github.com/mainflux/mainflux/blob/master/docker/addons/lora-adapter/docker-compose.yml#L23-L37) service section in
docker-compose to see how service is deployed.

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/mainflux/mainflux

cd mainflux

# compile the lora adapter
make lora

# copy binary to bin
make install

# set the environment variables and run the service
MF_LORA_ADAPTER_LOG_LEVEL=[Lora Adapter Log Level] \
MF_NATS_URL=[NATS instance URL] \
MF_LORA_ADAPTER_MESSAGES_URL=[LoRa adapter MQTT broker URL] \
MF_LORA_ADAPTER_MESSAGES_TOPIC=[LoRa adapter MQTT subscriber Topic] \
MF_LORA_ADAPTER_MESSAGES_USER=[LoRa adapter MQTT subscriber Username] \
MF_LORA_ADAPTER_MESSAGES_PASS=[LoRa adapter MQTT subscriber Password] \
MF_LORA_ADAPTER_MESSAGES_TIMEOUT=[LoRa adapter MQTT subscriber Timeout]
MF_LORA_ADAPTER_ROUTE_MAP_URL=[Lora adapter routemap URL] \
MF_LORA_ADAPTER_ROUTE_MAP_PASS=[Lora adapter routemap password] \
MF_LORA_ADAPTER_ROUTE_MAP_DB=[Lora adapter routemap instance] \
MF_THINGS_ES_URL=[Things service event source URL] \
MF_THINGS_ES_PASS=[Things service event source password] \
MF_THINGS_ES_DB=[Things service event source password] \
MF_OPCUA_ADAPTER_EVENT_CONSUMER=[LoRa adapter instance name] \
$GOBIN/mainflux-lora
```

### Using docker-compose

This service can be deployed using docker containers.
Docker compose file is available in `<project_root>/docker/addons/lora-adapter/docker-compose.yml`. In order to run Mainflux lora-adapter, execute the following command:

```bash
docker-compose -f docker/addons/lora-adapter/docker-compose.yml up -d
```

## Usage

For more information about service capabilities and its usage, please check out
the [Mainflux documentation](https://docs.mainflux.io/lora).
