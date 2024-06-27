// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/journal"
)

type retrieveJournalsReq struct {
	token string
	page  journal.Page
}

func (req retrieveJournalsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.page.Limit > api.DefLimit {
		return apiutil.ErrLimitSize
	}
	if req.page.Direction != "" && req.page.Direction != api.AscDir && req.page.Direction != api.DescDir {
		return apiutil.ErrInvalidDirection
	}
	if req.page.EntityID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}
