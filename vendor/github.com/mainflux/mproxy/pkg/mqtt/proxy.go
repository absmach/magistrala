package mqtt

import (
	"fmt"
	"io"
	"net"

	"github.com/google/uuid"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mproxy/pkg/events"
)

// Proxy is main MQTT proxy struct
type Proxy struct {
	host   string
	port   string
	target string
	event  events.Event
	logger logger.Logger
}

// New will setup a new Proxy struct after parsing the options
func New(host, port, targetHost, targetPort string, event events.Event, logger logger.Logger) *Proxy {
	return &Proxy{
		host:   host,
		port:   port,
		target: fmt.Sprintf("%s:%s", targetHost, targetPort),
		event:  event,
		logger: logger,
	}
}

func (p *Proxy) accept(l net.Listener) {
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

func (p *Proxy) handleConnection(inbound net.Conn) {
	defer inbound.Close()

	outbound, err := net.Dial("tcp", p.target)
	if err != nil {
		p.logger.Error("Cannot connect to remote broker " + p.target)
		return
	}
	defer outbound.Close()

	uuid, err := uuid.NewRandom()
	if err != nil {
		return
	}

	s := newSession(uuid.String(), inbound, outbound, p.event, p.logger)
	if err := s.stream(); err != io.EOF {
		p.logger.Warn("Exited session " + s.id + "with error: " + err.Error())
	}
	s.logger.Info("Session " + s.id + "closed: " + s.outbound.LocalAddr().String())
}

// Proxy of the server, this will block.
func (p *Proxy) Proxy() error {
	addr := fmt.Sprintf("%s:%s", p.host, p.port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer l.Close()

	// Acceptor loop
	p.accept(l)

	p.logger.Info("Server Exiting...")
	return nil
}
