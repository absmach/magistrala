// Package coap provides a CoAP client and server.
package coap

import (
	"crypto/tls"
	"fmt"

	piondtls "github.com/pion/dtls/v2"
	"github.com/plgd-dev/go-coap/v3/dtls"
	dtlsServer "github.com/plgd-dev/go-coap/v3/dtls/server"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/go-coap/v3/net"
	"github.com/plgd-dev/go-coap/v3/options"
	"github.com/plgd-dev/go-coap/v3/tcp"
	tcpServer "github.com/plgd-dev/go-coap/v3/tcp/server"
	"github.com/plgd-dev/go-coap/v3/udp"
	udpServer "github.com/plgd-dev/go-coap/v3/udp/server"
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
		s := udp.NewServer(options.WithMux(handler))
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
		s := tcp.NewServer(options.WithMux(handler))
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
	s := tcp.NewServer(options.WithMux(handler))
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
	s := dtls.NewServer(options.WithMux(handler))
	return s.Serve(l)
}

// ListenAndServeWithOption Starts a server on address and network specified Invoke options
// for incoming queries. The options is only support tcpServer.Option and udpServer.Option
func ListenAndServeWithOptions(network, addr string, opts ...any) (err error) {
	tcpOptions := []tcpServer.Option{}
	udpOptions := []udpServer.Option{}
	for _, opt := range opts {
		switch o := opt.(type) {
		case tcpServer.Option:
			tcpOptions = append(tcpOptions, o)
		case udpServer.Option:
			udpOptions = append(udpOptions, o)
		default:
			return fmt.Errorf("only support tcpServer.Option and udpServer.Option")
		}
	}

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
		s := udp.NewServer(udpOptions...)
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
		s := tcp.NewServer(tcpOptions...)
		return s.Serve(l)
	default:
		return fmt.Errorf("invalid network (%v)", network)
	}
}

// ListenAndServeTCPTLSWithOptions Starts a server on address and network over TLS specified Invoke options
// for incoming queries.
func ListenAndServeTCPTLSWithOptions(network, addr string, config *tls.Config, opts ...tcpServer.Option) (err error) {
	l, err := net.NewTLSListener(network, addr, config)
	if err != nil {
		return err
	}
	defer func() {
		if errC := l.Close(); errC != nil && err == nil {
			err = errC
		}
	}()
	s := tcp.NewServer(opts...)
	return s.Serve(l)
}

// ListenAndServeDTLSWithOptions Starts a server on address and network over DTLS specified Invoke options
// for incoming queries.
func ListenAndServeDTLSWithOptions(network string, addr string, config *piondtls.Config, opts ...dtlsServer.Option) (err error) {
	l, err := net.NewDTLSListener(network, addr, config)
	if err != nil {
		return err
	}
	defer func() {
		if errC := l.Close(); errC != nil && err == nil {
			err = errC
		}
	}()
	s := dtls.NewServer(opts...)
	return s.Serve(l)
}
