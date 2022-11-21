package mocks

import "github.com/mainflux/mainflux/pkg/messaging"

type MockPublisher struct{}

// NewPublisher returns mock message publisher.
func NewPublisher() messaging.Publisher {
	return MockPublisher{}
}

func (pub MockPublisher) Publish(topic string, msg messaging.Message) error {
	return nil
}

func (pub MockPublisher) Close() error {
	return nil
}
