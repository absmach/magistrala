var jwt = require('jsonwebtoken');

var config = require('../config/config');

var token = jwt.sign({foo: 'bar'}, config.tokenSecret, {
    expiresInMinutes: config.userTokenExpirePeriod
});

console.log(token);
