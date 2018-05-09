'use strict';

// Service configuration
module.exports = {
    mqtt_port: process.env.MQTT_PORT || 1883,
    ws_port: process.env.WS_PORT || 8880,
    // NATS broker URL
    nats_url: process.env.MQTT_ADAPTER_NATS_URL || 'nats://localhost:4222',
    // Auth service URL
    auth_url: process.env.AUTH_URL || 'http://localhost',
    // Auth service port
    auth_port: process.env.AUTH_PORT || 8180
};
