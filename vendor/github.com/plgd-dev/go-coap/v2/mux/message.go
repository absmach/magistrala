package mux

import "github.com/plgd-dev/go-coap/v2/message"

// RouteParams contains all the information related to a route
type RouteParams struct {
	Path         string
	Vars         map[string]string
	PathTemplate string
}

// Message contains message with sequence number.
type Message struct {
	*message.Message
	// SequenceNumber identifies the order of the message from a TCP connection. For UDP it is just for debugging.
	SequenceNumber uint64
	// IsConfirmable indicates that a UDP message is confirmable. For TCP the value has no semantic.
	// When a handler blocks a confirmable message, the client might decide to issue a re-transmission.
	// Long running handlers can be handled in a go routine and send the response via w.Client().
	// The ACK is sent as soon as the handler returns.
	IsConfirmable bool
	RouteParams   *RouteParams
}
