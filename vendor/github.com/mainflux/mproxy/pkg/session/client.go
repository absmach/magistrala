package session

// Client stores MQTT client data.
type Client struct {
	ID       string
	Username string
	Password []byte
}
