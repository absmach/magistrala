package mqtt

import (
	"io"
	"net"

	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mproxy/pkg/session"
)

// Proxy is main MQTT proxy struct
type Proxy struct {
	address string
	target  string
	event   session.Event
	logger  logger.Logger
}

func New(address, target string, event session.Event, logger logger.Logger) *Proxy {
	return &Proxy{
		address: address,
		target:  target,
		event:   event,
		logger:  logger,
	}
}

func (p Proxy) accept(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			p.logger.Warn("Accept error " + err.Error())
			continue
		}

		p.logger.Info("Accepted new client")
		go p.handleConnection(conn)
	}
}

func (p Proxy) handleConnection(inbound net.Conn) {
	defer inbound.Close()

	outbound, err := net.Dial("tcp", p.target)
	if err != nil {
		p.logger.Error("Cannot connect to remote broker " + p.target)
		return
	}
	defer outbound.Close()

	c := session.New(inbound, outbound, p.event, p.logger)

	if err := c.Stream(); err != io.EOF {
		p.logger.Warn("Broken connection for client: " + c.Client.ID + " with error: " + err.Error())
	}
}

// Proxy of the server, this will block.
func (p Proxy) Proxy() error {
	l, err := net.Listen("tcp", p.address)
	if err != nil {
		return err
	}
	defer l.Close()

	// Acceptor loop
	p.accept(l)

	p.logger.Info("Server Exiting...")
	return nil
}
