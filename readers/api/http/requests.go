// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"slices"
	"strings"
	"time"

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

// deviceViewDefaultWindow bounds an MG-15 observed-device query to the last
// 24h when the caller supplies neither from nor to. `GROUP BY device_id`
// (or publisher) with no time bound at all is a full, unbounded partition
// scan on a busy channel — this keeps that form from being the easy one to
// call by accident. A caller who supplies either bound is assumed to mean
// it and gets exactly what they asked for, unmodified.
const deviceViewDefaultWindow = 24 * time.Hour

func defaultTimeWindow(from, to float64) (float64, float64) {
	if from != 0 || to != 0 {
		return from, to
	}
	now := time.Now()
	return float64(now.Add(-deviceViewDefaultWindow).Unix()), float64(now.Unix())
}

// deviceViewReq is the shared request shape of both MG-15 observed-device
// endpoints: list the distinct devices seen through a gateway, or the
// distinct gateways seen relaying a device. filterVal is that gateway's
// publisher id or that device's serial, taken from the URL, and is checked
// against the caller's grant exactly as an explicit publisher/device_id
// filter would be on a message read.
type deviceViewReq struct {
	chanID    string
	token     string
	domain    string
	key       string
	filterVal string
	pageMeta  readers.PageMetadata
}

func (req deviceViewReq) validate() error {
	if req.token == "" && req.key == "" {
		return apiutil.ErrBearerToken
	}

	if req.chanID == "" || req.filterVal == "" {
		return apiutil.ErrMissingID
	}

	if req.pageMeta.Limit < 1 || req.pageMeta.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	return nil
}
