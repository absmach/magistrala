package dtls

import "github.com/plgd-dev/go-coap/v3/dtls/server"

func NewServer(opt ...server.Option) *server.Server {
	return server.New(opt...)
}
