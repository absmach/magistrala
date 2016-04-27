/**
 * Copyright (c) Mainflux
 *
 * Mainflux server is licensed under an Apache license, version 2.0 license.
 * All rights not explicitly granted in the Apache license, version 2.0 are reserved.
 * See the included LICENSE file for more details.
 */
var restify = require('restify');
var domain = require('domain');
var config = require('./config');
var log = require('./app/logger');


/**
 * HTTP Restify
 */

/** Create httpServer */
var httpServer = restify.createServer({
    name: "Mainflux"
});


httpServer.pre(restify.pre.sanitizePath());
httpServer.use(restify.acceptParser(httpServer.acceptable));
httpServer.use(restify.bodyParser());
httpServer.use(restify.queryParser());
httpServer.use(restify.authorizationParser());
httpServer.use(restify.CORS());
httpServer.use(restify.fullResponse());

/** Global error handler */
httpServer.use(function(req, res, next) {
    var domainHandler = domain.create();

    domainHandler.on('error', function(err) {
        var errMsg = 'Request: \n' + req + '\n';
        errMsg += 'Response: \n' + res + '\n';
        errMsg += 'Context: \n' + err;
        errMsg += 'Trace: \n' + err.stack + '\n';

        console.log(err.message);

        log.info(err);
    });

    domainHandler.enter();
    next();
});


/**
 * ROUTES
 */
var route = require('./app/routes');
route(httpServer);


/**
 * SERVER START
 */
var port = process.env.PORT || config.http.port;

httpServer.listen(port, function() {
    console.log('HTTP magic happens on port ' + port);
});


/**
 * Exports
 */
module.exports = httpServer;
