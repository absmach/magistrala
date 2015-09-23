/**
 * Dependencies
 */
var mongoose = require('mongoose');


/**
 * Private variables and functions
 */
var Schema = mongoose.Schema;


/**
 * Exports
 */
var DeviceSchema = new Schema({
	name: {
		type: String,
		required: true
	},
	description: {
		type: String,
		required: false
	},
	creator: {
		type: String,
		required: true,
	},
	owner: {
		type: String,
		required: true,
	},
	group: {
		type: Array,
		default: []
	},
	deviceId: {
		type: String,
		required: true,
		index: true,
		match: /^[0-9a-f]{10}$/
	},
	apiKey: {
		type: String,
		required: true,
		index: true
	},
	createdAt: {
		type: Date,
		index: true,
		default: Date.now
	},
	isPublic: {
		type: Boolean,
		index: true,
		default: false
	},
	online: {
		type: Boolean,
		index: true,
		default: false
	},
	lastSeen: {
		type: Date
	},
	updatedAt: {
		type: Date
	},
	manufacturerId: {
		type: String,
		required: false,
		index: true,
		match: /^[0-9a-f]{10}$/
	},
	serialNumber: {
		type: String,
		required: false,
		index: true,
		match: /^[0-9a-f]{10}$/
	},
	productId: {
		type: String,
		required: false,
		index: true,
		match: /^[0-9a-f]{10}$/
	},
	activationCode: {
		type: String,
		required: false,
		index: true,
		match: /^[0-9a-f]{10}$/
	},
	deviceLocation: {
		type: String,
		required: false,
		index: true,
		match: /^[0-9a-f]{10}$/
	},
	firmwareVersion: {
		type: String,
		required: false,
		index: true,
		match: /^[0-9a-f]{10}$/
	}

});


DeviceSchema.static('exists', function (apikey, deviceid, callback) {
	this.where({ apiKey: apikey, deviceId: deviceid }).findOne(callback);
});

DeviceSchema.static('getDeviceByDeviceId', function (deviceid, callback) {
	this.where({ deviceId: deviceid }).findOne(callback);
});

DeviceSchema.static('getDevicesByApikey', function (apikey, callback) {
	this.where('apiKey', apikey).find(callback);
});


module.exports = mongoose.model('Device', DeviceSchema);

