Once a channel is provisioned and thing is connected to it, it can start to
publish messages on the channel. The following sections will provide an example
of message publishing for each of the supported protocols.

## HTTP

To publish message over channel, thing should send following request:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X POST -H "Content-Type: application/senml+json" -H "Authorization: <thing_token>" https://localhost/http/channels/<channel_id>/messages -d '[{"bn":"some-base-name:","bt":1.276020076001e+09, "bu":"A","bver":5, "n":"voltage","u":"V","v":120.1}, {"n":"current","t":-5,"v":1.2}, {"n":"current","t":-4,"v":1.3}]'
```

Note that you should always send array of messages in senML format.

## WebSocket

To publish and receive messages over channel using web socket, you should first
send handshake request to `/channels/<channel_id>/messages` path. Don't forget
to send `Authorization` header with thing authorization token.

If you are not able to send custom headers in your handshake request, send it as
query parameter `authorization`. Then your path should look like this
`/channels/<channel_id>/messages?authorization=<thing_auth_key>`.

If you are using the docker environment prepend the url with `ws`. So for example
`/ws/channels/<channel_id>/messages?authorization=<thing_auth_key>`

### Basic nodejs example

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

## MQTT

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

###### Subtopics
In order to use MQTT subtopics and extend the channel with subtopic or name the channel using your own naming convention, You can simply add any suffix on base ` /channels/<channel_id>/messages` topic.

Example subtopic publish/subscribe for bedroom temperature would be

`channels/<channel_id>/messages/bedroom/temperature`

Subtopics are generic and multilevel, You can use any prefix with any level depth.

Topic with subtopics are propagated to NATS broker in the following format

`channel.<channel_id>.<optional_subtopic>`

Subscription directly to NATS on our example channel with subtopic

`channels/<channel_id>/messages/bedroom/temperature` would be in format `channel.<channel_id>.bedroom.temperature`


For more information and examples checkout [official nats.io documentation](https://nats.io/documentation/writing_applications/subscribing/)


## CoAP

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
