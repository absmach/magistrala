/**
 * Copyright (c) Mainflux
 *
 * Mainflux server is licensed under an Apache license, version 2.0 license.
 * All rights not explicitly granted in the Apache license, version 2.0 are reserved.
 * See the included LICENSE file for more details.
 */

'use strict';

var http = require('http');
var websocket = require('websocket-stream');
var net = require('net');
var aedes = require('aedes')();
var logging = require('aedes-logging');
var request = require('request');
const util = require('util')

var config = require('./mqtt.config');
var nats = require('nats').connect(config.nats_url);

var protobuf = require('protocol-buffers');
const fs = require('fs');

// pass a proto file as a buffer/string or pass a parsed protobuf-schema object
var message = protobuf(fs.readFileSync('message.proto'));

var servers = [
    startWs(),
    startMqtt()
];

logging({
    instance: aedes,
    servers: servers
});

/**
 * WebSocket
 */
function startWs() {
    var server = http.createServer();
    websocket.createServer({
        server: server
    }, aedes.handle);
    server.listen(config.ws_port);
    return server;
}
/**
 * MQTT
 */
function startMqtt() {
    return net.createServer(aedes.handle).listen(config.mqtt_port);
}
/**
 * NATS
 */
nats.subscribe('channel.*', function (msg) {

    var m = message.RawMessage.decode(Buffer.from(msg))

    if (m.Protocol == 'mqtt') {
        // Ignore MQTT loopback,
        // packet has been already published by MQTT broker
        // before sending it to NATS
        return;
    }

    // Parse and adjust content-type
    if (m.ContentType == "application/senml+json") {
        m.ContentType = "senml-json"
    }

    var packet = {
        cmd: 'publish',
        qos: 2,
        topic: 'mainflux/channels/' + m.Channel + '/messages/' + m.ContentType,
        payload: m.Payload,
        retain: false
    };

    aedes.publish(packet);
});

/**
 * Hooks
 */
// AuthZ PUB
aedes.authorizePublish = function (client, packet, callback) {
    // Topics are in the form `mainflux/channels/<channel_id>/messages/senml-json`
    var channel = packet.topic.split('/')[2];

    /**
     * Check if PUB is authorized
     */
    var options = {
        url: config.auth_url + ':' + config.auth_port + '/channels/' + channel + '/access-grant',
        method: 'GET',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': client.password
        }
    };

    request(options, function (err, res) {
        var error = null;
        var msg = {};
        if (res && (res.statusCode === 200)) {
            console.log('Publish authorized OK');

          var msg = message.RawMessage.encode({
              /**
               * We must publish on NATS here, because on_publish() is also called
               * when we receive message from NATS from other adapters (in nats.subscribe()),
               * so we must avoid re-publishing on NATS what came from other adapters
               */
              Publisher: client.id,
              Channel: channel,
              Protocol: 'mqtt',
              ContentType: packet.topic.split('/')[4],
              Payload: packet.payload
            });

            console.log(msg);

            console.log(util.inspect(packet, false, null))

            // Pub on NATS
            nats.publish('channel.' + channel, msg);
        } else {
            console.log('Publish not authorized');
            error = 4; // Bad username or password
        }
        callback(error);
    });
};

// AuthZ SUB
aedes.authorizeSubscribe = function (client, packet, callback) {
    // Topics are in the form `mainflux/channels/<channel_id>/messages/senml-json`
    var channel = packet.topic.split('/')[2];
    /**
    * Check if PUB is authorized
    */
    var options = {
        url: config.auth_url + ':' + config.auth_port + '/channels/' + channel + '/access-grant',
        method: 'GET',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': client.password
        }
    };

    request(options, function (err, res) {
        var error = null;
        if (res && (res.statusCode === 200)) {
            console.log('Subscribe authorized OK');
        } else {
            console.log('Subscribe not authorized');
            error = 4; // Bad username or password
        }
        callback(error, packet);
    });
};

// AuthX
aedes.authenticate = function (client, username, password, callback) {
    var c = client;
    var options = {
        url: config.auth_url + ':' + config.auth_port + '/access-grant',
        method: 'GET',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': password
        }
    };
    request(options, function (err, res) {
        var error = null;
        var success = null;
        if (res && (res.statusCode === 200)) {
            // Set MQTT client.id to correspond to Mainflux device UUID
            c.id = res.headers['x-client-id'];
            // Store password for future references
            c.password = password;
            success = true;
        } else {
            error = new Error('Auth error');
            error.returnCode = 4; // Bad username or password
            success = false;
        }
        // Respond with auth error and success
        callback(error, success);
    });
};
/**
 * Handlers
 */
aedes.on('clientDisconnect', function (client) {
    var c = client;
    console.log('client disconnect', client.id);
    // Remove client password
    c.password = null;
});

aedes.on('clientError', function (client, err) {
  console.log('client error', client.id, err.message, err.stack)
})

aedes.on('connectionError', function (client, err) {
  console.log('client error', client, err.message, err.stack)
})
