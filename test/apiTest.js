/** Chai stuff */
var should      = require('chai').should;
var expect      = require('chai').expect;

/** Supertest for API */
var supertest   = require('supertest');
var server      = require('../server');
var api         = supertest(server);

/**
 * API test description
 */
describe('loading express', function () {
    /**
     * /status
     */
    it('responds to /status', function testSlash(done) {
        api
        .get('/status')
        .expect(200)
        .end(function(err, res){
            expect(res.body.status).to.equal("running");
            done();
        });
    });

    /**
     * /foo/bar
     */
    it('404 /foo/bar', function testPath(done) {
        api
        .get('/foo/bar')
        .expect(404, done);
    });
});

after(function(done) {
    server.close(done);    
});

