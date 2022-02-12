package client

import (
	"io"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	udpMessage "github.com/plgd-dev/go-coap/v2/udp/message"
	"github.com/plgd-dev/go-coap/v2/udp/message/pool"
)

func HandlerFuncToMux(m mux.Handler) HandlerFunc {
	h := func(w *ResponseWriter, r *pool.Message) {
		muxw := &muxResponseWriter{
			w: w,
		}
		muxr, err := pool.ConvertTo(r)
		if err != nil {
			return
		}
		m.ServeCOAP(muxw, &mux.Message{
			Message:        muxr,
			SequenceNumber: r.Sequence(),
			IsConfirmable:  r.Type() == udpMessage.Confirmable,
			RouteParams:    new(mux.RouteParams),
		})
	}
	return h
}

type muxResponseWriter struct {
	w *ResponseWriter
}

func (w *muxResponseWriter) SetResponse(code codes.Code, contentFormat message.MediaType, d io.ReadSeeker, opts ...message.Option) error {
	return w.w.SetResponse(code, contentFormat, d, opts...)
}

func (w *muxResponseWriter) Client() mux.Client {
	return w.w.ClientConn().Client()
}
