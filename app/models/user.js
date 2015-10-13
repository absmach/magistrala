// app/models/user.js

var mongoose     = require('mongoose');
var Schema       = mongoose.Schema;
var bcrypt = require('bcrypt');
var _ = require('lodash');

var UserSchema   = new Schema({
    firstName: {
        type: String,
        trim: true,
        default: '',
    },
    lastName: {
        type: String,
        trim: true,
        default: '',
    },
    displayName: {
        type: String,
        trim: true
    },
    email: {
        type: String,
        trim: true,
        unique: true,
    },
    password: {
        type: String,
        default: '',
        require: true
    },
    created: {
        type: Date,
        default: Date.now
    },
});

/**
 * Hook a pre save method to hash the user password
 */
UserSchema.pre('save', function(next) {
    var user = this;
    // only hash the password if it has been modified (or is new)
    if (!user.isModified('password')) return next()
    bcrypt.genSalt(10, function(err, salt) {
        bcrypt.hash(user.password, salt, function(err, hash) {
            if(err) return next(err);
            // set the hashed password back on our user document
            user.password = hash;
            //user.save();
            next();
        });
    });
});

// User methods

/**
 * Validate user password on singin and compare with hased password stored in DB
 */

 UserSchema.methods.validateUserPassword = function(plainPassword, cb) {
    var user = this;
     bcrypt.compare(plainPassword, user.password, function(err, isMatch) {
        if (err) return cb(err);
        cb(null, isMatch);
    });
 };

// TODO: check if user with email already exist

module.exports = mongoose.model('User', UserSchema);
