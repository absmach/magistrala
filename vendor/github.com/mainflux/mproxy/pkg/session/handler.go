package session

import "context"

// Handler is an interface for mProxy hooks
type Handler interface {
	// Authorization on client `CONNECT`
	// Each of the params are passed by reference, so that it can be changed
	AuthConnect(ctx context.Context) error

	// Authorization on client `PUBLISH`
	// Topic is passed by reference, so that it can be modified
	AuthPublish(ctx context.Context, topic *string, payload *[]byte) error

	// Authorization on client `SUBSCRIBE`
	// Topics are passed by reference, so that they can be modified
	AuthSubscribe(ctx context.Context, topics *[]string) error

	// After client successfully connected
	Connect(ctx context.Context) error

	// After client successfully published
	Publish(ctx context.Context, topic *string, payload *[]byte) error

	// After client successfully subscribed
	Subscribe(ctx context.Context, topics *[]string) error

	// After client unsubscribed
	Unsubscribe(ctx context.Context, topics *[]string) error

	// Disconnect on connection with client lost
	Disconnect(ctx context.Context) error
}
