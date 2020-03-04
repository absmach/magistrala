package mqtt

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

type session struct {
	logger   logger.Logger
	inbound  net.Conn
	outbound net.Conn
	event    Event
	client   Client
}

func newSession(inbound, outbound net.Conn, event Event, logger logger.Logger) *session {
	return &session{
		logger:   logger,
		inbound:  inbound,
		outbound: outbound,
		event:    event,
	}
}

func (s *session) stream() error {
	// In parallel read from client, send to broker
	// and read from broker, send to client
	errs := make(chan error, 2)

	go s.streamUnidir(up, s.inbound, s.outbound, errs)
	go s.streamUnidir(down, s.outbound, s.inbound, errs)

	err := <-errs
	s.event.Disconnect(&s.client)
	return err
}

func (s *session) streamUnidir(dir direction, r, w net.Conn, errs chan error) {
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

func (s *session) authorize(pkt packets.ControlPacket) error {
	switch p := pkt.(type) {
	case *packets.ConnectPacket:
		s.client = Client{
			ID:       p.ClientIdentifier,
			Username: p.Username,
			Password: p.Password,
		}
		if err := s.event.AuthConnect(&s.client); err != nil {
			return err
		}
		// Copy back to the packet in case values are changed by Event handler.
		// This is specific to CONN, as only that package type has credentials.
		p.ClientIdentifier = s.client.ID
		p.Username = s.client.Username
		p.Password = s.client.Password
		return nil
	case *packets.PublishPacket:
		return s.event.AuthPublish(&s.client, &p.TopicName, &p.Payload)
	case *packets.SubscribePacket:
		return s.event.AuthSubscribe(&s.client, &p.Topics)
	default:
		return nil
	}
}

func (s *session) notify(pkt packets.ControlPacket) {
	switch p := pkt.(type) {
	case *packets.ConnectPacket:
		s.event.Connect(&s.client)
	case *packets.PublishPacket:
		s.event.Publish(&s.client, &p.TopicName, &p.Payload)
	case *packets.SubscribePacket:
		s.event.Subscribe(&s.client, &p.Topics)
	case *packets.UnsubscribePacket:
		s.event.Unsubscribe(&s.client, &p.Topics)
	default:
		return
	}
}
