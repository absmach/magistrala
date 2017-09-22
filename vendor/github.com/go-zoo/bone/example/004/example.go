// +build go1.7

package main

import (
	"context"
	"net/http"

	"github.com/go-zoo/bone"
)

func main() {
	mux := bone.New()

	mux.GetFunc("/ctx/:var", rootHandler)

	http.ListenAndServe(":8080", mux)
}

func rootHandler(rw http.ResponseWriter, req *http.Request) {
	ctx := context.WithValue(req.Context(), "var", bone.GetValue(req, "var"))
	subHandler(rw, req.WithContext(ctx))
}

func subHandler(rw http.ResponseWriter, req *http.Request) {
	val := req.Context().Value("var")
	rw.Write([]byte(val.(string)))
}
