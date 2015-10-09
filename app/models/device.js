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

/**
 * \b Device Schema
 *
 * @param   name            {String}    Friendly name of the device
 * @param   description     {String}    Description of the device
 * @param   creator         {String}    Device creator
 * @param   owner           {String}    Owner of the device
 * @param   group           {String}    Device group that device belongs to
 * @param   deviceId        {String}    UUID of the device
 * @param   apiKey          {String}    Authentication token for accessing Mainflux API
 * @param   createdAt       {Date}      Timestamp of the device creation
 * @param   isPublic        {Boolean}   Is device publicly shared (not claimed yet)
 * @param   online          {Boolean}   Is device currently connected
 * @param   lastSeen        {Date}      When was the device last time connected
 * @param   updatedAt       {Date}      Timestamp of the last interaction between device and cloud
 * @param   manufacturerId  {String}    UUID of the manufacturing company
 * @param   serialNumber    {String}    Manufacturer marks devices by serial number
 * @param   productId       {String}    devices belong to some product (ex. HUE lights)
 * @param   activationCode  {String}    3rd party apps might prefer codes for device claiming
 * @param   deviceLocation  {String}    Physical location of the device
 * @param   firmwareVersion {String}    Needed for the OTA updates
 */
var DeviceSchema = new Schema({
    name: {
        type: String,
        required: false
    },
    description: {
        type: String,
        required: false
    },
    creator: {
        type: String,
        required: false,
    },
    owner: {
        type: String,
        required: false,
    },
    group: {
        type: Array,
        default: []
    },
    deviceId: {
        type: String,
        required: false,
        index: true,
        match: /^[0-9a-f]{10}$/
    },
    apiKey: {
        type: String,
        required: false,
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

