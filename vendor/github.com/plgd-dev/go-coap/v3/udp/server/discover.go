package server

import (
	"context"
	"fmt"
	"net"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	coapNet "github.com/plgd-dev/go-coap/v3/net"
	"github.com/plgd-dev/go-coap/v3/net/responsewriter"
	"github.com/plgd-dev/go-coap/v3/pkg/errors"
	"github.com/plgd-dev/go-coap/v3/udp/client"
	"github.com/plgd-dev/go-coap/v3/udp/coder"
)

// Discover sends GET to multicast or unicast address and waits for responses until context timeouts or server shutdown.
// For unicast there is a difference against the Dial. The Dial is connection-oriented and it means that, if you send a request to an address, the peer must send the response from the same
// address where was request sent. For Discover it allows the client to send a response from another address where was request send.
// By default it is sent over all network interfaces and all compatible source IP addresses with hop limit 1.
// Via opts you can specify the network interface, source IP address, and hop limit.
func (s *Server) Discover(ctx context.Context, address, path string, receiverFunc func(cc *client.Conn, resp *pool.Message), opts ...coapNet.MulticastOption) error {
	token, err := s.cfg.GetToken()
	if err != nil {
		return fmt.Errorf("cannot get token: %w", err)
	}
	req := s.cfg.MessagePool.AcquireMessage(ctx)
	defer s.cfg.MessagePool.ReleaseMessage(req)
	err = req.SetupGet(path, token)
	if err != nil {
		return fmt.Errorf("cannot create discover request: %w", err)
	}
	req.SetMessageID(s.cfg.GetMID())
	req.SetType(message.NonConfirmable)
	return s.DiscoveryRequest(req, address, receiverFunc, opts...)
}

// DiscoveryRequest sends request to multicast/unicast address and wait for responses until request timeouts or server shutdown.
// For unicast there is a difference against the Dial. The Dial is connection-oriented and it means that, if you send a request to an address, the peer must send the response from the same
// address where was request sent. For Discover it allows the client to send a response from another address where was request send.
// By default it is sent over all network interfaces and all compatible source IP addresses with hop limit 1.
// Via opts you can specify the network interface, source IP address, and hop limit.
func (s *Server) DiscoveryRequest(req *pool.Message, address string, receiverFunc func(cc *client.Conn, resp *pool.Message), opts ...coapNet.MulticastOption) error {
	token := req.Token()
	if len(token) == 0 {
		return fmt.Errorf("invalid token")
	}
	c := s.conn()
	if c == nil {
		return fmt.Errorf("server doesn't serve connection")
	}
	addr, err := net.ResolveUDPAddr(c.Network(), address)
	if err != nil {
		return fmt.Errorf("cannot resolve address: %w", err)
	}

	data, err := req.MarshalWithEncoder(coder.DefaultCoder)
	if err != nil {
		return fmt.Errorf("cannot marshal req: %w", err)
	}
	s.multicastRequests.Store(token.Hash(), req)
	defer s.multicastRequests.Delete(token.Hash())
	if _, loaded := s.multicastHandler.LoadOrStore(token.Hash(), func(w *responsewriter.ResponseWriter[*client.Conn], r *pool.Message) {
		receiverFunc(w.Conn(), r)
	}); loaded {
		return errors.ErrKeyAlreadyExists
	}
	defer func() {
		_, _ = s.multicastHandler.LoadAndDelete(token.Hash())
	}()

	if addr.IP.IsMulticast() {
		err = c.WriteMulticast(req.Context(), addr, data, opts...)
		if err != nil {
			return err
		}
	} else {
		err = c.WriteWithContext(req.Context(), addr, data)
		if err != nil {
			return err
		}
	}

	select {
	case <-req.Context().Done():
		return nil
	case <-s.ctx.Done():
		return fmt.Errorf("server was closed: %w", s.ctx.Err())
	}
}
