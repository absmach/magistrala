package mux

import (
	"io"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/net/responsewriter"
)

// ToHandler converts mux handler to udp/dtls/tcp handler.
func ToHandler[C Conn](m Handler) func(w *responsewriter.ResponseWriter[C], r *pool.Message) {
	return func(w *responsewriter.ResponseWriter[C], r *pool.Message) {
		muxw := &muxResponseWriter[C]{
			w: w,
		}
		m.ServeCOAP(muxw, &Message{
			Message:     r,
			RouteParams: new(RouteParams),
		})
	}
}

type muxResponseWriter[C Conn] struct {
	w *responsewriter.ResponseWriter[C]
}

// SetResponse simplifies the setup of the response for the request. ETags must be set via options. For advanced setup, use Message().
func (w *muxResponseWriter[C]) SetResponse(code codes.Code, contentFormat message.MediaType, d io.ReadSeeker, opts ...message.Option) error {
	return w.w.SetResponse(code, contentFormat, d, opts...)
}

// Conn peer connection.
func (w *muxResponseWriter[C]) Conn() Conn {
	return w.w.Conn()
}

// Message direct access to the response.
func (w *muxResponseWriter[C]) Message() *pool.Message {
	return w.w.Message()
}

// SetMessage replaces the response message. Ensure that Token, MessageID(udp), and Type(udp) messages are paired correctly.
func (w *muxResponseWriter[C]) SetMessage(msg *pool.Message) {
	w.w.SetMessage(msg)
}
