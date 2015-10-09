var express = require('express');
var router = express.Router();              // get an instance of the express Router

var Device   = require('../models/device');

/**
 * /devices
 */
router.route('/')

    /** Create a device (accessed at POST http://localhost:8080/devices) */
    .post(function(req, res) {

		console.log("req.headers['mainflux_uuid']", req.headers['mainflux_uuid']);
		console.log("req.headers['mainflux_token']", req.headers['mainflux_token']);
        
        var device = new Device();		// create a new instance of the Device model
        device.name = req.body.name;	// set the device's name (comes from the request)

        /** Save the device and check for errors */
        device.save(function(err) {
            if (err)
                res.send(err);

            res.json({ message: 'Device created!' });
        });
        
    })

    /** Get all the devices (accessed at GET http://localhost:8080/devices) */
    .get(function(req, res) {
        Device.find(function(err, devices) {
            if (err)
                res.send(err);

            res.json(devices);
        });
    });

    
/**
 * /devices/:device_id
 * N.B. Colon (`:`) is needed because of Express `req.params`: http://expressjs.com/api.html#req.params
 */
router.route('/:device_id')

    /** Get the device with that id (accessed at GET http://localhost:8080/devices/:device_id) */
    .get(function(req, res) {
        Device.findById(req.params.device_id, function(err, device) {
            if (err)
                res.send(err);
            res.json(device);
        });
    })

    /** Update the device with this id (accessed at PUT http://localhost:8080/devices/:device_id) */
    .put(function(req, res) {

        /** Use our device model to find the device we want */
        Device.findById(req.params.device_id, function(err, device) {

            if (err)
                res.send(err);

            device.name = req.body.name;  // update the devices info

            /** Save the device */
            device.save(function(err) {
                if (err)
                    res.send(err);

                res.json({ message: 'Device updated!' });
            });

        })
    })

    /** Delete the device with this id (accessed at DELETE http://localhost:8080/devices/:device_id) */
    .delete(function(req, res) {
        Device.remove({
            _id: req.params.device_id
        }, function(err, device) {
            if (err)
                res.send(err);

            res.json({ message: 'Successfully deleted' });
        });
    });

/**
 * Export router module
 */
module.exports = router;
