/**
 * Copyright (c) Mainflux
 *
 * Mainflux server is licensed under an Apache license, version 2.0 license.
 * All rights not explicitly granted in the Apache license, version 2.0 are reserved.
 * See the included LICENSE file for more details.
 */

/**
 * Extrenal configs are kept in the config.js file on the same level
 */
var config = require('./config/config');
console.log(config.message);


/**
 * SETUP
 */
var express     = require('express');        // call express
var app         = express();                 // define our app using express
var bodyParser  = require('body-parser');

/** MongoDB */
var mongoose    = require('mongoose');

/** Docker MongoDB url */
var docker_mongo_url = process.env.MAINFLUX_MONGODB_1_PORT_27017_TCP_ADDR

/** Connect to DB */
mongoose.connect(docker_mongo_url || config.db.path + ':' + config.db.port + '/' + config.db.name);

/** Configure app to use bodyParser() */
app.use(bodyParser.urlencoded({ extended: true }));
app.use(bodyParser.json());

var port = process.env.PORT || config.port;        // set our port


/**
 * ROUTES
 */
app.use('/status', require('./app/routes/status'));
app.use('/devices', require('./app/routes/devices'));
app.use('/users', require('./app/routes/users'));
app.use('/sessions', require('./app/routes/sessions'));




/**
 * SERVER START
 */
app.listen(port);
console.log('Magic happens on port ' + port);


/**
 * Export app for testing
 */
module.exports = app;
