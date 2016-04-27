var restify = require('restify');
var config = require('../config');
var devicesController = require('./controllers/devices');
var statusController = require('./controllers/status');

function routes(api) {
    /**
     * /STATUS
     */
    api.get('/status', statusController.getStatus);


    /**
     * /DEVICES
     */
    /** Create a device (accessed at POST http://localhost:8080/devices) */
    api.post('/devices', devicesController.createDevice);

    /** Get all the devices (accessed at GET http://localhost:8080/devices) */
    api.get('/devices', devicesController.getAllDevices);

    /**
     * /devices/:device_id
     * N.B. Colon (`:`) is needed because of Express `req.params`: http://expressjs.com/api.html#req.params
     */
    /** Get the device with given id (accessed at GET http://localhost:8080/devices/:device_id) */
    api.get('/devices/:device_id', devicesController.getDevice)

    /** Update the device with given id (accessed at PUT http://localhost:8080/devices/:device_id) */
    api.put('/devices/:device_id', devicesController.updateDevice)

    /** Delete the device with given id (accessed at DELETE http://localhost:8080/devices/:device_id) */
    api.del('/devices/:device_id', devicesController.deleteDevice);
}

module.exports = routes;
