// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/absmach/magistrala/re"
	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
)

const (
	maxLimitSize = 1000
	MaxNameSize  = 1024
)

type addRuleReq struct {
	re.Rule
}

func (req addRuleReq) validate() error {
	if len(req.Name) > api.MaxNameSize || req.Name == "" {
		return apiutil.ErrNameSize
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
	if req.Dir != "" && (req.Dir != api.AscDir && req.Dir != api.DescDir) {
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
	if len(req.Rule.Logic.Value) == 0 {
		return apiutil.ErrEmptyList
	}
	if len(req.Rule.Name) > api.MaxNameSize || req.Rule.Name == "" {
		return apiutil.ErrNameSize
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
