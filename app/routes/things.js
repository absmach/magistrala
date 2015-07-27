var express = require('express');
var router = express.Router();              // get an instance of the express Router

var Thing   = require('../models/thing');

// on routes that end in /things
// ----------------------------------------------------
router.route('/')

    // create a things (accessed at POST http://localhost:8080/things)
    .post(function(req, res) {
        
        var thing = new Thing();        // create a new instance of the Bear model
        thing.name = req.body.name;     // set the thing's name (comes from the request)

        // save the thing and check for errors
        thing.save(function(err) {
            if (err)
                res.send(err);

            res.json({ message: 'Thing created!' });
        });
        
    })

    // get all the things (accessed at GET http://localhost:8080/things)
    .get(function(req, res) {
        Thing.find(function(err, things) {
            if (err)
                res.send(err);

            res.json(things);
        });
    });

    
// on routes that end in /things/:thing_id
// ----------------------------------------------------
router.route('/:thing_id')

    // get the thing with that id (accessed at GET http://localhost:8080/things/:thing_id)
    .get(function(req, res) {
        Thing.findById(req.params.thing_id, function(err, thing) {
            if (err)
                res.send(err);
            res.json(thing);
        });
    })

    // update the thing with this id (accessed at PUT http://localhost:8080/things/:thing_id)
    .put(function(req, res) {

        // use our thing model to find the thing we want
        Thing.findById(req.params.thing_id, function(err, thing) {

            if (err)
                res.send(err);

            thing.name = req.body.name;  // update the things info

            // save the thing
            thing.save(function(err) {
                if (err)
                    res.send(err);

                res.json({ message: 'Thing updated!' });
            });

        })
    })

    // delete the thing with this id (accessed at DELETE http://localhost:8080/things/:thing_id)
    .delete(function(req, res) {
        Thing.remove({
            _id: req.params.thing_id
        }, function(err, thing) {
            if (err)
                res.send(err);

            res.json({ message: 'Successfully deleted' });
        });
    });

// export router module
module.exports = router;
