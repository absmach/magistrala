var express = require('express');
var router = express.Router();              // get an instance of the express Router

var devices = require('../controllers/devices');

/**
 * /devices
 */
router.route('/')

    /** Create a device (accessed at POST http://localhost:8080/devices) */
    .post(devices.createDevice)

    /** Get all the devices (accessed at GET http://localhost:8080/devices) */
    .get(devices.getAllDevices);

    
/**
 * /devices/:device_id
 * N.B. Colon (`:`) is needed because of Express `req.params`: http://expressjs.com/api.html#req.params
 */
router.route('/:device_id')

    /** Get the device with that id (accessed at GET http://localhost:8080/devices/:device_id) */
    .get(devices.getDevice)

    /** Update the device with this id (accessed at PUT http://localhost:8080/devices/:device_id) */
    .put(devices.updateDevice)

    /** Delete the device with this id (accessed at DELETE http://localhost:8080/devices/:device_id) */
    .delete(devices.deleteDevice);

/**
 * Export router module
 */
module.exports = router;
