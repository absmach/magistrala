/********************************
*** Multiplexer for Go        ***
*** Bone is under MIT license ***
*** Code by CodingFerret      ***
*** github.com/go-zoo         ***
*********************************/

package bone

import "net/http"

// Router is the same as a http.Handler
type Router interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}

// Register the route in the router
func (m *Mux) Register(method string, path string, handler http.Handler) *Route {
	return m.register(method, path, handler)
}

// GetFunc add a new route to the Mux with the Get method
func (m *Mux) GetFunc(path string, handler http.HandlerFunc) *Route {
	return m.register("GET", path, handler)
}

// PostFunc add a new route to the Mux with the Post method
func (m *Mux) PostFunc(path string, handler http.HandlerFunc) *Route {
	return m.register("POST", path, handler)
}

// PutFunc add a new route to the Mux with the Put method
func (m *Mux) PutFunc(path string, handler http.HandlerFunc) *Route {
	return m.register("PUT", path, handler)
}

// DeleteFunc add a new route to the Mux with the Delete method
func (m *Mux) DeleteFunc(path string, handler http.HandlerFunc) *Route {
	return m.register("DELETE", path, handler)
}

// HeadFunc add a new route to the Mux with the Head method
func (m *Mux) HeadFunc(path string, handler http.HandlerFunc) *Route {
	return m.register("HEAD", path, handler)
}

// PatchFunc add a new route to the Mux with the Patch method
func (m *Mux) PatchFunc(path string, handler http.HandlerFunc) *Route {
	return m.register("PATCH", path, handler)
}

// OptionsFunc add a new route to the Mux with the Options method
func (m *Mux) OptionsFunc(path string, handler http.HandlerFunc) *Route {
	return m.register("OPTIONS", path, handler)
}

// NotFoundFunc the mux custom 404 handler
func (m *Mux) NotFoundFunc(handler http.HandlerFunc) {
	m.notFound = handler
}
