# LoRa Adapter
Adapter between Mainflux IoT system and [LoRa Server](https://github.com/brocaar/loraserver).

This adapter sits between Mainflux and LoRa server and just forwards the messages form one system to another via MQTT protocol, using the adequate MQTT topics and in the good message format (JSON and SenML), i.e. respecting the APIs of both systems.

LoRa Server is used for connectivity layer and data is pushed via this adapter service to Mainflux, where it is persisted and routed to other protocols via Mainflux multi-protocol message broker. Mainflux adds user accounts, application management and security in order to obtain the overall end-to-end LoRa solution.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                         | Description                           | Default               |
|----------------------------------|---------------------------------------|-----------------------|
| MF_LORA_ADAPTER_HTTP_PORT        | Service HTTP port                     | 8180                  |
| MF_LORA_ADAPTER_LOG_LEVEL        | Log level for the Lora Adapter        | error                 |
| MF_NATS_URL                      | NATS instance URL                     | nats://localhost:4222 |
| MF_LORA_ADAPTER_LORA_MESSAGE_URL | LoRa Server mqtt broker URL           | tcp://localhost:1883  |
| MF_LORA_ADAPTER_ROUTEMAP_URL     | Routemap database URL                 | localhost:6379        |
| MF_LORA_ADAPTER_ROUTEMAP_PASS    | Routemap database password            |                       |
| MF_LORA_ADAPTER_ROUTEMAP_DB      | Routemap instance that should be used | 0                     |
| MF_THINGS_ES_URL                 | Things service event store URL        | localhost:6379        |
| MF_THINGS_ES_PASS                | Things service event store password   |                       |
| MF_THINGS_ES_DB                  | Things service event store db         | 0                     |
| MF_LORA_ADAPTER_INSTANCE_NAME    | LoRa adapter instance name            | lora                  |

## Deployment

The service is distributed as Docker container. The following snippet provides
a compose file template that can be used to deploy the service container locally:

```yaml
version: "2"
services:
  adapter:
    image: mainflux/lora:[version]
    container_name: [instance name]
    environment:
      MF_LORA_ADAPTER_LOG_LEVEL: [Lora Adapter Log Level]
      MF_NATS_URL: [NATS instance URL]
      MF_LORA_ADAPTER_LORA_MESSAGE_URL: [LoRa Server mqtt broker URL]
      MF_LORA_ADAPTER_ROUTEMAP_URL: [Lora adapter routemap URL]
      MF_LORA_ADAPTER_ROUTEMAP_PASS: [Lora adapter routemap password]
      MF_LORA_ADAPTER_ROUTEMAP_DB: [Lora adapter routemap instance]
      MF_THINGS_ES_URL: [Things service event store URL]
      MF_THINGS_ES_PASS: [Things service event store password]
      MF_THINGS_ES_DB: [Things service event store db]
      MF_LORA_ADAPTER_INSTANCE_NAME: [LoRa adapter instance name]
```

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
go get github.com/mainflux/mainflux

cd $GOPATH/src/github.com/mainflux/mainflux

# compile the lora adapter
make lora

# copy binary to bin
make install

# set the environment variables and run the service
MF_LORA_ADAPTER_LOG_LEVEL=[Lora Adapter Log Level] MF_NATS_URL=[NATS instance URL] MF_LORA_ADAPTER_LORA_MESSAGE_URL=[LoRa Server mqtt broker URL] MF_LORA_ADAPTER_ROUTEMAP_URL=[Lora adapter routemap URL] MF_LORA_ADAPTER_ROUTEMAP_PASS=[Lora adapter routemap password] MF_LORA_ADAPTER_ROUTEMAP_DB=[Lora adapter routemap instance] MF_THINGS_ES_URL=[Things service event store URL] MF_THINGS_ES_PASS=[Things service event store password] MF_THINGS_ES_DB=[Things service event store db] MF_LORA_ADAPTER_INSTANCE_NAME=[LoRa adapter instance name] $GOBIN/mainflux-lora
```

### Using docker-compose

This service can be deployed using docker containers.
Docker compose file is available in `<project_root>/docker/addons/lora-adapter/docker-compose.yml`. In order to run Mainflux lora-adapter, execute the following command:

```bash
docker-compose -f docker/addons/lora-adapter/docker-compose.yml up -d
```

## Usage

First of all you must run your LoRa Server. You need to clone the repository with command `go get github.com/brocaar/loraserver-docker` and deploy docker containers with command `docker-compose up` from `$GOPATH/src/github.com/brocaar/loraserver-docker` repository.
Then you must provision your system. Basically it means create a Network-server where to connect your Gateways and Applications where to connect Devices. If you need more information about the LoRa Server setup you can check [the LoRa App Server documentation](https://www.loraserver.io/lora-app-server/overview/)

Once you are done with the LoRa Server setup you can run the lora-adapter. This service uses RedisDB to create a route map between both systems. As in Mainflux we use Channels to connect Things, Loraserser uses Applications to connect Devices. Route map connects applicationsID with channelID and deviceEUI with thingID.
The lora-adapter uses the matadata of provision events emitted by mainflux-things service to update the route map.
For that, you must provision Mainflux Channels and Things with an extra `metadata` key in the JSON Body of the HTTP request. It must be a JSON object with keys `type` and `appID` or `devEUI`. Obviously `type` must be `lora`, `appID` and `devEUI` must be an existent Lora application ID and device EUI.

```
{
  "name": "<channel name>",
  "metadata:":{
    "type": "lora",
    "appID": "<application ID>"
  }
}

```
```
{
  "type": "device",
  "name": "<thing name>",
  "metadata:":{
    "type": "lora",
    "devEUI": "<device EUI>"
  }
}
```

To receive Lora messages the lora-adapter subscribes to topic `applications/+/devices/+` of the LoRa Server MQTT broker. The [LoRa Gateway Bridge](https://www.loraserver.io/lora-gateway-bridge/overview/)uses the same topic to publish decoded messages received from gateways as UDP packets. The lora-adapter verify the applicationID and the deviceEUI of published message and if they are known it forwards the message on the Mainflux NATS broker as corresponding channel and thing.


For more information about service capabilities and its usage, please check out
the [API documentation](swagger.yaml).
