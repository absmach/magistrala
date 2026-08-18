// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"slices"
	"strings"
	"time"

	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/readers"
)

const (
	maxLimitSize = 1000

	// deviceViewDefaultWindow bounds an MG-15 observed-device query to the
	// last 24h when the caller supplies neither from nor to, mirroring the
	// HTTP layer's defaultTimeWindow. GROUP BY device_id (or publisher) with
	// no time bound at all is a full, unbounded partition scan on a busy
	// channel, and the gRPC path can trigger it just as easily as the HTTP
	// path — the caller must opt out of the bound by naming one explicitly.
	deviceViewDefaultWindow = 24 * time.Hour

	aggregationMax   = "MAX"
	aggregationMin   = "MIN"
	aggregationAvg   = "AVG"
	aggregationSum   = "SUM"
	aggregationCount = "COUNT"
)

func defaultTimeWindow(from, to float64) (float64, float64) {
	if from != 0 || to != 0 {
		return from, to
	}
	now := time.Now()
	return float64(now.Add(-deviceViewDefaultWindow).Unix()), float64(now.Unix())
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
// device's serial, depending on direction. As with ReadMessages over gRPC,
// there is no per-caller authorization here — pageMeta.DeviceScope has no
// wire representation (see PageMetadata in readers.proto), so a gRPC caller
// always gets the unrestricted query, exactly as ReadMessages does today;
// scoping to a caller's grant is HTTP-only (MG-08).
type deviceViewReq struct {
	chanID    string
	domain    string
	filterVal string
	pageMeta  readers.PageMetadata
}

func (req deviceViewReq) validate() error {
	if req.chanID == "" || req.domain == "" || req.filterVal == "" {
		return apiutil.ErrMissingID
	}

	if req.pageMeta.Limit < 1 || req.pageMeta.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	return nil
}
