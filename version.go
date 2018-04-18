package mainflux

import (
	"encoding/json"
	"net/http"
)

const version string = "0.2.0"

type response struct {
	Version string
}

// Version exposes an HTTP handler for retrieving service version.
func Version() http.HandlerFunc {
	return http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		res := response{Version: version}

		data, _ := json.Marshal(res)

		rw.Write(data)
	})
}
