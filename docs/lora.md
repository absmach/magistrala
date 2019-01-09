Before running Mainflux `lora-adapter` you must install and run [LoRa Server](https://www.loraserver.io/loraserver/overview). First, execute the following command: 

```bash
go get github.com/brocaar/loraserver-docker
```

Once everything is installed, execute the following command from the LoRa Server project root:

```bash
docker-compose up
```

The Mainflux lora-adapter can do bridging between both systems. Basically, the service subscribe to the [LoRa Gateway Bridge](https://www.loraserver.io/lora-gateway-bridge/overview/), a service which abstracts the [SemTech packet-forwarder UDP protocol](https://github.com/Lora-net/packet_forwarder/blob/master/PROTOCOL.TXT) into JSON over MQTT.

You must configure the `MF_LORA_ADAPTER_LORA_MESSAGE_URL` variable in the [lora-adapter docker-compose.yml](https://github.com/mainflux/mainflux/blob/master/docker/addons/lora-adapter/docker-compose.yml) with the address of your LoRa Gateway Bridge, otherwise the composition will fail:

```bash
docker-compose -f docker/addons/lora-adapter/docker-compose up
```

At this point Mainflux and LoRa Server are running. To provision the LoRa Server with Networks, Organizations, Gateways, Applications and Devices you have to implement the gRPC API. Over the [LoRa App Server](https://www.loraserver.io/lora-app-server/overview), which is good example of the gRPC API implementation, you can do it as well.
 
#### LoRa Server setup

- **Create Organization:** To add your own Gateways to the network you must have an Organization.
- **Add Network LoRa Server:** Set the address of your Network-Server API that is used by LoRa App Server or other custom components interacting with LoRa Server (by default loraserver:8000).
- **Create a Gateways-Profile:** In this profile you can select the radio LoRa channels and the LoRa Network Server to use.
- **Create a Service-profile:** A service-profile connects an organization to a network-server and defines the features that an organization can use on this Network-Server.
- **Create a Gateway:** You must set proper ID in order to be discovered by LoRa Server.
- **Create a LoRa Server Application:** You can then create Devices by connecting them to this application. This is equivalent to Devices connected to channels in Mainflux.
- **Create a Device-Profile:** Before creating Device you must create Device profile where you will define some parameter as LoRaWAN MAC version (format of the device address) and the LoRaWAN regional parameter (frequency band). This will allow you to create many devices using this profile.
- **Create a Device:** Then you can create a Device. You must configure the `network session key` and `application session key` of your Device. You can generate and copy them on your device configuration or you can use your own pre generated keys and set them using the [Lora App Server](https://www.loraserver.io/lora-app-server/overview/).
Device connect through OTAA. Make sure that loraserver device-profile is using same release as device. If MAC version is 1.0.X, `application key = app_key` and `app_eui = deviceEUI`. If MAC version is 1.1 or ABP both parameters will be needed, APP_key and Network key.

#### Connect Mainflux and LoRa Server with lora-adapter
 
This adapter sits between Mainflux and LoRa Server and forwards the MQTT messages from the [LoRa Gateways Bridge](https://www.loraserver.io/lora-gateway-bridge/overview) to the Mainflux multi-protocol message broker, using the adequate MQTT topics and in the good message format (JSON and SenML), i.e. respecting the APIs of both systems.
 
Once everything is running and the LoRa Server is provisioned, execute the following command from Mainflux project root to run the lora-adapter:
 
```bash
docker-compose -f docker/addons/lora-adapter/docker-compose.yml up -d
```

This service uses RedisDB to create a route map between both systems. As in Mainflux we use Channels to connect Things, LoRa Server uses Applications to connect Devices. There are then two routes, 
The lora-adapter uses the matadata of provision events emitted by mainflux-things service to update the route map. For that, you must provision Mainflux Channels and Things with an extra metadata key in the JSON Body of the HTTP request. It must be a JSON object with keys `type` and `appID` or `devEUI`. In this case `type` must be `lora`, `appID` and `devEUI` must be an existent Lora application ID and device EUI:

**Channel structure:**

```
{
  "name": "<channel name>",
  "metadata:":{
    "type": "lora",
    "appID": "<application ID>"
  }
}
```

**Thing structure:**

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

To forward LoRa messages the lora-adapter subscribes to topics `applications/+/devices/+` of the [LoRa Gateway Bridge](https://www.loraserver.io/lora-gateway-bridge/overview). It verifies `appID` and `devEUI` of published messages. If the mapping exists it uses corresponding `channelID` and `thingID` to sign and forwards the content of the LoRa message to the Mainflux message broker.
For more information about service capabilities and its usage, please check out the API documentation.