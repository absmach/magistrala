var config = require('../config');

/**
 * MONGO DB
 */
var mongojs = require('mongojs');

/** Connect to DB */
var collections = ['devices'];

/** Check if we run with Docker compose */
var dockerMongo = process.env.MONGODB_NAME;
var dbUrl = '';
if (dockerMongo && dockerMongo == '/mainflux-api/mongodb') {
    dbUrl = 'mongodb://' + process.env.MONGODB_PORT_27017_TCP_ADDR + ':' + process.env.MONGODB_PORT_27017_TCP_PORT + '/' + config.db.name;
} else {
    dbUrl = 'mongodb://' + config.db.host + ':' + config.db.port + '/' + config.db.name;
}

var db = mongojs(dbUrl);


/**
 * EXPORTS
 */
module.exports = db;
