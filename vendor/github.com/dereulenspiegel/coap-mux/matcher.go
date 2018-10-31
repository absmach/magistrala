package mux

import (
	"fmt"
	"net"
	"strings"

	coap "github.com/dustin/go-coap"
)

// methodMatcher matches the request against HTTP methods.
type methodMatcher []coap.COAPCode

func (m methodMatcher) Match(msg *coap.Message, addr *net.UDPAddr) bool {
	for _, v := range m {
		if v == msg.Code {
			return true
		}
	}
	return false
}

type coaptypeMatcher []coap.COAPType

func (m coaptypeMatcher) Match(msg *coap.Message, addr *net.UDPAddr) bool {
	for _, v := range m {
		if v == msg.Type {
			return true
		}
	}
	return false
}

// addRegexpMatcher adds a host or path matcher and builder to a route.
func (r *Route) addRegexpMatcher(tpl string) error {
	if len(tpl) == 0 || tpl[0] != '/' {
		return fmt.Errorf("mux: path must start with a slash, got %q", tpl)
	}
	if r.regexp != nil {
		tpl = strings.TrimRight(r.regexp.template, "/") + tpl
	}
	rr, err := newRouteRegexp(tpl)
	if err != nil {
		return err
	}
	r.regexp = rr
	r.addMatcher(rr)
	return nil
}
