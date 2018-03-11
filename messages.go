package mainflux

// Message represents a resolved (normalized) raw message.
type Message struct {
	Channel     string
	Publisher   string
	Protocol    string
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

// MessagePublisher specifies a message publishing API.
type MessagePublisher interface {
	// Publishes message to the stream. A non-nil error is returned to indicate
	// operation failure.
	Publish(RawMessage) error
}
