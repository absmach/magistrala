var mongojs = require('mongojs');
var devicesDb = require('../database').collection('devices');

/** createDevice() */
exports.createDevice = function(req, res, next) {

    console.log("req.headers['x-auth-token'] = ", req.headers['x-auth-token']);
        
    /** Save the device and check for errors */
    devicesDb.insert(req.body, function(err, device) {
        if (err)
            res.send(err);

        res.json(device);
    });

    return next();
}

/** getAllDevices() */
exports.getAllDevices = function(req, res, next) {

	console.log("req.headers['x-auth-token'] = ", req.headers['x-auth-token']);
		
    devicesDb.find(req.body, function(err, devices) {
        if (err)
            res.send(err);

        res.json(devices);
        return next();
    });
}

/** getDevice() */
exports.getDevice = function(req, res, next) {

    console.log(req.params.device_id);
    devicesDb.findOne({_id: mongojs.ObjectId(req.params.device_id)}, function(err, device) {
        if (err)
            return next(err);
        
        res.json(device);
        return next();
    });
}

/** updateDevice() */
exports.updateDevice = function(req, res, next) {
    /** Use our device model to find the device we want */
    console.log(req.body);
    devicesDb.update({
        _id: mongojs.ObjectId(req.params.device_id)
    },
        {$set: req.body},
        function(err, device) {
            if (err)
                return next(err);

            res.send('OK');
            return next();
    });
}

/** deleteDevice() */
exports.deleteDevice = function(req, res, next) {
    deviceDb.remove({
        _id: mongojs.ObjectId(req.params.device_id)
    }, function(err, device) {
        if (err)
            return next(err);

        res.send('OK');
        return next();
    });
}

