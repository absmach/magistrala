var express = require('express');
var router = express.Router();              // get an instance of the express Router
var _ = require('lodash');
var jwt    = require('jsonwebtoken'); // used to create, sign, and verify tokens

var config = require('../../config/config');
var User   = require('../models/user');

// on routes that end in /things
// ----------------------------------------------------
router.route('/')

 // Authenticate the user and return user token (accessed at POST http://localhost:8080/sessions)
    .post(function(req, res) {
        if(req.body.email && req.body.password) {
          // Find the user
          User.findOne({
            email: req.body.email
          }, function(err, user) {
                if (err) throw err;
                if(!user){
                    // User with this email does not exist
                    res.json({status:404, message: 'Authentication failed. User not found.' });
                }
                // Validate user password
                user.validateUserPassword(req.body.password,function (err, isMatch) {
                    if(err){
                        res.json({status:401, message: 'Unauthorized.' });
                    }
                    if(isMatch){
                       // Generate user token
                       var token = jwt.sign(user, config.secretToken, {
                            expiresInMinutes: config.userTokenExpirePeriod
                        });
                       res.json({status:200, token: token, message: 'Authentication succeeded.' });
                    } else {
                        res.json({status:401,  message: 'Unauthorized.' });
                    }
                });
            })
        }
        else {
            // Email or password are not provided
            res.json({status:400, message: 'Bad request.' });
        }

    });

// export router module
module.exports = router;

