'use strict';

var http = require('http'),
    net = require('net'),
    aedes = require('aedes')(),
    logging = require('aedes-logging'),
    protobuf = require('protocol-buffers'),
    websocket = require('websocket-stream'),
    grpc = require('grpc'),
    fs = require('fs'),
    bunyan = require('bunyan');

// pass a proto file as a buffer/string or pass a parsed protobuf-schema object
var logger = bunyan.createLogger({name: "mqtt"}),
    config = {
        mqtt_port: process.env.MF_MQTT_ADAPTER_PORT || 1883,
        ws_port: process.env.MF_MQTT_WS_PORT || 8880,
        nats_url: process.env.MF_NATS_URL || 'nats://localhost:4222',
        auth_url: process.env.MF_THINGS_URL || 'localhost:8181',
        schema_dir: process.argv[2] || '.'
    },
    message = protobuf(fs.readFileSync(config.schema_dir + '/message.proto')),
    thingsSchema = grpc.load(config.schema_dir + "/internal.proto").mainflux,
    nats = require('nats').connect(config.nats_url),
    things = new thingsSchema.ThingsService(config.auth_url, grpc.credentials.createInsecure()),
    servers = [
        startMqtt(),
        startWs()
    ];

logging({
    instance: aedes,
    servers: servers
});

// MQTT over WebSocket
function startWs() {
    var server = http.createServer();
    websocket.createServer({server: server}, aedes.handle);
    server.listen(config.ws_port);
    return server;
}

function startMqtt() {
    return net.createServer(aedes.handle).listen(config.mqtt_port);
}

nats.subscribe('channel.*', function (msg) {
    var m = message.RawMessage.decode(Buffer.from(msg)),
        packet = {
            cmd: 'publish',
            qos: 2,
            topic: 'channels/' + m.Channel + '/messages',
            payload: m.Payload,
            retain: false
        };

    aedes.publish(packet);
});

aedes.authorizePublish = function (client, packet, publish) {
    // Topics are in the form `channels/<channel_id>/messages`
    var channel = packet.topic.split('/')[1],
        accessReq = {
            token: client.password,
            chanID: channel
        },
        onAuthorize = function (err, res) {
            var rawMsg
            if (!err) {
                logger.info('authorized publish');
                
                rawMsg = message.RawMessage.encode({
                    Publisher: client.id,
                    Channel: channel,
                    Protocol: 'mqtt',
                    Payload: packet.payload
                });
                nats.publish('channel.' + channel, rawMsg);
    
                // Set empty topic for packet so that it won't be published two times.
                packet.topic = '';
                publish(0);
            } else {
                logger.warn("unauthorized publish: %s", err.message);
                publish(4); // Bad username or password
            }
        };

    things.CanAccess(accessReq, onAuthorize);
};


aedes.authorizeSubscribe = function (client, packet, subscribe) {
    // Topics are in the form `channels/<channel_id>/messages`
    var channel = packet.topic.split('/')[1],
        accessReq = {
            token: client.password,
            chanID: channel
        },
        onAuthorize = function (err, res) {
            if (!err) {
                logger.info('authorized subscribe');
                subscribe(null, packet);
            } else {
                logger.warn('unauthorized subscribe: %s', err);
                subscribe(4, packet); // Bad username or password
            }
        };
    
    things.canAccess(accessReq, onAuthorize);
};

aedes.authenticate = function (client, username, password, acknowledge) {
    var pass = (password || "").toString(),
        identity = {value: pass},
        onIdentify = function(err, res) {
            if (!err) {
                client.id = res.value.toString() || "";
                client.password = pass;
                acknowledge(null, true);
            } else {
                logger.warn('failed to authenticate client with key %s', password);
                acknowledge(err, false);
            }
        };
        
    things.identify(identity, onIdentify);
};

aedes.on('clientDisconnect', function (client) {
    logger.info('disconnect client %s', client.id);
    client.password = null;
});

aedes.on('clientError', function (client, err) {
  logger.warn('client error: client: %s, error: %s', client.id, err.message);
});

aedes.on('connectionError', function (client, err) {
  logger.warn('client error: client: %s, error: %s', client.id, err.message);
});
