package mqtt

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"

	"github.com/mainflux/mproxy/pkg/logger"
	"github.com/mainflux/mproxy/pkg/session"
	mptls "github.com/mainflux/mproxy/pkg/tls"
)

// Proxy is main MQTT proxy struct
type Proxy struct {
	address string
	target  string
	handler session.Handler
	logger  logger.Logger
	dialer  net.Dialer
}

// New returns a new mqtt Proxy instance.
func New(address, target string, handler session.Handler, logger logger.Logger) *Proxy {
	return &Proxy{
		address: address,
		target:  target,
		handler: handler,
		logger:  logger,
	}
}

func (p Proxy) accept(ctx context.Context, l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			p.logger.Warn("Accept error " + err.Error())
			continue
		}

		p.logger.Info("Accepted new client")
		go p.handle(ctx, conn)
	}
}

func (p Proxy) handle(ctx context.Context, inbound net.Conn) {
	defer p.close(inbound)
	outbound, err := p.dialer.Dial("tcp", p.target)
	if err != nil {
		p.logger.Error("Cannot connect to remote broker " + p.target + " due to: " + err.Error())
		return
	}
	defer p.close(outbound)

	clientCert, err := mptls.ClientCert(inbound)
	if err != nil {
		p.logger.Error("Failed to get client certificate: " + err.Error())
		return
	}

	if err = session.Stream(ctx, inbound, outbound, p.handler, clientCert); err != io.EOF {
		p.logger.Warn(err.Error())
	}
}

// Listen of the server, this will block.
func (p Proxy) Listen(ctx context.Context) error {
	l, err := net.Listen("tcp", p.address)
	if err != nil {
		return err
	}
	defer l.Close()

	// Acceptor loop
	p.accept(ctx, l)

	p.logger.Info("Server Exiting...")
	return nil
}

// ListenTLS - version of Listen with TLS encryption
func (p Proxy) ListenTLS(ctx context.Context, tlsCfg *tls.Config) error {

	l, err := tls.Listen("tcp", p.address, tlsCfg)
	if err != nil {
		return err
	}
	defer l.Close()

	// Acceptor loop
	p.accept(ctx, l)

	p.logger.Info("Server Exiting...")
	return nil
}

func (p Proxy) close(conn net.Conn) {
	if err := conn.Close(); err != nil {
		p.logger.Warn(fmt.Sprintf("Error closing connection %s", err.Error()))
	}
}
