// app/models/thing.js

var mongoose     = require('mongoose');
var Schema       = mongoose.Schema;

var ThingSchema   = new Schema({
    name: String
});

module.exports = mongoose.model('Thing', ThingSchema);
