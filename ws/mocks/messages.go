package mocks

import (
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/ws"
)

var _ ws.Service = (*mockService)(nil)

type mockService struct {
	subscriptions map[uint64]ws.Channel
	pubError      error
}

// NewService returns mock message publisher.
func NewService(subs map[uint64]ws.Channel, pubError error) ws.Service {
	return mockService{subs, pubError}
}

func (svc mockService) Publish(msg mainflux.RawMessage) error {
	if len(msg.Payload) == 0 {
		return svc.pubError
	}
	svc.subscriptions[msg.Channel].Messages <- msg
	return nil
}

func (svc mockService) Subscribe(chanID uint64, channel ws.Channel) error {
	if _, ok := svc.subscriptions[chanID]; !ok {
		return ws.ErrFailedSubscription
	}
	svc.subscriptions[chanID] = channel
	return nil
}
