package gosenml

import (
	"encoding/json"
)

// JSONEncoder encodes SenML messages to JSON
type JSONEncoder struct{}

// NewJSONEncoder returns a new JSONEncoder
func NewJSONEncoder() *JSONEncoder {
	return &JSONEncoder{}
}

// EncodeMessage encodes a SenML message to JSON
func (je *JSONEncoder) EncodeMessage(m *Message) ([]byte, error) {
	if err := m.Validate(); err != nil {
		return []byte{}, err
	}

	b, err := json.Marshal(m)
	if err != nil {
		return []byte{}, err
	}
	return b, nil
}

// EncodeEntry encodes a SenML entry to JSON
func (je *JSONEncoder) EncodeEntry(e *Entry) ([]byte, error) {
	b, err := json.Marshal(e)
	if err != nil {
		return []byte{}, err
	}
	return b, nil
}

// JSONDecoder decodes SenML messages from JSON
type JSONDecoder struct{}

// NewJSONDecoder returns a new JSONDecoder
func NewJSONDecoder() *JSONDecoder {
	return &JSONDecoder{}
}

// DecodeMessage decodes a SenML messages from JSON
func (jd *JSONDecoder) DecodeMessage(data []byte) (Message, error) {
	m := Message{}
	err := json.Unmarshal(data, &m)
	if err != nil {
		return Message{}, err
	}

	if err = m.Validate(); err != nil {
		return m, err
	}
	return m, nil
}

// DecodeEntry decodes a SenML entry from JSON
func (jd *JSONDecoder) DecodeEntry(data []byte) (Entry, error) {
	e := Entry{}
	err := json.Unmarshal(data, &e)
	if err != nil {
		return Entry{}, err
	}
	return e, nil
}
