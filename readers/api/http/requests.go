// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"regexp"
	"slices"
	"strings"
	"time"

	api "github.com/absmach/magistrala/api/http"
	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/readers"
)

const maxLimitSize = 1000

var validAggregations = []string{"MAX", "MIN", "AVG", "SUM", "COUNT"}

// validFormat matches a bare, unquoted Postgres identifier -- the only shape
// format may safely take. format ultimately gets interpolated into a FROM
// clause (readers/postgres and readers/timescale's messages.go) rather than
// bound as a query parameter, because it names a table, not a value; per-
// device authorization there is enforced entirely as WHERE predicates against
// whatever table format names (MG-08), so anything that lets a caller inject
// SQL through it — a comment sequence, a UNION, even just whitespace —
// bypasses that authorization wholesale rather than merely misnaming a table.
// 63 bytes matches Postgres's own NAMEDATALEN identifier limit.
var validFormat = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]{0,62}$`)

type listMessagesReq struct {
	chanID    string
	token     string
	workspace string
	key       string
	pageMeta  readers.PageMetadata
}

func (req listMessagesReq) validate() error {
	if req.token == "" && req.key == "" {
		return apiutil.ErrBearerToken
	}

	if req.chanID == "" {
		return apiutil.ErrMissingID
	}

	if req.pageMeta.Limit < 1 || req.pageMeta.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	if req.pageMeta.Format != "" && !validFormat.MatchString(req.pageMeta.Format) {
		return apiutil.ErrInvalidFormat
	}

	if !readers.ValidTimeRange(req.pageMeta.From) || !readers.ValidTimeRange(req.pageMeta.To) {
		return apiutil.ErrInvalidTimeRange
	}

	if req.pageMeta.Comparator != "" &&
		req.pageMeta.Comparator != readers.EqualKey &&
		req.pageMeta.Comparator != readers.LowerThanKey &&
		req.pageMeta.Comparator != readers.LowerThanEqualKey &&
		req.pageMeta.Comparator != readers.GreaterThanKey &&
		req.pageMeta.Comparator != readers.GreaterThanEqualKey {
		return apiutil.ErrInvalidComparator
	}

	if req.pageMeta.Aggregation != "" {
		if req.pageMeta.From == 0 {
			return apiutil.ErrMissingFrom
		}

		if req.pageMeta.To == 0 {
			return apiutil.ErrMissingTo
		}

		if !slices.Contains(validAggregations, strings.ToUpper(req.pageMeta.Aggregation)) {
			return apiutil.ErrInvalidAggregation
		}

		if _, err := time.ParseDuration(req.pageMeta.Interval); err != nil {
			return apiutil.ErrInvalidInterval
		}
	}

	return nil
}

// deviceViewReq is the shared request shape of both MG-15 observed-device
// endpoints: list the distinct devices seen through a gateway, or the
// distinct gateways seen relaying a device. filterVal is that gateway's
// publisher id or that device's serial, taken from the URL. The device's
// serial is checked against the caller's grant exactly as an explicit
// device_id filter would be on a message read; the gateway's publisher id is
// the source of the roster rather than a filter the caller must hold, and
// the caller's DeviceScope does the narrowing instead (see ListGatewayDevices
// in readers/messages.go).
//
// filterIsPublisher marks the gateway->devices direction, where filterVal is
// a publisher client id and must therefore parse as a UUID: the publishers
// column is a UUID column, so a malformed value would otherwise surface as a
// database error rather than a request error. The device->gateways direction
// carries a device serial, which has no format constraint at all (MG-09;
// Atom's external_id accepts `/`, spaces and unicode), so it is left
// unvalidated.
type deviceViewReq struct {
	chanID            string
	token             string
	workspace         string
	key               string
	filterVal         string
	filterIsPublisher bool
	pageMeta          readers.PageMetadata
}

func (req deviceViewReq) validate() error {
	if req.token == "" && req.key == "" {
		return apiutil.ErrBearerToken
	}

	if req.chanID == "" || req.filterVal == "" {
		return apiutil.ErrMissingID
	}

	if req.filterIsPublisher {
		if err := api.ValidateUUID(req.filterVal); err != nil {
			return err
		}
	}

	if req.pageMeta.Limit < 1 || req.pageMeta.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	if !readers.ValidTimeRange(req.pageMeta.From) || !readers.ValidTimeRange(req.pageMeta.To) {
		return apiutil.ErrInvalidTimeRange
	}

	return nil
}
