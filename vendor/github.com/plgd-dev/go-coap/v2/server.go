// Package coap provides a CoAP client and server.
package coap

import (
	"crypto/tls"
	"fmt"

	piondtls "github.com/pion/dtls/v2"
	"github.com/plgd-dev/go-coap/v2/dtls"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/go-coap/v2/net"
	"github.com/plgd-dev/go-coap/v2/tcp"
	"github.com/plgd-dev/go-coap/v2/udp"
)

// ListenAndServe Starts a server on address and network specified Invoke handler
// for incoming queries.
func ListenAndServe(network string, addr string, handler mux.Handler) (err error) {
	switch network {
	case "udp", "udp4", "udp6", "":
		l, err := net.NewListenUDP(network, addr)
		if err != nil {
			return err
		}
		defer func() {
			if errC := l.Close(); errC != nil && err == nil {
				err = errC
			}
		}()
		s := udp.NewServer(udp.WithMux(handler))
		return s.Serve(l)
	case "tcp", "tcp4", "tcp6":
		l, err := net.NewTCPListener(network, addr)
		if err != nil {
			return err
		}
		defer func() {
			if errC := l.Close(); errC != nil && err == nil {
				err = errC
			}
		}()
		s := tcp.NewServer(tcp.WithMux(handler))
		return s.Serve(l)
	default:
		return fmt.Errorf("invalid network (%v)", network)
	}
}

// ListenAndServeTCPTLS Starts a server on address and network over TLS specified Invoke handler
// for incoming queries.
func ListenAndServeTCPTLS(network, addr string, config *tls.Config, handler mux.Handler) (err error) {
	l, err := net.NewTLSListener(network, addr, config)
	if err != nil {
		return err
	}
	defer func() {
		if errC := l.Close(); errC != nil && err == nil {
			err = errC
		}
	}()
	s := tcp.NewServer(tcp.WithMux(handler))
	return s.Serve(l)
}

// ListenAndServeDTLS Starts a server on address and network over DTLS specified Invoke handler
// for incoming queries.
func ListenAndServeDTLS(network string, addr string, config *piondtls.Config, handler mux.Handler) (err error) {
	l, err := net.NewDTLSListener(network, addr, config)
	if err != nil {
		return err
	}
	defer func() {
		if errC := l.Close(); errC != nil && err == nil {
			err = errC
		}
	}()
	s := dtls.NewServer(dtls.WithMux(handler))
	return s.Serve(l)
}
