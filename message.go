package mainflux

import "encoding/json"

const (
	// SenMLJSON represents SenML in JSON format content type.
	SenMLJSON = "application/senml+json"

	// SenMLCBOR represents SenML in CBOR format content type.
	SenMLCBOR = "application/senml+cbor"
)

// Type messageType is introduced to prevent cycle when calling Message
// MarshalJSON and UnmarshalJSON methods.
type messageType Message

// Struct message is an internal representation of Mainflux message to be serialized to JSON.
// Field `Value` is added to prevent marshaling of corresponding Message field.
type message struct {
	messageType
	Value       isMessage_Value `json:"Value,omitempty"`
	FloatValue  *float64        `json:"value,omitempty"`
	StringValue *string         `json:"stringValue,omitempty"`
	BoolValue   *bool           `json:"boolValue,omitempty"`
	DataValue   *string         `json:"dataValue,omitempty"`
	ValueSum    *float64        `json:"valueSum,omitempty"`
}

// MarshalJSON method is used by `json` package to serialize Message.
func (m Message) MarshalJSON() ([]byte, error) {
	msg := message{messageType: messageType(m)}

	switch m.Value.(type) {
	case *Message_FloatValue:
		floatVal := m.GetFloatValue()
		msg.FloatValue = &floatVal
	case *Message_StringValue:
		strVal := m.GetStringValue()
		msg.StringValue = &strVal
	case *Message_DataValue:
		dataVal := m.GetDataValue()
		msg.DataValue = &dataVal
	case *Message_BoolValue:
		boolVal := m.GetBoolValue()
		msg.BoolValue = &boolVal
	}

	if m.GetValueSum() != nil {
		valueSum := m.GetValueSum().GetValue()
		msg.ValueSum = &valueSum
	}

	return json.Marshal(msg)
}

// UnmarshalJSON method is used by `json` package to unmarshal data to Message.
func (m *Message) UnmarshalJSON(data []byte) error {
	var msg message
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}

	*m = Message(msg.messageType)
	m.Value = nil

	switch {
	case msg.FloatValue != nil:
		m.Value = &Message_FloatValue{*msg.FloatValue}
	case msg.StringValue != nil:
		m.Value = &Message_StringValue{*msg.StringValue}
	case msg.DataValue != nil:
		m.Value = &Message_DataValue{*msg.DataValue}
	case msg.BoolValue != nil:
		m.Value = &Message_BoolValue{*msg.BoolValue}
	}

	if msg.ValueSum != nil {
		m.ValueSum = &SumValue{Value: *msg.ValueSum}
	}

	return nil
}
