var express = require('express');
var router = express.Router();              // get an instance of the express Router
var _ = require('lodash');
var jwt    = require('jsonwebtoken'); // used to create, sign, and verify tokens

var config = require('../../config/config');
var User   = require('../models/user');

// on routes that end in /things
// ----------------------------------------------------
router.route('/')

 // create a new user account and return user token (accessed at POST http://localhost:8080/users)
    .post(function(req, res) {

        if (!req.body.email || !req.body.password) { // Check for email and password in request
            return res.json({status:400, message: 'Bad request.' });
        }
        //TODO: Check if user with this email already exist
        var user = new User({
            email: req.body.email,
            password: req.body.password
        });
        // save the user and generate token
        user.save(function(err) {
            if (err)
                res.send(err);
            // Create user token
            var token = jwt.sign(user, config.secretToken, {
                expiresInMinutes: config.userTokenExpirePeriod
            });
            res.json({
                status: 200,
                message: 'Account created!',
                token: token
            });
        });
    });

// export router module
module.exports = router;

