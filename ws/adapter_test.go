package ws_test

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/ws"
	"github.com/mainflux/mainflux/ws/mocks"
	broker "github.com/nats-io/go-nats"
	"github.com/stretchr/testify/assert"

	"github.com/mainflux/mainflux"
)

const (
	chanID   = "123e4567-e89b-12d3-a456-000000000001"
	pubID    = "1"
	protocol = "ws"
)

var (
	msg = mainflux.RawMessage{
		Channel:   chanID,
		Publisher: pubID,
		Protocol:  protocol,
		Payload:   []byte(`[{"n":"current","t":-5,"v":1.2}]`),
	}
	channel = ws.Channel{make(chan mainflux.RawMessage), make(chan bool)}
)

func newService() ws.Service {
	subs := map[string]ws.Channel{chanID: channel}
	pubsub := mocks.NewService(subs, broker.ErrInvalidMsg)
	return ws.New(pubsub)
}

func TestPublish(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc string
		msg  mainflux.RawMessage
		err  error
	}{
		{"publish valid message", msg, nil},
		{"publish empty message", mainflux.RawMessage{}, ws.ErrFailedMessagePublish},
	}

	for _, tc := range cases {
		// Check if message was sent.
		go func(desc string, tcMsg mainflux.RawMessage) {
			msg := <-channel.Messages
			assert.Equal(t, tcMsg, msg, fmt.Sprintf("%s: expected %s got %s\n", desc, tcMsg, msg))
		}(tc.desc, tc.msg)

		// Check if publish succeeded.
		err := svc.Publish(tc.msg)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestSubscribe(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc    string
		chanID  string
		channel ws.Channel
		err     error
	}{
		{"subscription to valid channel", chanID, channel, nil},
		{"subscription to channel that should fail", "non-existent-chan-id", channel, ws.ErrFailedSubscription},
	}

	for _, tc := range cases {
		err := svc.Subscribe(tc.chanID, tc.channel)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
