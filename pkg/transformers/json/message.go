// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package json

// Payload represents JSON Message payload.
type Payload map[string]any

// Message represents a JSON messages.
type Message struct {
	Channel   string  `json:"channel,omitempty" db:"channel" bson:"channel"`
	Created   int64   `json:"created,omitempty" db:"created" bson:"created"`
	Subtopic  string  `json:"subtopic,omitempty" db:"subtopic" bson:"subtopic,omitempty"`
	Publisher string  `json:"publisher,omitempty" db:"publisher" bson:"publisher"`
	Protocol  string  `json:"protocol,omitempty" db:"protocol" bson:"protocol"`
	Payload   Payload `json:"payload,omitempty" db:"payload" bson:"payload,omitempty"`
	DeviceId  string  `json:"device_id,omitempty" db:"device_id" bson:"device_id,omitempty"`
}

// Messages represents a list of JSON messages.
type Messages struct {
	Data   []Message
	Format string
}
