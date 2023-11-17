// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/readers"
)

const maxLimitSize = 1000

type listMessagesReq struct {
	chanID   string
	token    string
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

	return nil
}
