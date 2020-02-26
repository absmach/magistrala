package mqtt

import (
	"net"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mproxy/pkg/events"
)

const (
	up direction = iota
	down
)

type direction int

type mqttClient struct {
	ID       string
	username string
	password []byte
}

type session struct {
	id       string
	logger   logger.Logger
	inbound  net.Conn
	outbound net.Conn
	event    events.Event
	client   mqttClient
}

func newSession(uuid string, inbound, outbound net.Conn, event events.Event, logger logger.Logger) *session {
	return &session{
		id:       uuid,
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

	return <-errs
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
		if err := s.event.AuthRegister(&p.Username, &p.ClientIdentifier, &p.Password); err != nil {
			return err
		}
		s.client.username = p.Username
		s.client.password = p.Password
		s.client.ID = p.ClientIdentifier
		return nil
	case *packets.PublishPacket:
		return s.event.AuthPublish(s.client.username, s.client.ID, &p.TopicName, &p.Payload)
	case *packets.SubscribePacket:
		return s.event.AuthSubscribe(s.client.username, s.client.ID, &p.Topics)
	default:
		return nil
	}
}

func (s *session) notify(pkt packets.ControlPacket) {
	switch p := pkt.(type) {
	case *packets.ConnectPacket:
		s.event.Register(s.client.ID)
	case *packets.PublishPacket:
		s.event.Publish(s.client.ID, p.TopicName, p.Payload)
	case *packets.SubscribePacket:
		s.event.Subscribe(s.client.ID, p.Topics)
	case *packets.UnsubscribePacket:
		s.event.Unsubscribe(s.client.ID, p.Topics)
	default:
		return
	}
}
