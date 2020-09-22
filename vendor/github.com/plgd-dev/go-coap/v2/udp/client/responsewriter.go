package client

import (
	"io"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/message/noresponse"
	udpMessage "github.com/plgd-dev/go-coap/v2/udp/message"
	"github.com/plgd-dev/go-coap/v2/udp/message/pool"
)

// A ResponseWriter interface is used by an COAP handler to construct an COAP response.
type ResponseWriter struct {
	noResponseValue *uint32
	response        *pool.Message
	cc              *ClientConn
}

func NewResponseWriter(response *pool.Message, cc *ClientConn, requestOptions message.Options) *ResponseWriter {
	var noResponseValue *uint32
	v, err := requestOptions.GetUint32(message.NoResponse)
	if err == nil {
		noResponseValue = &v
	}

	return &ResponseWriter{
		response:        response,
		cc:              cc,
		noResponseValue: noResponseValue,
	}
}

func (r *ResponseWriter) SetResponse(code codes.Code, contentFormat message.MediaType, d io.ReadSeeker, opts ...message.Option) error {
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
		if !r.response.HasOption(message.ETag) {
			etag, err := message.GetETag(d)
			if err != nil {
				return err
			}
			r.response.SetOptionBytes(message.ETag, etag)
		}
	}
	return nil
}

func (r *ResponseWriter) ClientConn() *ClientConn {
	return r.cc
}

func (r *ResponseWriter) SendReset() {
	r.response.Reset()
	r.response.SetCode(codes.Empty)
	r.response.SetType(udpMessage.Reset)
}
