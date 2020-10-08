package session

import "crypto/x509"

// Client stores MQTT client data.
type Client struct {
	ID       string
	Username string
	Password []byte
	Cert     x509.Certificate
}
