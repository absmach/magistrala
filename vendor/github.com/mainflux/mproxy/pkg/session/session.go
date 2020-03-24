package session

import (
	"net"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/mainflux/mainflux/logger"
)

const (
	up direction = iota
	down
)

type direction int

type Session struct {
	logger   logger.Logger
	inbound  net.Conn
	outbound net.Conn
	event    Event
	Client   Client
}

func New(inbound, outbound net.Conn, event Event, logger logger.Logger) *Session {
	return &Session{
		logger:   logger,
		inbound:  inbound,
		outbound: outbound,
		event:    event,
	}
}

func (s Session) Stream() error {
	// In parallel read from client, send to broker
	// and read from broker, send to client
	errs := make(chan error, 2)

	go s.stream(up, s.inbound, s.outbound, errs)
	go s.stream(down, s.outbound, s.inbound, errs)

	err := <-errs
	s.event.Disconnect(&s.Client)
	return err
}

func (s Session) stream(dir direction, r, w net.Conn, errs chan error) {
	for {
		// Read from one connection
		pkt, err := packets.ReadPacket(r)
		if err != nil {
			errs <- err
			return
		}

		if dir == up {
			if err := s.authorize(pkt); err != nil {
				errs <- err
				return
			}
		}

		// Send to another
		if err := pkt.Write(w); err != nil {
			errs <- err
			return
		}

		if dir == up {
			s.notify(pkt)
		}
	}
}

func (s *Session) authorize(pkt packets.ControlPacket) error {
	switch p := pkt.(type) {
	case *packets.ConnectPacket:
		s.Client = Client{
			ID:       p.ClientIdentifier,
			Username: p.Username,
			Password: p.Password,
		}
		if err := s.event.AuthConnect(&s.Client); err != nil {
			return err
		}
		// Copy back to the packet in case values are changed by Event handler.
		// This is specific to CONN, as only that package type has credentials.
		p.ClientIdentifier = s.Client.ID
		p.Username = s.Client.Username
		p.Password = s.Client.Password
		return nil
	case *packets.PublishPacket:
		return s.event.AuthPublish(&s.Client, &p.TopicName, &p.Payload)
	case *packets.SubscribePacket:
		return s.event.AuthSubscribe(&s.Client, &p.Topics)
	default:
		return nil
	}
}

func (s Session) notify(pkt packets.ControlPacket) {
	switch p := pkt.(type) {
	case *packets.ConnectPacket:
		s.event.Connect(&s.Client)
	case *packets.PublishPacket:
		s.event.Publish(&s.Client, &p.TopicName, &p.Payload)
	case *packets.SubscribePacket:
		s.event.Subscribe(&s.Client, &p.Topics)
	case *packets.UnsubscribePacket:
		s.event.Unsubscribe(&s.Client, &p.Topics)
	default:
		return
	}
}
