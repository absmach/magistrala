/**
 * Copyright (c) Mainflux
 *
 * Mainflux server is licensed under an Apache license, version 2.0 license.
 * All rights not explicitly granted in the Apache license, version 2.0 are reserved.
 * See the included LICENSE file for more details.
 */
var restify = require('restify');
var jwt = require('restify-jwt');
var domain = require('domain');
var config = require('./config/config');
var bunyan = require('bunyan');
var log = bunyan.createLogger({name: "Mainflux"});

/**
 * RESTIFY
 */

/** Create server */
var server = restify.createServer({
    name: "Mainflux"
});

server.pre(restify.pre.sanitizePath());
server.use(restify.acceptParser(server.acceptable));
server.use(restify.bodyParser());
server.use(restify.queryParser());
server.use(restify.authorizationParser());

console.log('Enabling CORS');
server.use(restify.CORS());
server.use(restify.fullResponse());

/** JWT */
server.use(jwt({
    secret: config.tokenSecret,
    requestProperty: 'token',
    getToken: function fromHeaderOrQuerystring(req) {
        var token = (req.body && req.body.access_token) ||
            (req.query && req.query.access_token) ||
            req.headers['x-auth-token'];

        return token;
    }
}).unless({
    path: [
        '/status',
        {url: '/devices', methods: ['POST']}
    ]
}));

/** Global error handler */
server.use(function(req, res, next) {
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
route(server);


/**
 * SERVER START
 */
var port = process.env.PORT || config.port;

console.log('Starting the server');
server.listen(port, function() {
    console.log('%s is running at %s', server.name, server.url);
});


/**
 * Exports
 */
module.exports = server;
