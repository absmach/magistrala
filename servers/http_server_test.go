/**
 * Copyright (c) Mainflux
 *
 * Mainflux server is licensed under an Apache license, version 2.0.
 * All rights not explicitly granted in the Apache license, version 2.0 are reserved.
 * See the included LICENSE file for more details.
 */

package servers

import (
	"testing"
	"time"
	"log"
	"os"
    "net/http"
    "net/http/httptest"

	"github.com/mainflux/mainflux/config"
	"github.com/mainflux/mainflux/controllers"
	mfdb "github.com/mainflux/mainflux/db"

	"github.com/ory-am/dockertest"
	"gopkg.in/mgo.v2"
)


func TestMain(m *testing.M) {
	// We are in testing - notify the program
	// so that it is not confused if some other commad line 
	// arguments come in - for example when test is started with `go test -v ./...`
	// which is what Travis does
	os.Setenv("TEST_ENV", "1")

	var db *mgo.Session
	c, err := dockertest.ConnectToMongoDB(15, time.Millisecond*500, func(url string) bool {
		// This callback function checks if the image's process is responsive.
		// Sometimes, docker images are booted but the process (in this case MongoDB) is still doing maintenance
		// before being fully responsive which might cause issues like "TCP Connection reset by peer".
		var err error
		db, err = mgo.Dial(url)
		if err != nil {
			return false
		}

		// Sometimes, dialing the database is not enough because the port is already open but the process is not responsive.
		// Most database conenctors implement a ping function which can be used to test if the process is responsive.
		// Alternatively, you could execute a query to see if an error occurs or not.
		return db.Ping() == nil
	})
	if err != nil {
		log.Fatalf("Could not connect to database: %s", err)
	}

	// Set-up DB
	mfdb.SetMainSession(db)
	mfdb.SetMainDb("mainflux_test")

	// Run tests
	result := m.Run()

	// Close database connection.
	db.Close()

	// Clean up image.
	c.KillRemove()

	// Exit tests.
	os.Exit(result)
}

func TestServer(t *testing.T) {

	// Config
	var cfg config.Config
	cfg.Parse()


	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
    // pass 'nil' as the third parameter.
    req, err := http.NewRequest("GET", "/status", nil)
    if err != nil {
		t.Fatal(err)
    }

    // We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
    rr := httptest.NewRecorder()
    handler := http.HandlerFunc(controllers.GetStatus)

    // Our handlers satisfy http.Handler, so we can call their ServeHTTP method 
    // directly and pass in our Request and ResponseRecorder.
    handler.ServeHTTP(rr, req)

    // Check the status code is what we expect.
    if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
    }

    // Check the response body is what we expect.
    expected := `{"running": true}`
    if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
    }
}

