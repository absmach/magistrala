// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/journal"
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

type retrieveClientTelemetryReq struct {
	clientID string
}

func (req retrieveClientTelemetryReq) validate() error {
	if req.clientID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}
