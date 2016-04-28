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
 * Mosca
 */
config.ws = {
    port: 9090,
}


module.exports = config;
