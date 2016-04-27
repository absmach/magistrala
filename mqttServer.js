/**
 * Copyright (c) Mainflux
 *
 * Mainflux server is licensed under an Apache license, version 2.0 license.
 * All rights not explicitly granted in the Apache license, version 2.0 are reserved.
 * See the included LICENSE file for more details.
 */
var config = require('./config');


/***
 * MQTT Mosca
 */
var mosca = require('mosca');

var mqttServer = new mosca.Server(config.mosca);

mqttServer.on('clientConnected', function(client) {
    console.log('client connected', client.id);
});

/** Message received */
mqttServer.on('published', function(packet, client) {
  console.log('Published', packet.payload);
});

mqttServer.on('ready', setupMqtt);

/** Mqtt server ready */
function setupMqtt() {
  console.log('MQTT magic happens on port 1883');
}

/**
 * Exports
 */
module.exports = mqttServer;
