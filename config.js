/**
 * Copyright (c) Mainflux
 *
 * Mainflux server is licensed under an Apache license, version 2.0 license.
 * All rights not explicitly granted in the Apache license, version 2.0 are reserved.
 * See the included LICENSE file for more details.
 */
var mosca = require('mosca');

var config = {};

/**
 * HTTP
 */
config.http = {
    message : 'We are in development',
    port : 7070,
    version: 0.1
}

/**
 * MongoDB
 */
config.db = {
    host : 'localhost',
    port : 27017,
    name : 'test'
}

/**
 * Mosca
 */
var moscaBackend = {
    type: 'redis',
    redis: require('redis'),
    db: 7,
    port: 6379,
    return_buffers: true, // to handle binary payloads
    host: "localhost"
}

config.mosca = {
    port: 1883,
    backend: moscaBackend,
    persistence: {
        factory: mosca.persistence.Redis
    }
}


module.exports = config;
