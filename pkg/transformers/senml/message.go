// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package senml

// Message represents a resolved (normalized) SenML record.
type Message struct {
	Channel     string   `json:"channel,omitempty" db:"channel" bson:"channel"`
	Subtopic    string   `json:"subtopic,omitempty" db:"subtopic" bson:"subtopic,omitempty"`
	Publisher   string   `json:"publisher,omitempty" db:"publisher" bson:"publisher"`
	Protocol    string   `json:"protocol,omitempty" db:"protocol" bson:"protocol"`
	Name        string   `json:"name,omitempty" db:"name" bson:"name,omitempty"`
	Unit        string   `json:"unit,omitempty" db:"unit" bson:"unit,omitempty"`
	Time        float64  `json:"time,omitempty" db:"time" bson:"time,omitempty"`
	UpdateTime  float64  `json:"update_time,omitempty" db:"update_time" bson:"update_time,omitempty"`
	Value       *float64 `json:"value,omitempty" db:"value" bson:"value,omitempty"`
	StringValue *string  `json:"string_value,omitempty" db:"string_value" bson:"string_value,omitempty"`
	DataValue   *string  `json:"data_value,omitempty" db:"data_value" bson:"data_value,omitempty"`
	BoolValue   *bool    `json:"bool_value,omitempty" db:"bool_value" bson:"bool_value,omitempty"`
	Sum         *float64 `json:"sum,omitempty" db:"sum" bson:"sum,omitempty"`
}
