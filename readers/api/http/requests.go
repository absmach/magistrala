// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"slices"
	"strings"
	"time"

	api "github.com/absmach/magistrala/api/http"
	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/readers"
)

const maxLimitSize = 1000

var validAggregations = []string{"MAX", "MIN", "AVG", "SUM", "COUNT"}

type listMessagesReq struct {
	chanID   string
	token    string
	domain   string
	key      string
	pageMeta readers.PageMetadata
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

// deviceViewDefaultWindow bounds an MG-15 observed-device query. `GROUP BY
// device_id` (or publisher) with no time bound at all is a full, unbounded
// partition scan on a busy channel — this keeps that form from being the easy
// one to call by accident.
//
// The window is expressed in the readers package's stored-time unit, which
// for the built-in senml/json transformers is Unix nanoseconds: they run
// every absolute time through transformers.ToUnixNano before the writers
// persist it, and PageMetadata.From/To and the returned LastSeen values use
// that same representation (see DeviceStat). On a deployment storing any
// other unit a caller relying on the default would get an empty roster for
// the last 24h; supply from/to explicitly in your own unit.
const deviceViewDefaultWindow = 24 * time.Hour

// defaultTimeWindow fills in whichever of the two bounds the caller left at
// zero so the resulting query is always bounded to a 24h window:
//   - neither bound supplied: last 24h, [now-24h, now]
//   - only to supplied:        [to-24h, to]
//   - only from supplied:      [from, from+24h]
//
// The bounds are Unix nanoseconds, matching the unit the senml/json
// transformers store time in (see deviceViewDefaultWindow). A caller who
// supplies both bounds gets exactly what they asked for, unmodified.
func defaultTimeWindow(from, to float64) (float64, float64) {
	window := float64(deviceViewDefaultWindow.Nanoseconds())
	switch {
	case from == 0 && to == 0:
		now := float64(time.Now().UnixNano())
		return now - window, now
	case from == 0:
		return to - window, to
	case to == 0:
		return from, from + window
	default:
		return from, to
	}
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
	domain            string
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

	return nil
}
