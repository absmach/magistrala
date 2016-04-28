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

module.exports = config;
