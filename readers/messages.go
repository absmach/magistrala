// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package readers

import "errors"

const (
	// EqualKey represents the equal comparison operator key.
	EqualKey = "eq"
	// LowerThanKey represents the lower-than comparison operator key.
	LowerThanKey = "lt"
	// LowerThanEqualKey represents the lower-than-or-equal comparison operator key.
	LowerThanEqualKey = "le"
	// GreaterThanKey represents the greater-than-or-equal comparison operator key.
	GreaterThanKey = "gt"
	// GreaterThanEqualKey represents the greater-than-or-equal comparison operator key.
	GreaterThanEqualKey = "ge"
)

// ErrReadMessages indicates failure occurred while reading messages from database.
var ErrReadMessages = errors.New("failed to read messages from database")

// MessageRepository specifies message reader API.
type MessageRepository interface {
	// ReadAll skips given number of messages for given channel and returns next
	// limited number of messages.
	ReadAll(chanID string, pm PageMetadata) (MessagesPage, error)
}

// Message represents any message format.
type Message any

// MessagesPage contains page related metadata as well as list of messages that
// belong to this page.
type MessagesPage struct {
	PageMetadata
	Total    uint64
	Messages []Message
}

// PageMetadata represents the parameters used to create database queries.
type PageMetadata struct {
	Offset     uint64   `json:"offset"`
	Limit      uint64   `json:"limit"`
	Order      string   `json:"order,omitempty"`
	Dir        string   `json:"dir,omitempty"`
	Subtopic   string   `json:"subtopic,omitempty"`
	Publisher  string   `json:"publisher,omitempty"`
	Publishers []string `json:"publishers,omitempty"`
	// DeviceIDs filters on the device serial denormalised onto every row by the
	// writers. Values are matched byte-for-byte; they are external serials, not
	// platform UUIDs, so a caller holding Atom entity IDs must translate them to
	// external_id first (MG-08).
	//
	// An EMPTY OR NIL SLICE MEANS "NO DEVICE FILTER", NOT "MATCH NOTHING".
	// The WHERE builders marshal this struct to JSON and iterate the resulting
	// map, so `omitempty` erases an empty slice exactly as it erases a nil one
	// and no condition is emitted. The distinction cannot be restored with a
	// pointer either: proto3 `repeated` has no field presence, so an empty and
	// an absent list arrive identically over gRPC.
	//
	// Authorization must therefore never express "authorized for zero devices"
	// by assigning an empty slice here. It must short-circuit before calling
	// ReadAll and return an empty page — the pattern publisherAuthorizer already
	// uses for publishers via its noAccess result.
	DeviceIDs   []string `json:"device_ids,omitempty"`
	Protocol    string   `json:"protocol,omitempty"`
	Name        string   `json:"name,omitempty"`
	Value       float64  `json:"v,omitempty"`
	Comparator  string   `json:"comparator,omitempty"`
	BoolValue   bool     `json:"vb,omitempty"`
	StringValue string   `json:"vs,omitempty"`
	DataValue   string   `json:"vd,omitempty"`
	From        float64  `json:"from,omitempty"`
	To          float64  `json:"to,omitempty"`
	Format      string   `json:"format,omitempty"`
	Aggregation string   `json:"aggregation,omitempty"`
	Interval    string   `json:"interval,omitempty"`
}

// ParseValueComparator convert comparison operator keys into mathematic anotation.
func ParseValueComparator(query map[string]any) string {
	comparator := "="
	val, ok := query["comparator"]
	if ok {
		switch val.(string) {
		case EqualKey:
			comparator = "="
		case LowerThanKey:
			comparator = "<"
		case LowerThanEqualKey:
			comparator = "<="
		case GreaterThanKey:
			comparator = ">"
		case GreaterThanEqualKey:
			comparator = ">="
		}
	}

	return comparator
}
