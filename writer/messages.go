// Package writer provides message writer concept definitions.
package writer

// Message represents a message emitted by the mainflux adapters layer.
type Message struct {
	Channel     string
	Publisher   string
	Protocol    string
	BaseName    string  `json:"bn,omitempty"`
	BaseTime    float64 `json:"bt,omitempty"`
	BaseUnit    string  `json:"bu,omitempty"`
	BaseValue   float64 `json:"bv,omitempty"`
	BaseSum     float64 `json:"bs,omitempty"`
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

// MessageRepository specifies a message persistence API.
type MessageRepository interface {
	// Save persists the message. A non-nil error is returned to indicate
	// operation failure.
	Save(Message) error
}
