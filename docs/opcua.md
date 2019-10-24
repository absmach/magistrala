Bridging with an OPC-UA Server can be done over the [opcua-adapter](https://github.com/mainflux/mainflux/tree/master/opcua). This service sits between Mainflux and an [OPC-UA Server](https://en.wikipedia.org/wiki/OPC_Unified_Architecture) and just forwards the messages from one system to another.

## Run OPC-UA Server

The OPC-UA Server is used for connectivity layer. It allows various methods to read informations from the OPC-UA server and its nodes. The current version of the opcua-adapter still experimental and only `Read` and `Subscribe` methods are implemented.
[Public OPC-UA test servers](https://github.com/node-opcua/node-opcua/wiki/publicly-available-OPC-UA-Servers-and-Clients) are available for testing of OPC-UA clients and can be used for development and test purposes.

## Mainflux and OPC-UA Server

Once the OPC-UA Server you want to connect is running, execute the following command from Mainflux project root to run the opcua-adapter:

```bash
docker-compose -f docker/addons/opcua-adapter/docker-compose.yml up -d
```

**Remark:** By defaut, `MF_OPCUA_ADAPTER_SERVER_URI` is set as `opc.tcp://opcua.rocks:4840` in the [docker-compose.yml](https://github.com/mainflux/mainflux/blob/master/docker/addons/opcua-adapter/docker-compose.yml) file of the adapter. If you run the composition without configure this variable you will start to receive messages from the public test server [OPC UA rocks](https://opcua.rocks/open62541-online-test-server).

### Route Map

The opcua-adapter use [Redis](https://redis.io/) database to create a route-map between Mainflux and an OPC-UA Server. As Mainflux use Things and Channels IDs to sign messages, OPC-UA servers use Node Namespaces and Node Identifiers (the combination is called NodeID). The adapter route-mmap associate a `ThingID` with a `Node Identifier` and a `ChannelID` with a `Node Namespace`

The opcua-adapter uses the matadata of provision events emitted by Mainflux system to update its route map. For that, you must provision Mainflux `Channels` and `Things` with an extra metadata key in the JSON Body of the HTTP request. It must be a JSON object with key `opcua` which value is another JSON object. This nested JSON object should contain `namespace` or `id` field. In this case `namespace` or `id` must be an existent OPC-UA `Node Namespace` or `Node Identifier`:

**Channel structure:**

```
{
  "name": "<channel name>",
  "metadata:": {
    "opcua": {
      "namespace": "<Node Namespace>"
    }
  }
}
```

**Thing structure:**

```
{
  "type": "device",
  "name": "<thing name>",
  "metadata:": {
    "opcua": {
      "id": "<Node Identifier>"
    }
  }
}
```

##### Messaging

To forward OPC-UA messages the opcua-adapter subscribes to the NodeID `ns=<namespace>;i=<identifier>` of the OPC-UA Server. It verifies `namespace` and `id` of published messages. If the mapping exists it uses corresponding `ChannelID` and `ThingID` to sign and forwards the content of the OPC-UA message to the Mainflux message broker.
