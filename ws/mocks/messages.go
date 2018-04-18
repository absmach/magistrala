package mocks

import (
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/ws"
	broker "github.com/nats-io/go-nats"
)

var _ ws.Service = (*mockService)(nil)

type mockService struct {
	subscriptions map[string]ws.Channel
}

// NewService returns mock message publisher.
func NewService(subs map[string]ws.Channel) ws.Service {
	return mockService{subs}
}

func (svc mockService) Publish(msg mainflux.RawMessage) error {
	if len(msg.Payload) == 0 {
		return broker.ErrInvalidMsg
	}
	svc.subscriptions[msg.Channel].Messages <- msg
	return nil
}

func (svc mockService) Subscribe(chanID string, channel ws.Channel) error {
	if _, ok := svc.subscriptions[chanID]; !ok {
		return ws.ErrFailedSubscription
	}
	svc.subscriptions[chanID] = channel
	return nil
}
