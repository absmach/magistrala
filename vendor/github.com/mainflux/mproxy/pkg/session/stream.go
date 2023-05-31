package session

import (
	"context"
	"crypto/x509"
	"fmt"
	"io"
	"net"

	"github.com/eclipse/paho.mqtt.golang/packets"
)

type direction int

const (
	up direction = iota
	down
)

const unknownID = "unknown"

var (
	errBroker = "failed to proxy from MQTT client with id %s to MQTT broker with error: %s"
	errClient = "failed to proxy from MQTT broker to client with id %s with error: %s"
)

// Stream starts proxy between client and broker.
func Stream(ctx context.Context, inbound, outbound net.Conn, handler Handler, cert x509.Certificate) error {
	s := Session{
		Cert: cert,
	}
	ctx = NewContext(ctx, &s)
	errs := make(chan error, 2)

	go stream(ctx, up, inbound, outbound, handler, errs)
	go stream(ctx, down, outbound, inbound, handler, errs)

	// Handle whichever error happens first.
	// The other routine won't be blocked when writing
	// to the errors channel because it is buffered.
	err := <-errs

	handler.Disconnect(ctx)
	return err
}

func stream(ctx context.Context, dir direction, r, w net.Conn, h Handler, errs chan error) {
	for {
		// Read from one connection.
		pkt, err := packets.ReadPacket(r)
		if err != nil {
			errs <- wrap(ctx, err, dir)
			return
		}

		if dir == up {
			if err = authorize(ctx, pkt, h); err != nil {
				errs <- wrap(ctx, err, dir)
				return
			}
		}

		// Send to another.
		if err := pkt.Write(w); err != nil {
			errs <- wrap(ctx, err, dir)
			return
		}

		if dir == up {
			notify(ctx, pkt, h)
		}
	}
}

func authorize(ctx context.Context, pkt packets.ControlPacket, h Handler) error {
	switch p := pkt.(type) {
	case *packets.ConnectPacket:
		s, ok := FromContext(ctx)
		if ok {
			s.ID = p.ClientIdentifier
			s.Username = p.Username
			s.Password = p.Password
		}

		ctx = NewContext(ctx, s)
		if err := h.AuthConnect(ctx); err != nil {
			return err
		}
		// Copy back to the packet in case values are changed by Event handler.
		// This is specific to CONN, as only that package type has credentials.
		p.ClientIdentifier = s.ID
		p.Username = s.Username
		p.Password = s.Password
		return nil
	case *packets.PublishPacket:
		return h.AuthPublish(ctx, &p.TopicName, &p.Payload)
	case *packets.SubscribePacket:
		return h.AuthSubscribe(ctx, &p.Topics)
	default:
		return nil
	}
}

func notify(ctx context.Context, pkt packets.ControlPacket, h Handler) {
	switch p := pkt.(type) {
	case *packets.ConnectPacket:
		h.Connect(ctx)
	case *packets.PublishPacket:
		h.Publish(ctx, &p.TopicName, &p.Payload)
	case *packets.SubscribePacket:
		h.Subscribe(ctx, &p.Topics)
	case *packets.UnsubscribePacket:
		h.Unsubscribe(ctx, &p.Topics)
	default:
		return
	}
}

func wrap(ctx context.Context, err error, dir direction) error {
	if err == io.EOF {
		return err
	}
	cid := unknownID
	if s, ok := FromContext(ctx); ok {
		cid = s.ID
	}
	switch dir {
	case up:
		return fmt.Errorf(errClient, cid, err.Error())
	case down:
		return fmt.Errorf(errBroker, cid, err.Error())
	default:
		return err
	}
}
