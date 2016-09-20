/**
 * Copyright (c) Mainflux
 *
 * Mainflux server is licensed under an Apache license, version 2.0.
 * All rights not explicitly granted in the Apache license, version 2.0 are reserved.
 * See the included LICENSE file for more details.
 */

package servers

import (
	"fmt"
	"testing"
	"time"
	"log"
	"os"

	"github.com/mainflux/mainflux-lite/config"
	mfdb "github.com/mainflux/mainflux-lite/db"

	"github.com/kataras/iris"
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

	go HttpServer(cfg)

	// prepare test framework
	if ok := <-iris.Available; !ok {
		t.Fatal("Unexpected error: server cannot start, please report this as bug!!")
	}


	e := iris.Tester(t)
	r := e.Request("GET", "/status").Expect().Status(iris.StatusOK).JSON()
	fmt.Println("%v", r)

}

