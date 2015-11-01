var express = require('express');
var router = express.Router();              // get an instance of the express Router

// on routes that end in /things
// ----------------------------------------------------
router.route('/')

    // get the status (accessed at GET http://localhost:8080/status)
    .get(function(req, res) {
        var stat = {"status":"running"}
        res.send(stat);
    });

// export router module
module.exports = router;
