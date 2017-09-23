package main

import (
	"log"
	"net/http"

	"github.com/go-zoo/bone"
)

var (
	mux = bone.New()
)

func main() {
	// Custom 404
	mux.NotFoundFunc(Handler404)
	// Handle with any http method, Handle takes http.Handler as argument.
	mux.Handle("/index", http.HandlerFunc(homeHandler))
	mux.Handle("/index/:var/info/:test", http.HandlerFunc(varHandler))
	// Get, Post etc... takes http.HandlerFunc as argument.
	mux.Post("/home", http.HandlerFunc(homeHandler))
	mux.Get("/home/:var", http.HandlerFunc(varHandler))

	mux.GetFunc("/test/*", func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(req.RequestURI))
	})

	// Start Listening
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func homeHandler(rw http.ResponseWriter, req *http.Request) {
	rw.Write([]byte("WELCOME HOME"))
}

func varHandler(rw http.ResponseWriter, req *http.Request) {
	varr := bone.GetValue(req, "var")
	test := bone.GetValue(req, "test")

	rw.Write([]byte(varr + " " + test))
}

func Handler404(rw http.ResponseWriter, req *http.Request) {
	rw.Write([]byte("These are not the droids you're looking for ..."))
}
