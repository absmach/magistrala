package mocks

import (
	"github.com/mainflux/mainflux/mqtt/redis"
)

type MockEventStore struct{}

func NewEventStore() redis.EventStore {
	return MockEventStore{}
}

func (es MockEventStore) Connect(clientID string) error {
	return nil
}

func (es MockEventStore) Disconnect(clientID string) error {
	return nil
}
