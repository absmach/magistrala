// Package mainflux acts as an umbrella package containing multiple different
// microservices / deliverables. It provides the top-level platform versioning.
package mainflux

import (
	"encoding/json"
	"net/http"
)

const version string = "1.0.0"

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
