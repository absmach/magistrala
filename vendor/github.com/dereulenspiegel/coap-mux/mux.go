package mux

import (
	"net"
	"sync"

	coap "github.com/dustin/go-coap"
)

var (
	mutex   sync.RWMutex
	msgVars = make(map[*coap.Message]map[string]string)
)

func setVar(msg *coap.Message, name, data string) {
	mutex.Lock()
	r, exists := msgVars[msg]
	if !exists {
		r = make(map[string]string)
	}
	r[name] = data
	mutex.Unlock()
}

func setVars(msg *coap.Message, vars map[string]string) {
	mutex.Lock()
	msgVars[msg] = vars
	mutex.Unlock()
}

// Var retrieves a named variable for this message
func Var(msg *coap.Message, name string) string {
	mutex.RLock()
	var value = ""

	r, exists := msgVars[msg]
	if exists {
		value = r[name]
	}
	mutex.RUnlock()
	return value
}

func clearVars(msg *coap.Message) {
	mutex.Lock()
	delete(msgVars, msg)
	mutex.Unlock()
}

// Router handles routing CoAP Messages to the correct handlers. Currently
// It doesn't support default routes etc.
type Router struct {
	NotFoundHandler coap.Handler
	routes          []*Route
}

// NewRouter creates a new Router
func NewRouter() *Router {
	return &Router{routes: make([]*Route, 0, 50)}
}

// Match matches the incoming message against all routes. The first matching route
// wins.
func (r *Router) Match(msg *coap.Message, addr *net.UDPAddr, match *RouteMatch) bool {
	for _, route := range r.routes {
		if route.Match(msg, addr, match) {
			return true
		}
	}
	return false
}

// NewRoute creates a new unconfigured Route
func (r *Router) NewRoute() *Route {
	route := &Route{}
	r.routes = append(r.routes, route)
	return route
}

// Handle creates a new Route with the given handler and a matcher for the
// given path
func (r *Router) Handle(path string, handler coap.Handler) *Route {
	return r.NewRoute().Path(path).Handler(handler)
}

// Path creates a new Route with a matcher for the given path
func (r *Router) Path(tpl string) *Route {
	return r.NewRoute().Path(tpl)
}

// This method implements the interface for coap.Handler
func (r *Router) ServeCOAP(l *net.UDPConn, a *net.UDPAddr, m *coap.Message) *coap.Message {
	match := &RouteMatch{}
	var returnMessage *coap.Message
	if r.Match(m, a, match) {
		// TODO set vars
		setVars(m, match.Vars)
		returnMessage = match.Handler.ServeCOAP(l, a, m)
		clearVars(m)
	} else {
		returnMessage = r.NotFoundHandler.ServeCOAP(l, a, m)
	}
	return returnMessage
}
