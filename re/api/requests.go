// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/absmach/magistrala/pkg/schedule"
	"github.com/absmach/magistrala/re"
	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/pkg/errors"
)

const (
	maxLimitSize = 1000
	MaxNameSize  = 1024
	MaxTitleSize = 37
)

type addRuleReq struct {
	re.Rule
}

func (req addRuleReq) validate() error {
	if len(req.Name) > api.MaxNameSize || req.Name == "" {
		return apiutil.ErrNameSize
	}
	if err := req.Rule.Schedule.Validate(); err != nil {
		return errors.Wrap(err, apiutil.ErrValidation)
	}

	return nil
}

type viewRuleReq struct {
	id string
}

func (req viewRuleReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type listRulesReq struct {
	re.PageMeta
}

func (req listRulesReq) validate() error {
	if req.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	switch req.Order {
	case "", api.NameKey, api.CreatedAtOrder, api.UpdatedAtOrder:
	default:
		return errors.Wrap(apiutil.ErrInvalidOrder, apiutil.ErrValidation)
	}

	if req.Dir != api.AscDir && req.Dir != api.DescDir {
		return apiutil.ErrInvalidDirection
	}

	return nil
}

type updateRuleReq struct {
	Rule re.Rule
}

func (req updateRuleReq) validate() error {
	if req.Rule.ID == "" {
		return apiutil.ErrMissingID
	}
	if len(req.Rule.Name) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}

	return nil
}

type updateRuleTagsReq struct {
	id   string
	Tags []string `json:"tags,omitempty"`
}

func (req updateRuleTagsReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type updateRuleScheduleReq struct {
	id       string
	Schedule schedule.Schedule `json:"schedule,omitempty"`
}

func (req updateRuleScheduleReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	if err := req.Schedule.Validate(); err != nil {
		return errors.Wrap(err, apiutil.ErrValidation)
	}

	return nil
}

type updateRuleStatusReq struct {
	id string
}

func (req updateRuleStatusReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type deleteRuleReq struct {
	id string
}

func (req deleteRuleReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}
