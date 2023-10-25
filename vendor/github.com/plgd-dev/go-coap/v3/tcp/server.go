package tcp

import (
	"github.com/plgd-dev/go-coap/v3/tcp/server"
)

func NewServer(opt ...server.Option) *server.Server {
	return server.New(opt...)
}
