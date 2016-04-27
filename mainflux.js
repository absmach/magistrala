/**
 * Copyright (c) Mainflux
 *
 * Mainflux server is licensed under an Apache license, version 2.0 license.
 * All rights not explicitly granted in the Apache license, version 2.0 are reserved.
 * See the included LICENSE file for more details.
 */
var domain = require('domain');
var config = require('./config');
var log = require('./app/logger');

var httpApi = require('./httpServer');
var mqttApi = require('./mqttServer');
var mqttApi = require('./wsServer');

var mainflux = {};


var banner = `
oocccdMMMMMMMMMWOkkkkoooolcclX
llc:::0MMMMMMMM0xxxxxdlllc:::d
lll:::cXMMMMMMXxxxxxxxdlllc:::
lllc:::cXMMMMNkxxxdxxxxolllc::
olllc:::oWMMNkxxxdloxxxxolllc:   ##     ##    ###    #### ##    ## ######## ##       ##     ## ##     ##
xolllc:::xWWOxxxdllloxxxxolllc   ###   ###   ## ##    ##  ###   ## ##       ##       ##     ##  ##   ## 
xxolllc:::x0xxxdllll:oxxxxllll   #### ####  ##   ##   ##  ####  ## ##       ##       ##     ##   ## ##  
xxxolllc::oxxxxllll:::dxxxdlll   ## ### ## ##     ##  ##  ## ## ## ######   ##       ##     ##    ###   
xxxdllll:lxxxxolllc:::Okxxxdll   ##     ## #########  ##  ##  #### ##       ##       ##     ##   ## ##  
0xxxdllloxxxxolllc:::OMNkxxxdl   ##     ## ##     ##  ##  ##   ### ##       ##       ##     ##  ##   ## 
W0xxxdllxxxxolllc:::xMMMXxxxxd   ##     ## ##     ## #### ##    ## ##       ########  #######  ##     ##
MWOxxxdxxxxdlllc:::oWMMMMKxxxx
MMWkxxxxxxdlllc:::oNMMMMMM0xxx
MMMXxxxxxdllllc::cXMMMMMMMWOxx
MMMM0xxxxolllc:::kMMMMMMMMMXxx
`

console.log(banner);

/**
 * Exports
 */
module.exports = mainflux;
