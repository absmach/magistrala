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

	// ListGatewayDevices returns the distinct device_id values observed on
	// channel chanID among messages published by publisherID — a gateway's
	// client id — each with the last time it was seen and how many messages
	// it produced (MG-15). This is "devices seen coming through this
	// gateway"; it says nothing about what was commissioned onto the
	// gateway, only what has actually published.
	//
	// Only pm's Offset, Limit, From, To and DeviceScope fields are honoured;
	// the rest of PageMetadata's filters do not apply to this aggregation.
	// DeviceScope, when set, narrows the returned device_id values to the
	// caller's authorized device set — a single gateway can relay for
	// devices belonging to more than one authorized caller, so the roster
	// has to be narrowed row by row, not just gated at the request level.
	ListGatewayDevices(chanID, publisherID string, pm PageMetadata) (DeviceStatsPage, error)

	// ListDeviceGateways returns the distinct publisher values observed on
	// channel chanID among messages carrying device_id deviceID — the
	// gateways that have relayed for this device — each with the last time
	// it was seen and how many messages it produced (MG-15). A device can be
	// relayed by more than one gateway, so this is a roster, not a lookup.
	//
	// Only pm's Offset, Limit, From and To fields are honoured. DeviceScope
	// narrowing does not apply on this side: deviceID is itself the
	// authorization boundary (checked before this is ever called), so every
	// row this can return already belongs to that one authorized device,
	// whichever gateway relayed it. Narrowing the publisher column against
	// the caller's own publisher grant would additionally require the
	// caller to be separately authorized for the relaying gateway's
	// identity, which is not how device authorization works here and would
	// wrongly empty out exactly the gateway-relayed case this exists to
	// serve.
	ListDeviceGateways(chanID, deviceID string, pm PageMetadata) (DeviceStatsPage, error)
}

// DeviceStat is one row of the observed-device aggregation (MG-15): a
// distinct identity — a device serial for the gateway-to-devices direction, a
// gateway's publisher id for the inverse — observed on a channel within a
// time range.
//
// LastSeen carries the raw numeric value of MAX(time) over the matching
// rows, the same representation senml.Message.Time and PageMetadata.From/To
// already use throughout this package, rather than a converted wall-clock
// timestamp. Postgres and Timescale do not agree end to end on the unit
// "time" is stored in (compare this package's own From/To handling with
// timescale's time_bucket arithmetic), so a conversion here would have to
// commit to a unit per backend and risks silently picking the wrong one.
// Passing the column value straight through keeps this consistent with
// every other time value the package already exposes.
type DeviceStat struct {
	ID           string
	LastSeen     float64
	MessageCount uint64
}

// DeviceStatsPage is a page of DeviceStat rows plus paging metadata,
// mirroring MessagesPage.
type DeviceStatsPage struct {
	PageMetadata
	Total uint64
	Stats []DeviceStat
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
// All three slices project the same held-device set. PublisherIDs and DeviceIDs
// hold their platform UUIDs and external serials respectively, so narrowing the
// device set must narrow both together, never one alone. SelfPublisherIDs is the
// subset of PublisherIDs not flagged as a gateway (attributes.is_gateway, spec
// §8 A12): for those devices, a row naming one as publisher is trusted as its
// own self-published data regardless of what device_id holds, since an ordinary
// device's own SenML base name (or an equivalent JSON key) can legitimately
// populate device_id without the row being relayed for anyone else (R1) — a row
// carrying only one of the two identities, not both meaningfully at once, is an
// assumption device_id alone cannot be trusted to signal. A gateway-flagged
// publisher does not get this: its relayed rows carry a real device_id for a
// different device, and matching it on publisher alone would readmit every
// device it has ever relayed for, which is exactly the leak device_id = ”
// exists to close (see readers/postgres and readers/timescale's fmtCondition).
//
// Unlike the convenience filters, EMPTY MEANS "MATCH NOTHING" HERE. The field is
// a pointer so `omitempty` keeps a non-nil empty scope in the query, where it
// becomes `= ANY('{}')` and excludes every row. A caller entitled to no devices
// must still end up with zero rows if it ever reaches the query, whatever the
// API layer does before that.
type DeviceScope struct {
	PublisherIDs     []string `json:"publisher_ids"`
	SelfPublisherIDs []string `json:"self_publisher_ids"`
	DeviceIDs        []string `json:"device_ids"`
}

// Publishers returns the in-scope publisher UUIDs, never nil, so a scope that
// authorizes nothing binds an empty array rather than NULL.
func (s *DeviceScope) Publishers() []string {
	if s == nil || s.PublisherIDs == nil {
		return []string{}
	}
	return s.PublisherIDs
}

// SelfPublishers returns the in-scope publisher UUIDs not flagged as a
// gateway, never nil, so a scope that authorizes nothing binds an empty array
// rather than NULL.
func (s *DeviceScope) SelfPublishers() []string {
	if s == nil || s.SelfPublisherIDs == nil {
		return []string{}
	}
	return s.SelfPublisherIDs
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
