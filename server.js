/**
 * Copyright (c) Mainflux
 *
 * Mainflux server is licensed under an Apache license, version 2.0 license.
 * All rights not explicitly granted in the Apache license, version 2.0 are reserved.
 * See the included LICENSE file for more details.
 */
var restify = require('restify');
var domain = require('domain');
var config = require('./config/config');
var log = require('./app/logger');

// MongoDB
var mongoose = require('mongoose');

/**
 * Connect to DB
 */
/** Check if we run with Docker compose */
var dockerMongo = process.env.MONGODB_NAME;
var dbUrl = '';
if (dockerMongo && dockerMongo == '/mainflux-api-docker/mongodb') {
    dbUrl = 'mongodb://' + process.env.MONGODB_PORT_27017_TCP_ADDR + ':' + process.env.MONGODB_PORT_27017_TCP_PORT + '/' + config.db.name;
} else {
    dbUrl = 'mongodb://' + config.db.addr + ':' + config.db.port + '/' + config.db.name;
}

mongoose.connect(dbUrl, {server:{auto_reconnect:true}});


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
server.use(restify.CORS());
server.use(restify.fullResponse());

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

var banner = `
oocccdMMMMMMMMMWOkkkkoooolcclX
llc:::0MMMMMMMM0xxxxxdlllc:::d
lll:::cXMMMMMMXxxxxxxxdlllc:::
lllc:::cXMMMMNkxxxdxxxxolllc::
olllc:::oWMMNkxxxdloxxxxolllc:   ##     ##    ###    #### ##    ## ######## ##       ##     ## ##     ##
xolllc:::xWWOxxxdllloxxxxolllc   ###   ###   ## ##    ##  ###   ## ##       ##       ##     ##  ##   ## 
xxolllc:::x0xxxdllll:oxxxxllll   #### ####  ##   ##   ##  ####  ## ##       ##       ##     ##   ## ##  
xxxolllc::oxxxxllll:::dxxxdlll   ## ### ## ##     ##  ##  ## ## ## ######   ##       ##     ##    ###   
xxxdllll:lxxxxolllc:::Okxxxdll   ##     ## #########  ##  ##  #### ##       ##       ##     ##   ## ##  
0xxxdllloxxxxolllc:::OMNkxxxdl   ##     ## ##     ##  ##  ##   ### ##       ##       ##     ##  ##   ## 
W0xxxdllxxxxolllc:::xMMMXxxxxd   ##     ## ##     ## #### ##    ## ##       ########  #######  ##     ##
MWOxxxdxxxxdlllc:::oWMMMMKxxxx
MMWkxxxxxxdlllc:::oNMMMMMM0xxx
MMMXxxxxxdllllc::cXMMMMMMMWOxx
MMMM0xxxxolllc:::kMMMMMMMMMXxx
`
server.listen(port, function() {
    console.log(banner);
    console.log('Magic happens on port ' + port);
});


/**
 * Exports
 */
module.exports = server;
