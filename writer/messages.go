// Package writer provides message writer concept definitions.
package writer

// Message represents a resolved (normalized) raw message.
type Message struct {
	Channel     string
	Publisher   string
	Protocol    string
	Version     int     `json:"bver,omitempty"`
	Name        string  `json:"n,omitempty"`
	Unit        string  `json:"u,omitempty"`
	Value       float64 `json:"v,omitempty"`
	StringValue string  `json:"vs,omitempty"`
	BoolValue   bool    `json:"vb,omitempty"`
	DataValue   string  `json:"vd,omitempty"`
	ValueSum    float64 `json:"s,omitempty"`
	Time        float64 `json:"t,omitempty"`
	UpdateTime  float64 `json:"ut,omitempty"`
	Link        string  `json:"l,omitempty"`
}

// RawMessage represents a message emitted by the mainflux adapters layer.
type RawMessage struct {
	Channel     string `json:"channel"`
	Publisher   string `json:"publisher"`
	Protocol    string `json:"protocol"`
	ContentType string `json:"content_type"`
	Payload     []byte `json:"payload"`
}

// MessageRepository specifies a message persistence API.
type MessageRepository interface {
	// Save persists the message. A non-nil error is returned to indicate
	// operation failure.
	Save(RawMessage) error
}
