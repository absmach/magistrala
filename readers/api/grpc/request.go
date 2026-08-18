// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"slices"
	"strings"
	"time"

	api "github.com/absmach/magistrala/api/http"
	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/readers"
)

const (
	maxLimitSize = 1000

	// deviceViewDefaultWindow bounds an MG-15 observed-device query to the
	// last 24h, mirroring the HTTP layer's defaultTimeWindow. GROUP BY
	// device_id (or publisher) with no time bound at all is a full,
	// unbounded partition scan on a busy channel, and the gRPC path can
	// trigger it just as easily as the HTTP path.
	//
	// The window is expressed in the readers package's stored-time unit,
	// which for the built-in senml/json transformers is Unix nanoseconds:
	// they run every absolute time through transformers.ToUnixNano before
	// the writers persist it, and PageMetadata.From/To and the returned
	// LastSeen values use that same representation (see DeviceStat). On a
	// deployment storing any other unit a caller relying on the default
	// would get an empty roster for the last 24h; supply from/to explicitly
	// in your own unit.
	deviceViewDefaultWindow = 24 * time.Hour

	aggregationMax   = "MAX"
	aggregationMin   = "MIN"
	aggregationAvg   = "AVG"
	aggregationSum   = "SUM"
	aggregationCount = "COUNT"
)

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

var validAggregations = []string{aggregationMax, aggregationMin, aggregationAvg, aggregationSum, aggregationCount}

type readMessagesReq struct {
	chanID   string
	domain   string
	pageMeta readers.PageMetadata
}

func (req readMessagesReq) validate() error {
	if req.chanID == "" {
		return apiutil.ErrMissingID
	}
	if req.domain == "" {
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

	if req.pageMeta.Aggregation == "AGGREGATION_UNSPECIFIED" {
		req.pageMeta.Aggregation = ""
	}

	if agg := strings.ToUpper(req.pageMeta.Aggregation); agg != "" && agg != "AGGREGATION_UNSPECIFIED" {
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

// deviceViewReq is the shared request shape of both MG-15 gRPC
// observed-device methods. filterVal is the gateway's publisher id or the
// device's serial, depending on direction, and filterIsPublisher records
// which one it is: a publisher id must parse as a UUID (the publishers
// column is a UUID column), while a device serial has no format constraint
// at all (MG-09), so it is left unvalidated.
//
// As with ReadMessages over gRPC, there is no per-caller authorization here —
// pageMeta.DeviceScope has no wire representation (see PageMetadata in
// readers.proto), so a gRPC caller always gets the unrestricted query: any
// channel they can read yields the full roster of distinct devices or
// gateways on it, not narrowed to their grant, exactly as ReadMessages
// returns all of a channel's messages today. Scoping to a caller's grant is
// HTTP-only (MG-08).
type deviceViewReq struct {
	chanID            string
	domain            string
	filterVal         string
	filterIsPublisher bool
	pageMeta          readers.PageMetadata
}

func (req deviceViewReq) validate() error {
	if req.chanID == "" || req.domain == "" || req.filterVal == "" {
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
