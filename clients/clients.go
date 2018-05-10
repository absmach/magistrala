package clients

import "strings"

// Client represents a Mainflux client. Each client is owned by one user, and
// it is assigned with the unique identifier and (temporary) access key.
type Client struct {
	ID      string `json:"id"`
	Owner   string `json:"-"`
	Type    string `json:"type"`
	Name    string `json:"name,omitempty"`
	Key     string `json:"key"`
	Payload string `json:"payload,omitempty"`
}

var clientTypes = map[string]bool{
	"app":    true,
	"device": true,
}

// Validate returns an error if client representation is invalid.
func (c *Client) Validate() error {
	if c.Type = strings.ToLower(c.Type); !clientTypes[c.Type] {
		return ErrMalformedEntity
	}

	return nil
}

// ClientRepository specifies a client persistence API.
type ClientRepository interface {
	// ID generates new resource identifier.
	ID() string

	// Save persists the client. Successful operation is indicated by non-nil
	// error response.
	Save(Client) error

	// Update performs an update to the existing client. A non-nil error is
	// returned to indicate operation failure.
	Update(Client) error

	// One retrieves the client having the provided identifier, that is owned
	// by the specified user.
	One(string, string) (Client, error)

	// All retrieves the subset of clients owned by the specified user.
	All(string, int, int) []Client

	// Remove removes the client having the provided identifier, that is owned
	// by the specified user.
	Remove(string, string) error
}
