// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/absmach/magistrala/re"
	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
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
	if len(req.Rule.Name) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}

	return nil
}

type updateRuleScheduleReq struct {
	id       string
	Schedule re.Schedule `json:"schedule,omitempty"`
}

func (req updateRuleScheduleReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
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

type updateReportConfigReq struct {
	re.ReportConfig `json:",inline"`
}

func (req updateReportConfigReq) validate() error {
	if req.ID == "" {
		return svcerr.ErrMalformedEntity
	}
	if req.Name == "" {
		return svcerr.ErrMalformedEntity
	}
	if len(req.ChannelIDs) == 0 {
		return svcerr.ErrMalformedEntity
	}
	return nil
}

type addReportConfigReq struct {
	re.ReportConfig `json:",inline"`
}

func (req addReportConfigReq) validate() error {
	if req.Name == "" {
		return svcerr.ErrMalformedEntity
	}
	if len(req.ChannelIDs) == 0 {
		return svcerr.ErrMalformedEntity
	}
	if req.Limit == 0 {
		return apiutil.ErrValidation
	}
	return nil
}

type viewReportConfigReq struct {
	ID string `json:"id"`
}

func (req viewReportConfigReq) validate() error {
	if req.ID == "" {
		return svcerr.ErrMalformedEntity
	}
	return nil
}

type listReportsConfigReq struct {
	re.PageMeta `json:",inline"`
}

func (req listReportsConfigReq) validate() error {
	if req.Limit > maxLimitSize {
		return svcerr.ErrMalformedEntity
	}
	return nil
}

type deleteReportConfigReq struct {
	ID string `json:"id"`
}

func (req deleteReportConfigReq) validate() error {
	if req.ID == "" {
		return svcerr.ErrMalformedEntity
	}
	return nil
}

type generateReportReq struct {
	re.ReportConfig
}

func (req generateReportReq) validate() error {
	if req.Name == "" {
		return svcerr.ErrMalformedEntity
	}
	if len(req.ChannelIDs) == 0 {
		return svcerr.ErrMalformedEntity
	}
	if req.Limit == 0 {
		return apiutil.ErrValidation
	}
	return nil
}

type updateReportStatusReq struct {
	id string
}

func (req updateReportStatusReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}
