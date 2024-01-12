// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/eventlogs"
	"github.com/absmach/magistrala/internal/apiutil"
)

const maxLimitSize = 1000

type listEventsReq struct {
	token string
	page  eventlogs.Page
}

func (req listEventsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.page.ID == "" {
		return apiutil.ErrMissingID
	}
	if req.page.EntityType == "" {
		return apiutil.ErrMissingEntityType
	}
	if req.page.EntityType != auth.UserType && req.page.EntityType != auth.GroupType && req.page.EntityType != auth.ThingType && req.page.EntityType != auth.DomainType && req.page.EntityType != auth.PlatformType {
		return apiutil.ErrInvalidEntityType
	}
	if req.page.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	return nil
}
