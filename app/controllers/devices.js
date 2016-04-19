var config = require('../../config/config');
var log = require('../logger');

var Device = require('../models/device');


/** createDevice() */
exports.createDevice = function(req, res, next) {

    console.log("req.headers['x-auth-token'] = ", req.headers['x-auth-token']);
    console.log("req.headers['content-type'] = ", req.headers['content-type']);
        
    /** Save the device and check for errors */
    var device = new Device();		// create a new instance of the Device model
    device.name = req.body.name;	// set the device's name (comes from the request)

    /** Save the device and check for errors */
    device.save(function(err) {
        if (err)
            res.send(err);

        res.json(device);
        return next();
    });
}

/** getAllDevices() */
exports.getAllDevices = function(req, res, next) {

	console.log("req.headers['x-auth-token'] = ", req.headers['x-auth-token']);

    Device.find(function(err, devices) {
        if (err)
            res.send(err);

        res.json(devices);
        return next();
    });
}

/** getDevice() */
exports.getDevice = function(req, res, next) {

    Device.findById(req.params.device_id, function(err, device) {
        if (err)
            res.send(err);
        
        res.json(device);
        return next();
    });
}

/** updateDevice() */
exports.updateDevice = function(req, res, next) {
    /** Use our device model to find the device we want */
    Device.findById(req.params.device_id, function(err, device) {
        if (err)
            res.send(err);

        device.name = req.body.name;  // update the devices info

        /** Save the device */
        device.save(function(err) {
            if (err)
                res.send(err);

            res.json(device);
            return next();
        });

    });
}

/** deleteDevice() */
exports.deleteDevice = function(req, res, next) {
    Device.remove({
            _id: req.params.device_id
        }, function(err, device) {
            if (err)
                res.send(err);

            res.json(device);
            return next();
        });
}

