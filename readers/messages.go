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
	// DeviceScope is the authorization boundary, as opposed to the convenience
	// filters above. It is set by the reader API from what the caller is
	// entitled to, never from the request.
	DeviceScope *DeviceScope `json:"device_scope,omitempty"`
}

// DeviceScope restricts results to rows attributable to a known set of devices.
//
// A device reaches a row through either of two columns, and which one depends on
// how the data arrived: a device publishing for itself is identified by
// `publisher`, while a device whose readings a gateway relays is identified by
// `device_id` — the gateway owns the publisher identity in that case. The two
// are therefore combined with OR: a row is in scope if either column names an
// in-scope device. Combining them with AND would make gateway-relayed data
// unreachable, which is the case this exists to serve.
//
// Both slices project the same set of devices — PublisherIDs holds their
// platform UUIDs, DeviceIDs the external serials denormalised onto message rows
// — so narrowing the device set must narrow both together, never one alone.
//
// Unlike the convenience filters, EMPTY MEANS "MATCH NOTHING" HERE. The field is
// a pointer so `omitempty` keeps a non-nil empty scope in the query, where it
// becomes `= ANY('{}')` and excludes every row. A caller entitled to no devices
// must still end up with zero rows if it ever reaches the query, whatever the
// API layer does before that.
type DeviceScope struct {
	PublisherIDs []string `json:"publisher_ids"`
	DeviceIDs    []string `json:"device_ids"`
}

// Publishers returns the in-scope publisher UUIDs, never nil, so a scope that
// authorizes nothing binds an empty array rather than NULL.
func (s *DeviceScope) Publishers() []string {
	if s == nil || s.PublisherIDs == nil {
		return []string{}
	}
	return s.PublisherIDs
}

// Devices returns the in-scope device serials, never nil, so a scope that
// authorizes nothing binds an empty array rather than NULL.
func (s *DeviceScope) Devices() []string {
	if s == nil || s.DeviceIDs == nil {
		return []string{}
	}
	return s.DeviceIDs
}

// Empty reports whether the scope admits no devices at all.
func (s *DeviceScope) Empty() bool {
	return s != nil && len(s.PublisherIDs) == 0 && len(s.DeviceIDs) == 0
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
