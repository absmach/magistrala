/**
 * Copyright (c) Mainflux
 *
 * Mainflux server is licensed under an Apache license, version 2.0 license.
 * All rights not explicitly granted in the Apache license, version 2.0 are reserved.
 * See the included LICENSE file for more details.
 */
var config = {};

/**
 * Core HTTP server
 */
config.server = {
    message : 'We are in development',
    port : 6969,
    version: 0.1
}

/**
 * MongoDB
 */
config.db = {
    host : mongo,
    port : 27017,
    name : 'test'
}

/**
 * NATS
 */
config.nats = {
    host : nats,
    port : 4222
}

module.exports = config;
