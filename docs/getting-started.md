## Prerequisites

Before proceeding, install the following prerequisites:

- [Docker](https://docs.docker.com/install/)
- [Docker compose](https://docs.docker.com/compose/install/)
- [jsonpp](https://jmhodges.github.io/jsonpp/) (optional)

Once everything is installed, execute the following commands from project root:

```bash
docker-compose -f docker/docker-compose.yml up -d
```

## User management

### Account creation

Use the Mainflux API to create user account:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X POST -H "Content-Type: application/json" https://localhost/users -d '{"email":"john.doe@email.com", "password":"123"}'
```

Note that when using official `docker-compose`, all services are behind `nginx`
proxy and all traffic is `TLS` encrypted.

### Obtaining an authorization key

In order for this user to be able to authenticate to the system, you will have
to create an authorization token for him:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X POST -H "Content-Type: application/json" https://localhost/tokens -d '{"email":"john.doe@email.com", "password":"123"}'
```

Response should look like this:
```
{
      "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1MjMzODg0NzcsImlhdCI6MTUyMzM1MjQ3NywiaXNzIjoibWFpbmZsdXgiLCJzdWIiOiJqb2huLmRvZUBlbWFpbC5jb20ifQ.cygz9zoqD7Rd8f88hpQNilTCAS1DrLLgLg4PRcH-iAI"
}
```

## System provisioning

Before proceeding, make sure that you have created a new account, and obtained
an authorization key.

### Provisioning devices

Devices are provisioned by executing request `POST /things`, with a
`"type":"device"` specified in JSON payload. Note that you will also need
`user_auth_token` in order to provision things (both devices and application)
that belong to this particular user.

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X POST -H "Content-Type: application/json" -H "Authorization: <user_auth_token>" https://localhost/things -d '{"type":"device", "name":"weio"}'
```

Response will contain `Location` header whose value represents path to newly
created thing:

```
HTTP/1.1 201 Created
Content-Type: application/json
Location: /things/81380742-7116-4f6f-9800-14fe464f6773
Date: Tue, 10 Apr 2018 10:02:59 GMT
Content-Length: 0
```

### Provisioning applications

Applications are provisioned by executing HTTP request `POST /things`, with
`"type":"app"` specified in JSON payload.

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X POST -H "Content-Type: application/json" -H "Authorization: <user_auth_token>" https://localhost/things -d '{"type":"app", "name":"myapp"}'
```

Response will contain `Location` header whose value represents path to newly
created thing (same as for devices):

```
HTTP/1.1 201 Created
Content-Type: application/json
Location: /things/cb63f852-2d48-44f0-a0cf-e450496c6c92
Date: Tue, 10 Apr 2018 10:33:17 GMT
Content-Length: 0
```

### Retrieving provisioned things

In order to retrieve data of provisioned things that is written in database, you
can send following request:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -H "Authorization: <user_auth_token>" https://localhost/things
```

Notice that you will receive only those things that were provisioned by
`user_auth_token` owner.

```
HTTP/1.1 200 OK
Content-Type: application/json
Date: Tue, 10 Apr 2018 10:50:12 GMT
Content-Length: 1105

{
  "things": [
    {
      "id": "81380742-7116-4f6f-9800-14fe464f6773",
      "type": "device",
      "name": "weio",
      "key": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpYXQiOjE1MjMzNTQ1NzksImlzcyI6Im1haW5mbHV4Iiwic3ViIjoiODEzODA3NDItNzExNi00ZjZmLTk4MDAtMTRmZTQ2NGY2NzczIn0.5s8s1hlK-l30kQAyHxEZO_M2NIQw53MQuy7b3Wf3OOE"
    },
    {
      "id": "cb63f852-2d48-44f0-a0cf-e450496c6c92",
      "type": "app",
      "name": "myapp",
      "key": "cbf02d60-72f2-4180-9f82-2c957db929d1"
    }
  ]
}
```

You can specify `offset` and `limit` parameters in order to fetch specific
group of things. In that case, your request should look like:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -H "Authorization: <user_auth_token>" https://localhost/things?offset=0&limit=5
```

If you don't provide them, default values will be used instead: 0 for `offset`,
and 10 for `limit`. Note that `limit` cannot be set to values greater than 100. Providing
invalid values will be considered malformed request.

### Removing things

In order to remove you own thing you can send following request:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X DELETE -H "Authorization: <user_auth_token>" https://localhost/things/<thing_id>
```

### Provisioning channels

Channels are provisioned by executing request `POST /channels`:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X POST -H "Content-Type: application/json" -H "Authorization: <user_auth_token>" https://localhost/channels -d '{"name":"mychan"}'
```

After sending request you should receive response with `Location` header that
contains path to newly created channel:

```
HTTP/1.1 201 Created
Content-Type: application/json
Location: /channels/19daa7a8-a489-4571-8714-ef1a214ed914
Date: Tue, 10 Apr 2018 11:30:07 GMT
Content-Length: 0
```

### Retrieving provisioned channels

To retreve provisioned channels you should send request to `/channels` with
authorization token in `Authorization` header:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -H "Authorization: <user_auth_token>" https://localhost/channels
```

Note that you will receive only those channels that were created by authorization
token's owner.

```
HTTP/1.1 200 OK
Content-Type: application/json
Date: Tue, 10 Apr 2018 11:38:06 GMT
Content-Length: 139

{
  "channels": [
    {
      "id": "19daa7a8-a489-4571-8714-ef1a214ed914",
      "name": "mychan"
    }
  ]
}
```

You can specify  `offset` and  `limit` parameters in order to fetch specific
group of channels. In that case, your request should look like:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -H "Authorization: <user_auth_token>" https://localhost/channels?offset=0&limit=5
```

If you don't provide them, default values will be used instead: 0 for `offset`,
and 10 for `limit`. Note that `limit` cannot be set to values greater than 100. Providing
invalid values will be considered malformed request.

### Removing channels

In order to remove specific channel you should send following request:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X DELETE -H "Authorization: <user_auth_token>" https://localhost/channels/<channel_id>
```

## Access control

Channel can be observed as a communication group of things. Only things that
are connected to the channel can send and receive messages from other things
in this channel. things that are not connected to this channel are not allowed
to communicate over it.

Only user, who is the owner of a channel and of the things, can connect the
things to the channel (which is equivalent of giving permissions to these things
to communicate over given communication group).

To connect thing to the channel you should send following request:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X PUT -H "Authorization: <user_auth_token>" https://localhost/channels/<channel_id>/things/<thing_id>
```

You can observe which things are connected to specific channel:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -H "Authorization: <user_auth_token>" https://localhost/channels/<channel_id>
```

You should receive response with the lists of connected things in `connected` field
similar to this one:

```
{
  "id": "19daa7a8-a489-4571-8714-ef1a214ed914",
  "name": "mychan",
  "connected": [
    {
      "id": "81380742-7116-4f6f-9800-14fe464f6773",
      "type": "device",
      "name": "weio",
      "key": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpYXQiOjE1MjMzNTQ1NzksImlzcyI6Im1haW5mbHV4Iiwic3ViIjoiODEzODA3NDItNzExNi00ZjZmLTk4MDAtMTRmZTQ2NGY2NzczIn0.5s8s1hlK-l30kQAyHxEZO_M2NIQw53MQuy7b3Wf3OOE"
    }
  ]
}
```

If you want to disconnect your device from the channel, send following request:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X DELETE -H "Authorization: <user_auth_token>" https://localhost/channels/<channel_id>/things/<thing_id>
```

## Sending messages

Once a channel is provisioned and thing is connected to it, it can start to
publish messages on the channel. The following sections will provide an example
of message publishing for each of the supported protocols.

### HTTP

To publish message over channel, thing should send following request:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X POST -H "Content-Type: application/senml+json" -H "Authorization: <thing_token>" https://localhost/http/channels/<channel_id>/messages -d '[{"bn":"some-base-name:","bt":1.276020076001e+09, "bu":"A","bver":5, "n":"voltage","u":"V","v":120.1}, {"n":"current","t":-5,"v":1.2}, {"n":"current","t":-4,"v":1.3}]'
```

Note that you should always send array of messages in senML format.

### WebSocket

To publish and receive messages over channel using web socket, you should first
send handshake request to `/channels/<channel_id>/messages` path. Don't forget
to send `Authorization` header with thing authorization token.

If you are not able to send custom headers in your handshake request, send it as
query parameter `authorization`. Then your path should look like this
`/channels/<channel_id>/messages?authorization=<thing_auth_key>`.

If you are using the docker environment prepend the url with `ws`. So for example
`/ws/channels/<channel_id>/messages?authorization=<thing_auth_key>`

#### Basic nodejs example

```javascript
const WebSocket = require('ws');

// do not verify self-signed certificates if you are using one
process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0'

// cbf02d60-72f2-4180-9f82-2c957db929d1  is an example of a thing_auth_key
const ws = new WebSocket('wss://localhost/ws/channels/1/messages?authorization=cbf02d60-72f2-4180-9f82-2c957db929d1')

ws.on('open', () => {
    ws.send('something')
})

ws.on('message', (data) => {
    console.log(data)
})
ws.on('error', (e) => {
    console.log(e)
})
```

### MQTT

To send and receive messages over MQTT you could use [Mosquitto tools](https://mosquitto.org),
or [Paho](https://www.eclipse.org/paho/) if you want to use MQTT over WebSocket.

To publish message over channel, thing should call following command:

```
mosquitto_pub -u <thing_id> -P <thing_key> -t channels/<channel_id>/messages -h localhost -m '[{"bn":"some-base-name:","bt":1.276020076001e+09, "bu":"A","bver":5, "n":"voltage","u":"V","v":120.1}, {"n":"current","t":-5,"v":1.2}, {"n":"current","t":-4,"v":1.3}]'
```

To subscribe to channel, thing should call following command:

```
mosquitto_sub -u <thing_id> -P <thing_key> -t channels/<channel_id>/messages -h localhost
```

If you are using TLS to secure MQTT connection, add `--cafile docker/ssl/certs/ca.crt`
to every command.

### CoAP

CoAP adapter implements CoAP protocol using underlying UDP and according to [RFC 7252](https://tools.ietf.org/html/rfc7252). To send and receive messages over CoAP, you can use [Copper](https://github.com/mkovatsc/Copper) CoAP user-agent. To set the add-on, please follow the installation instructions provided [here](https://github.com/mkovatsc/Copper#how-to-integrate-the-copper-sources-into-firefox). Once the Mozilla Firefox and Copper are ready and CoAP adapter is running locally on the default port (5683), you can navigate to the appropriate URL and start using CoAP. The URL should look like this:

```
coap://localhost/channels/<channel_id>/messages?authorization=<thing_auth_key>
```

To send a message, use `POST` request. To subscribe, send `GET` request with Observe option set to 0. There are two ways to unsubscribe:
  1) Send `GET` request with Observe option set to 1.
  2) Forget the token and send `RST` message as a response to `CONF` message received by the server.

The most of the notifications received from the Adapter are non-confirmable. By [RFC 7641](https://tools.ietf.org/html/rfc7641#page-18):

> Server must send a notification in a confirmable message instead of a non-confirmable message at least every 24 hours. This prevents a client that went away or is no longer interested from remaining in the list of observers indefinitely.

CoAP Adapter sends these notifications every 12 hours. To configure this period, please check [adapter documentation](../coap/README.md) If the client is no longer interested in receiving notifications, the second scenario described above can be used to unsubscribe

### LoRa
 
 Before running Mainflux lora-adapter you must install and run [LoRa Server](https://www.loraserver.io/loraserver/overview). First, execute the following command: 
 
 ```bash
 go get github.com/brocaar/loraserver-docker
 ```
 
 Once everything is installed, execute the following command from the LoRa Server project root:
 
 ```bash
 docker-compose up
 
 ```
 
 The Mainflux lora-adapter can do bridging between both systems. Basically, the service subscribe to the [LoRa Gateway Bridge](https://www.loraserver.io/lora-gateway-bridge/overview/), a service which abstracts the [SemTech packet-forwarder UDP protocol](https://github.com/Lora-net/packet_forwarder/blob/master/PROTOCOL.TXT) into JSON over MQTT.
 
 You must configure the `MF_LORA_ADAPTER_LORA_MESSAGE_URL` variable in the lora-adapter docker-compose.yml with the address of your LoRa Gateway Bridge, otherwise the composition will fail:
 
 ```bash
 docker-compose -f docker/addons/lora-adapter/docker-compose up
 
 ```
  
 At this point Mainflux and LoRa Server are running. To provision the LoRa Server with Networks, Organizations, Gateways, Applications and Devices
 you have to implement the gRPC API. Over the [LoRa App Server](https://www.loraserver.io/lora-app-server/overview), which is good example of the gRPC API implementation, you can do it as well.
 
 #### LoRa Server setup
 - **Create Organization:** To add your own Gateways to the network you must have an Organization.
 - **Add Network LoRa Server:** Set the address of your Network-Server API that is used by LoRa App Server or other custom components interacting with LoRa Server (by default loraserver:8000).
 - **Create a Gateways-Profile:** In this profile you can select the radio LoRa channels and the LoRa Network Server to use.
 - **Create a Service-profile:** A service-profile connects an organization to a network-server and defines the features that an organization can use on this Network-Server.
 - **Create a Gateway:** You must set proper ID in order to be discovered by LoRa Server.
 - **Create a LoRa Server Application:** You can then create Devices by connecting them to this application. This is equivalent to Devices connected to channels in Mainflux.
 - **Create a Device-Profile:** Before creating Device you must create Device profile where you will define some parameter as LoRaWAN MAC version (format of the device address) and the LoRaWAN regional parameter (frequency band). This will allow you to create many devices using this profile.
 - **Create a Device:** Then you can create a Device. To must configure the `network session key` and `application session key` of your Device. You can generate and copy them on your device configuration or you can use your own pre generated keys and set them using the [Lora App Server](https://www.loraserver.io/lora-app-server/overview/).
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
 
## Add-ons

The `<project_root>/docker` folder contains an `addons` directory. This directory is used for various services that are not core to the Mainflux platform but could be used for providing additional features.

In order to run these services, core services, as well as the network from the core composition, should be already running.

### Writers

Writers provide an implementation of various `message writers`. Message writers are services that consume normalized (in `SenML` format) Mainflux messages and store them in specific data store.

#### InfluxDB, InfluxDB-writer and Grafana

From the project root execute the following command:

```bash
docker-compose -f docker/addons/influxdb-writer/docker-compose.yml up -d
```
This will install and start:

- [InfluxDB](https://docs.influxdata.com/influxdb) - time series database
- InfluxDB writer - message repository implementation for InfluxDB
- [Grafana](https://grafana.com) - tool for database exploration and data visualization and analytics

Those new services will take some additional ports:

- 8086 by InfluxDB
- 8900 by InfluxDB writer service
- 3001 by Grafana

To access Grafana, navigate to `http://localhost:3001` and login with: `admin`, password: `admin`

#### Cassandra and Cassandra-writer

```bash
./docker/addons/cassandra-writer/init.sh
```
_Please note that Cassandra may not be suitable for your testing enviroment because it has high system requirements._

#### MongoDB and MongoDB-writer

```bash
docker-compose -f docker/addons/mongodb-writer/docker-compose.yml up -d
```
MongoDB default port (27017) is exposed, so you can use various tools for database inspection and data visualization.

### Readers

Readers provide an implementation of various `message readers`.
Message readers are services that consume normalized (in `SenML` format) Mainflux messages from data storage and opens HTTP API for message consumption.
Installing corresponding writer before reader is implied.


#### InfluxDB-reader

```bash
docker-compose -f docker/addons/influxdb-reader/docker-compose.yml up -d
```
Service exposes [HTTP API](https://github.com/mainflux/mainflux/blob/master/readers/swagger.yml) for fetching messages on port 8905


To read sent messages on channel with id `channel_id` you should send `GET` request to `/channels/<channel_id>/messages` with thing access token in `Authorization` header. That thing must be connected to  channel with `channel_id`

```
curl -s -S -i  -H "Authorization: <thing_token>" http://localhost:8905/channels/<channel_id>/messages
```

Response should look like this:

```
HTTP/1.1 200 OK
Content-Type: application/json
Date: Tue, 18 Sep 2018 18:56:19 GMT
Content-Length: 228

{
    "messages": [
        {
            "Channel": 1,
            "Publisher": 2,
            "Protocol": "mqtt",
            "Name": "name:voltage",
            "Unit": "V",
            "Value": 5.6,
            "Time": 48.56
        },
        {
            "Channel": 1,
            "Publisher": 2,
            "Protocol": "mqtt",
            "Name": "name:temperature",
            "Unit": "C",
            "Value": 24.3,
            "Time": 48.56
        }
    ]
}
```
Note that you will receive only those messages that were sent by authorization token's owner.
You can specify `offset` and `limit` parameters in order to fetch specific group of messages. In that case, your request should look like:

```
curl -s -S -i  -H "Authorization: <thing_token>" http://localhost:8905/channels/<channel_id>/messages?offset=0&limit=5
```
If you don't provide them, default values will be used instead: 0 for `offset`, and 10 for `limit`.

#### Cassandra-reader

```bash
docker-compose -f docker/addons/cassandra-reader/docker-compose.yml up -d
```
Service exposes [HTTP API](https://github.com/mainflux/mainflux/blob/master/readers/swagger.yml) for fetching messages on port 8903

Aside from port, reading request is same as for other readers:

```
curl -s -S -i  -H "Authorization: <thing_token>" http://localhost:8903/channels/<channel_id>/messages
```

#### MongoDB-reader

```bash
docker-compose -f docker/addons/mongodb-reader/docker-compose.yml up -d
```
Service exposes [HTTP API](https://github.com/mainflux/mainflux/blob/master/readers/swagger.yml) for fetching messages on port 8904

Aside from port, reading request is same as for other readers:

```
curl -s -S -i  -H "Authorization: <thing_token>" http://localhost:8904/channels/<channel_id>/messages
```

## TLS Configuration

By default gRPC communication is not secure.

### Server configuration

### Securing PostgreSQL connections

By default, Mainflux will connect to Postgres using insecure transport.
If a secured connection is required, you can select the SSL mode and set paths to any extra certificates and keys needed. 

`MF_USERS_DB_SSL_MODE` the SSL connection mode for Users.
`MF_USERS_DB_SSL_CERT` the path to the certificate file for Users.
`MF_USERS_DB_SSL_KEY` the path to the key file for Users.
`MF_USERS_DB_SSL_ROOT_CERT` the path to the root certificate file for Users.

`MF_THINGS_DB_SSL_MODE` the SSL connection mode for Things.
`MF_THINGS_DB_SSL_CERT` the path to the certificate file for Things.
`MF_THINGS_DB_SSL_KEY` the path to the key file for Things.
`MF_THINGS_DB_SSL_ROOT_CERT` the path to the root certificate file for Things.

Supported database connection modes are: `disabled` (default), `required`, `verify-ca` and `verify-full`

#### Users

If either the cert or key is not set, the server will use insecure transport.

`MF_USERS_SERVER_CERT` the path to server certificate in pem format.

`MF_USERS_SERVER_KEY` the path to the server key in pem format.

#### Things

If either the cert or key is not set, the server will use insecure transport.

`MF_THINGS_SERVER_CERT` the path to server certificate in pem format.

`MF_THINGS_SERVER_KEY` the path to the server key in pem format.

### Client configuration

If you wish to secure the gRPC connection to `things` and `users` services you must define the CAs that you trust.  This does not support mutual certificate authentication.

#### HTTP Adapter

`MF_HTTP_ADAPTER_CA_CERTS` - the path to a file that contains the CAs in PEM format. If not set, the default connection will be insecure. If it fails to read the file, the adapter will fail to start up.

#### Things

`MF_THINGS_CA_CERTS` - the path to a file that contains the CAs in PEM format. If not set, the default connection will be insecure. If it fails to read the file, the service will fail to start up.
