// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/readers"
)

type apiReq interface {
	validate() error
}

type listMessagesReq struct {
	chanID   string
	token    string
	pageMeta readers.PageMetadata
}

func (req listMessagesReq) validate() error {
	if req.token == "" {
		return errors.ErrAuthentication
	}
	if req.chanID == "" {
		return errors.ErrMalformedEntity
	}
	if req.pageMeta.Limit < 1 || req.pageMeta.Offset < 0 {
		return errors.ErrInvalidQueryParams
	}
	if req.pageMeta.Comparator != "" &&
		req.pageMeta.Comparator != readers.EqualKey &&
		req.pageMeta.Comparator != readers.LowerThanKey &&
		req.pageMeta.Comparator != readers.LowerThanEqualKey &&
		req.pageMeta.Comparator != readers.GreaterThanKey &&
		req.pageMeta.Comparator != readers.GreaterThanEqualKey {
		return errors.ErrInvalidQueryParams
	}

	return nil
}
