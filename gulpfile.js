/* File: gulpfile.js */

// grab our packages
var gulp   = require('gulp');
var jshint = require('gulp-jshint');
var nodemon = require('gulp-nodemon');

// define the default task and add the watch task to it
gulp.task('default', ['watch']);

// configure the jshint task
gulp.task('jshint', function() {
    return gulp.src('app/**/*.js')
        .pipe(jshint())
        .pipe(jshint.reporter('jshint-stylish'));
});

// configure which files to watch and what tasks to use on file changes
gulp.task('watch', function() {
    gulp.watch('app/**/*.js', ['jshint']);
    
    // Start up the server and have it reload when anything in the
    // ./build/ directory changes
    nodemon({script: 'server.js', watch: 'app/**'});
});
