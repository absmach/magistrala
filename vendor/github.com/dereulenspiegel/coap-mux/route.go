package mux

import (
	"net"

	coap "github.com/dustin/go-coap"
)

type RouteMatch struct {
	Handler coap.Handler
	Vars    map[string]string
}

// Matcher is the interface to implement if you want to match additional
// properties
type Matcher interface {
	Match(*coap.Message, *net.UDPAddr) bool
}

// Route represents a set of Matchers which all have to match for a handler to
// be executed
type Route struct {
	name     string
	handler  coap.Handler
	matchers []Matcher
	regexp   *routeRegexp
}

// getRegexpGroup returns regexp definitions from this route.
func (r *Route) getRegexp() *routeRegexp {
	if r.regexp == nil {
		r.regexp = new(routeRegexp)
	}
	return r.regexp
}

// Name sets a name for this Route. Currently we don't do anything with the name
func (r *Route) Name(name string) *Route {
	r.name = name
	return r
}

// Match matches this route against the received packet and the peers UDPAddr.
// All matchers have to return true to, for the route to be matched.
func (r *Route) Match(msg *coap.Message, addr *net.UDPAddr, match *RouteMatch) bool {
	for _, matcher := range r.matchers {
		if matched := matcher.Match(msg, addr); !matched {
			return false
		}
	}
	match.Handler = r.handler
	if match.Vars == nil {
		match.Vars = make(map[string]string)
	}
	// Set variables.
	if r.regexp != nil {
		r.regexp.setMatch(msg, match, r)
	}
	return true
}

// Handler sets the handler for this route
func (r *Route) Handler(h coap.Handler) *Route {
	r.handler = h
	return r
}

// addMatcher adds a matcher to the route.
func (r *Route) addMatcher(m Matcher) *Route {
	r.matchers = append(r.matchers, m)
	return r
}

// Matches adds a custom matcher to the Routes matchers
func (r *Route) Matches(matcher Matcher) *Route {
	r.addMatcher(matcher)
	return r
}

// Methods matches this Route for COAPCodes like GET, POST etc.
func (r *Route) Methods(methods ...coap.COAPCode) *Route {
	return r.addMatcher(methodMatcher(methods))
}

// Path matches this Route against a path. The path may contain variables
// enclosed in curly braces. Example:
//     /api/{id}/
func (r *Route) Path(tpl string) *Route {
	r.addRegexpMatcher(tpl)
	return r
}

// COAPType matches this Route against the type of the message
func (r *Route) COAPType(coapTypes ...coap.COAPType) *Route {
	r.addMatcher(coaptypeMatcher(coapTypes))
	return r
}
