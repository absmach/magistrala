var config = require('../config/config');

/**
 * MONGO DB
 */
var mongojs = require('mongojs');

/** Docker MongoDB url */
var docker_mongo_url = process.env.MAINFLUX_MONGODB_1_PORT_27017_TCP_ADDR

/** Connect to DB */
console.log("Connecting to DB");
var collections = ['devices'];
var db = mongojs(docker_mongo_url || config.db.path + ':' + config.db.port + '/' + config.db.name);

/**
 * EXPORTS
 */
module.exports = db;
