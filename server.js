// server.js

/**
 * Extrenal configs are kept in the config.js file on the same level
 */
var config = require('./config/config');
console.log(config.message);


// BASE SETUP
// =============================================================================

// call the packages we need
var express     = require('express');        // call express
var app         = express();                 // define our app using express
var bodyParser  = require('body-parser');

// MongoDB
var mongoose    = require('mongoose');
// Docker MongoDB url
var docker_mongo_url = process.env.MAINFLUX_MONGODB_1_PORT_27017_TCP_ADDR

mongoose.connect(docker_mongo_url || config.db.path + ':' + config.db.port + '/' + config.db.name); // connect to our database


// configure app to use bodyParser()
// this will let us get the data from a POST
app.use(bodyParser.urlencoded({ extended: true }));
app.use(bodyParser.json());

var port = process.env.PORT || config.port;        // set our port

// ROUTES FOR OUR API
// =============================================================================
app.use('/status', require('./app/routes/status'));
app.use('/things', require('./app/routes/things'));


// START THE SERVER
// =============================================================================
app.listen(port);
console.log('Magic happens on port ' + port);
