package mainflux

import (
	"encoding/json"
	"net/http"
)

const version string = "0.2.2"

type response struct {
	Version string
	Service string
}

// Version exposes an HTTP handler for retrieving service version.
func Version(service string) http.HandlerFunc {
	return http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		res := response{Version: version, Service: service}

		data, _ := json.Marshal(res)

		rw.Write(data)
	})
}
