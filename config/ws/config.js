/**
 * Copyright (c) Mainflux
 *
 * Mainflux server is licensed under an Apache license, version 2.0 license.
 * All rights not explicitly granted in the Apache license, version 2.0 are reserved.
 * See the included LICENSE file for more details.
 */
var config = {};

/**
 * WS Server
 */
config.ws = {
    host: '0.0.0.0',
    port: 9090,
}

/**
 * NATS
 */
config.nats = {
    host : 'nats',
    port : 4222
}


module.exports = config;
