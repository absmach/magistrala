package session

// Handler is an interface for mProxy hooks
type Handler interface {
	// Authorization on client `CONNECT`
	// Each of the params are passed by reference, so that it can be changed
	AuthConnect(client *Client) error

	// Authorization on client `PUBLISH`
	// Topic is passed by reference, so that it can be modified
	AuthPublish(client *Client, topic *string, payload *[]byte) error

	// Authorization on client `SUBSCRIBE`
	// Topics are passed by reference, so that they can be modified
	AuthSubscribe(client *Client, topics *[]string) error

	// After client successfully connected
	Connect(client *Client)

	// After client successfully published
	Publish(client *Client, topic *string, payload *[]byte)

	// After client successfully subscribed
	Subscribe(client *Client, topics *[]string)

	// After client unsubscribed
	Unsubscribe(client *Client, topics *[]string)

	// Disconnect on connection with client lost
	Disconnect(client *Client)
}
