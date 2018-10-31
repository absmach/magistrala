package coap

import (
	"net"
)

// ServeMux provides mappings from a common endpoint to handlers by
// request path.
type ServeMux struct {
	m map[string]muxEntry
}

type muxEntry struct {
	h       Handler
	pattern string
}

// NewServeMux creates a new ServeMux.
func NewServeMux() *ServeMux { return &ServeMux{m: make(map[string]muxEntry)} }

// Does path match pattern?
func pathMatch(pattern, path string) bool {
	if len(pattern) == 0 {
		// should not happen
		return false
	}
	n := len(pattern)
	if pattern[n-1] != '/' {
		return pattern == path
	}
	return len(path) >= n && path[0:n] == pattern
}

// Find a handler on a handler map given a path string
// Most-specific (longest) pattern wins
func (mux *ServeMux) match(path string) (h Handler, pattern string) {
	var n = 0
	for k, v := range mux.m {
		if !pathMatch(k, path) {
			continue
		}
		if h == nil || len(k) > n {
			n = len(k)
			h = v.h
			pattern = v.pattern
		}
	}
	return
}

func notFoundHandler(l *net.UDPConn, a *net.UDPAddr, m *Message) *Message {
	if m.IsConfirmable() {
		return &Message{
			Type: Acknowledgement,
			Code: NotFound,
		}
	}
	return nil
}

var _ = Handler(&ServeMux{})

// ServeCOAP handles a single COAP message.  The message arrives from
// the given listener having originated from the given UDPAddr.
func (mux *ServeMux) ServeCOAP(l *net.UDPConn, a *net.UDPAddr, m *Message) *Message {
	h, _ := mux.match(m.PathString())
	if h == nil {
		h, _ = funcHandler(notFoundHandler), ""
	}
	// TODO:  Rewrite path?
	return h.ServeCOAP(l, a, m)
}

// Handle configures a handler for the given path.
func (mux *ServeMux) Handle(pattern string, handler Handler) {
	for pattern != "" && pattern[0] == '/' {
		pattern = pattern[1:]
	}

	if pattern == "" {
		panic("http: invalid pattern " + pattern)
	}
	if handler == nil {
		panic("http: nil handler")
	}

	mux.m[pattern] = muxEntry{h: handler, pattern: pattern}
}

// HandleFunc configures a handler for the given path.
func (mux *ServeMux) HandleFunc(pattern string,
	f func(l *net.UDPConn, a *net.UDPAddr, m *Message) *Message) {
	mux.Handle(pattern, FuncHandler(f))
}
