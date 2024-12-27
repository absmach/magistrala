// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/re"
)

const maxLimitSize = 1000

type addRuleReq struct {
	re.Rule
}

func (req addRuleReq) validate() error {
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
	Rule re.Rule `json:",inline"`
}

func (req updateRuleReq) validate() error {
	if req.Rule.ID == "" {
		return apiutil.ErrMissingID
	}
	if len(req.Rule.Logic.Value) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}

type changeRuleStatusReq struct {
	id     string
	status re.Status
}

func (req changeRuleStatusReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}
