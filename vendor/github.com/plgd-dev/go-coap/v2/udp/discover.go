package udp

import (
	"context"
	"fmt"
	"net"

	"github.com/plgd-dev/go-coap/v2/udp/client"
	"github.com/plgd-dev/go-coap/v2/udp/message/pool"
)

var defaultMulticastOptions = multicastOptions{
	hopLimit: 2,
}

type multicastOptions struct {
	hopLimit int
}

// A MulticastOption sets options such as hop limit, etc.
type MulticastOption interface {
	apply(*multicastOptions)
}

// Discover sends GET to multicast address and wait for responses until context timeouts or server shutdown.
func (s *Server) Discover(ctx context.Context, multicastAddr, path string, receiverFunc func(cc *client.ClientConn, resp *pool.Message), opts ...MulticastOption) error {
	req, err := client.NewGetRequest(ctx, path)
	if err != nil {
		return fmt.Errorf("cannot create discover request: %w", err)
	}
	req.SetMessageID(s.getMID())
	defer pool.ReleaseMessage(req)
	return s.DiscoveryRequest(req, multicastAddr, receiverFunc, opts...)
}

// DiscoveryRequest sends request to multicast addressand wait for responses until request timeouts or server shutdown.
func (s *Server) DiscoveryRequest(req *pool.Message, multicastAddr string, receiverFunc func(cc *client.ClientConn, resp *pool.Message), opts ...MulticastOption) error {
	token := req.Token()
	if len(token) == 0 {
		return fmt.Errorf("invalid token")
	}
	cfg := defaultMulticastOptions
	for _, o := range opts {
		o.apply(&cfg)
	}
	c := s.conn()
	if c == nil {
		return fmt.Errorf("server doesn't serve connection")
	}
	addr, err := net.ResolveUDPAddr(c.Network(), multicastAddr)
	if err != nil {
		return fmt.Errorf("cannot resolve address: %w", err)
	}
	if !addr.IP.IsMulticast() {
		return fmt.Errorf("invalid multicast address")
	}
	data, err := req.Marshal()
	if err != nil {
		return fmt.Errorf("cannot marshal req: %w", err)
	}
	s.multicastRequests.Store(token.String(), req)
	defer s.multicastRequests.Delete(token.String())
	err = s.multicastHandler.Insert(token, func(w *client.ResponseWriter, r *pool.Message) {
		receiverFunc(w.ClientConn(), r)
	})
	if err != nil {
		return err
	}
	defer s.multicastHandler.Pop(token)

	err = c.WriteMulticast(req.Context(), addr, cfg.hopLimit, data)
	if err != nil {
		return err
	}
	select {
	case <-req.Context().Done():
		return nil
	case <-s.ctx.Done():
		return fmt.Errorf("server was closed: %w", req.Context().Err())
	}
}
