// server.js

// BASE SETUP
// =============================================================================

// call the packages we need
var express     = require('express');        // call express
var app         = express();                 // define our app using express
var bodyParser  = require('body-parser');

// MongoDB
var mongoose    = require('mongoose');
mongoose.connect('mongodb://localhost:27017/mainflux'); // connect to our database


var Thing       = require('./app/models/thing');

// configure app to use bodyParser()
// this will let us get the data from a POST
app.use(bodyParser.urlencoded({ extended: true }));
app.use(bodyParser.json());

var port = process.env.PORT || 8080;        // set our port

// ROUTES FOR OUR API
// =============================================================================
var router = express.Router();              // get an instance of the express Router

// on routes that end in /things
// ----------------------------------------------------
router.route('/things')

    // create a things (accessed at POST http://localhost:8080/api/things)
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

    // get all the things (accessed at GET http://localhost:8080/api/things)
    .get(function(req, res) {
        Thing.find(function(err, things) {
            if (err)
                res.send(err);

            res.json(things);
        });
    });

    
// on routes that end in /things/:thing_id
// ----------------------------------------------------
router.route('/things/:thing_id')

    // get the thing with that id (accessed at GET http://localhost:8080/api/things/:thing_id)
    .get(function(req, res) {
        Thing.findById(req.params.thing_id, function(err, thing) {
            if (err)
                res.send(err);
            res.json(thing);
        });
    })

    // update the thing with this id (accessed at PUT http://localhost:8080/api/things/:thing_id)
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

    // delete the thing with this id (accessed at DELETE http://localhost:8080/api/things/:thing_id)
    .delete(function(req, res) {
        Thing.remove({
            _id: req.params.thing_id
        }, function(err, thing) {
            if (err)
                res.send(err);

            res.json({ message: 'Successfully deleted' });
        });
    });



// middleware to use for all requests
router.use(function(req, res, next) {
    // do logging
    console.log('Something is happening.');
    next(); // make sure we go to the next routes and don't stop here
});


// test route to make sure everything is working (accessed at GET http://localhost:8080/api)
router.get('/', function(req, res) {
    res.json({ message: 'hooray! welcome to our api!' });   
});

// more routes for our API will happen here

// REGISTER OUR ROUTES -------------------------------
// all of our routes will be prefixed with /api
app.use('/api', router);


// START THE SERVER
// =============================================================================
app.listen(port);
console.log('Magic happens on port ' + port);
