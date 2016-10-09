package gosenml

import "fmt"

// Data model as described in
// https://tools.ietf.org/html/draft-jennings-senml-10

// Message is the root SenML variable
type Message struct {
	BaseName  string  `json:"bn,omitempty"`
	BaseTime  int64   `json:"bt,omitempty"`
	BaseUnits string  `json:"bu,omitempty"`
	Version   int     `json:"ver"`
	Entries   []Entry `json:"e"`
}

// Entry is a measurement of Parameter Entry
type Entry struct {
	Name         string   `json:"n,omitempty"`
	Units        string   `json:"u,omitempty"`
	Value        *float64 `json:"v,omitempty"`
	StringValue  *string  `json:"sv,omitempty"`
	BooleanValue *bool    `json:"bv,omitempty"`
	Sum          *float64 `json:"s,omitempty"`
	Time         int64    `json:"t,omitempty"`
	UpdateTime   int64    `json:"ut,omitempty"`
}

// NewMessage creates a SenML message from a number of entries
func NewMessage(entries ...Entry) *Message {
	return &Message{
		Version: 1.0,
		Entries: entries,
	}
}

// Makes a deep copy of the message
func (m *Message) copy() Message {
	mc := *m
	entries := make([]Entry, len(m.Entries))
	copy(entries, m.Entries)
	mc.Entries = entries
	return mc
}

// Validate validates a message
func (m *Message) Validate() error {
	if len(m.Entries) == 0 {
		return fmt.Errorf("Invalid Message: entries must be non-empty")
	}

	// Validate values in entries
	// https://tools.ietf.org/html/draft-jennings-senml-10#section-4
	for _, e := range m.Entries {
		vars := 0
		if e.Value != nil {
			vars++
		}
		if e.StringValue != nil {
			vars++
		}
		if e.BooleanValue != nil {
			vars++
		}
		if e.Sum == nil && vars != 1 {
			return fmt.Errorf("In an entry, exactly one of v, sv, or bv MUST appear when a sum value is not present")
		} else if e.Sum != nil && vars > 1 {
			return fmt.Errorf("In an entry, exactly one of v, sv, or bv CAN appear when a sum value is present")
		}
	}
	// TODO: more validation
	return nil
}

// Expand returns a copy of the message with all Entries expanded ("self-contained")
func (m *Message) Expand() Message {
	m2 := m.copy()

	for i, e := range m.Entries {
		// BaseName
		e.Name = m.BaseName + e.Name

		// BaseTime
		e.Time = m.BaseTime + e.Time

		// BaseUnits
		if e.Units == "" {
			e.Units = m.BaseUnits
		}
		m2.Entries[i] = e
	}
	m2.BaseName = ""
	m2.BaseTime = 0
	m2.BaseUnits = ""
	return m2
}

// Compact returns a copy of the message with all Entries compacted (common data put into Message)
func (m *Message) Compact() Message {
	m2 := m.copy()
	// TODO
	// BaseName
	// BaseTime
	// BaseUnits
	return m2
}

// Encoder interface
type Encoder interface {
	EncodeMessage(*Message) ([]byte, error)
	EncodeEntry(*Entry) ([]byte, error)
}

// Decoder interface
type Decoder interface {
	DecodeEntry([]byte) (Entry, error)
	DecodeMessage([]byte) (Message, error)
}
