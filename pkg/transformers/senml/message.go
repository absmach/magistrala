package senml

// Message represents a resolved (normalized) SenML record.
type Message struct {
	Channel     string   `json:"channel,omitempty"`
	Subtopic    string   `json:"subtopic,omitempty"`
	Publisher   string   `json:"publisher,omitempty"`
	Protocol    string   `json:"protocol,omitempty"`
	Name        string   `json:"name,omitempty"`
	Unit        string   `json:"unit,omitempty"`
	Time        float64  `json:"time,omitempty"`
	UpdateTime  float64  `json:"update_time,omitempty"`
	Value       *float64 `json:"value,omitempty"`
	StringValue *string  `json:"string_value,omitempty"`
	DataValue   *string  `json:"data_value,omitempty"`
	BoolValue   *bool    `json:"bool_value,omitempty"`
	Sum         *float64 `json:"sum,omitempty"`
}
