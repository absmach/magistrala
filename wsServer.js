/**
 * Copyright (c) Mainflux
 *
 * Mainflux server is licensed under an Apache license, version 2.0 license.
 * All rights not explicitly granted in the Apache license, version 2.0 are reserved.
 * See the included LICENSE file for more details.
 */

var WebSocketServer = require('ws').Server
  , wsServer = new WebSocketServer({ port: 5152 });

wsServer.on('connection', function connection(ws) {
  ws.on('message', function incoming(message) {
    console.log('received: %s', message);
  });

  ws.send('something');
});

console.log('WS magic happens on port 5152');

/**
 * Exports
 */
module.exports = wsServer;
