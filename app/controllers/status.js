/**
 * STATUS
 */
exports.getStatus = function(req, res, next) {

    console.log("req.headers['x-auth-token'] = ", req.headers['x-auth-token']);

    var stat = {"status":"running"};
    res.send(stat);
    
    return next();
}
