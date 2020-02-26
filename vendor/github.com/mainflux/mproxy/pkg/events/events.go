package events

// Event is an interface for mProxy hooks
type Event interface {
	// Authorization on client `CONNECT`
	// Each of the params are passed by reference, so that it can be changed
	AuthRegister(username, clientID *string, password *[]byte) error

	// Authorization on client `PUBLISH`
	// Topic is passed by reference, so that it can be modified
	AuthPublish(username, clientID string, topic *string, payload *[]byte) error

	// Authorization on client `SUBSCRIBE`
	// Topics are passed by reference, so that they can be modified
	AuthSubscribe(username, clientID string, topics *[]string) error

	// After client successfully connected
	Register(clientID string)

	// After client successfully published
	Publish(clientID, topic string, payload []byte)

	// After client successfully subscribed
	Subscribe(clientID string, topics []string)

	// After client unsubscribed
	Unsubscribe(clientID string, topics []string)
}
