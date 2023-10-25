package responsewriter

import (
	"io"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/noresponse"
	"github.com/plgd-dev/go-coap/v3/message/pool"
)

type Client interface {
	ReleaseMessage(msg *pool.Message)
}

// A ResponseWriter is used by an COAP handler to construct an COAP response.
type ResponseWriter[C Client] struct {
	noResponseValue *uint32
	response        *pool.Message
	cc              C
}

func New[C Client](response *pool.Message, cc C, requestOptions ...message.Option) *ResponseWriter[C] {
	var noResponseValue *uint32
	if len(requestOptions) > 0 {
		reqOpts := message.Options(requestOptions)
		v, err := reqOpts.GetUint32(message.NoResponse)
		if err == nil {
			noResponseValue = &v
		}
	}

	return &ResponseWriter[C]{
		response:        response,
		cc:              cc,
		noResponseValue: noResponseValue,
	}
}

// SetResponse simplifies the setup of the response for the request. ETags must be set via options. For advanced setup, use Message().
func (r *ResponseWriter[C]) SetResponse(code codes.Code, contentFormat message.MediaType, d io.ReadSeeker, opts ...message.Option) error {
	if r.noResponseValue != nil {
		err := noresponse.IsNoResponseCode(code, *r.noResponseValue)
		if err != nil {
			return err
		}
	}

	r.response.SetCode(code)
	r.response.ResetOptionsTo(opts)
	if d != nil {
		r.response.SetContentFormat(contentFormat)
		r.response.SetBody(d)
	}
	return nil
}

// SetMessage replaces the response message. The original message was released to the message pool, so don't use it any more. Ensure that Token, MessageID(udp), and Type(udp) messages are paired correctly.
func (r *ResponseWriter[C]) SetMessage(m *pool.Message) {
	r.cc.ReleaseMessage(r.response)
	r.response = m
}

// Message direct access to the response.
func (r *ResponseWriter[C]) Message() *pool.Message {
	return r.response
}

// Swap message in response without releasing.
func (r *ResponseWriter[C]) Swap(m *pool.Message) *pool.Message {
	tmp := r.response
	r.response = m
	return tmp
}

// CConn peer connection.
func (r *ResponseWriter[C]) Conn() C {
	return r.cc
}
