// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"

	"github.com/absmach/magistrala/re"
	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
)

var (
	errInvalidReportAction      = errors.New("invalid report action")
	errMetricsNotProvided       = errors.New("metrics not provided")
	errMissingReportConfig      = errors.New("missing report config")
	errMissingReportEmailConfig = errors.New("missing report email config")
	errInvalidRecurringPeriod   = errors.New("invalid recurring period")
	errTitleSize                = errors.New("invalid title size")
)

const (
	maxLimitSize = 1000
	MaxNameSize  = 1024
	MaxTitleSize = 37

	errInvalidMetric = "invalid metric[%d]: %w"
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
		return apiutil.ErrMissingID
	}
	return validateReportConfig(req.ReportConfig, false, false)
}

type updateReportScheduleReq struct {
	id       string
	Schedule re.Schedule `json:"schedule,omitempty"`
}

func (req updateReportScheduleReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type addReportConfigReq struct {
	re.ReportConfig `json:",inline"`
}

func (req addReportConfigReq) validate() error {
	if req.Name == "" {
		return apiutil.ErrMissingName
	}
	return validateReportConfig(req.ReportConfig, false, false)
}

type viewReportConfigReq struct {
	ID string `json:"id"`
}

func (req viewReportConfigReq) validate() error {
	if req.ID == "" {
		return apiutil.ErrMissingID
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
		return apiutil.ErrMissingID
	}
	return nil
}

type generateReportReq struct {
	re.ReportConfig
	action re.ReportAction
}

func (req generateReportReq) validate() error {
	if len(req.Config.Title) > MaxTitleSize {
		return errors.Wrap(apiutil.ErrValidation, errTitleSize)
	}

	switch req.action {
	case re.ViewReport, re.DownloadReport:
		return validateReportConfig(req.ReportConfig, true, true)
	case re.EmailReport:
		return validateReportConfig(req.ReportConfig, false, true)
	default:
		return errors.Wrap(apiutil.ErrValidation, errInvalidReportAction)
	}
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

func validateReportConfig(req re.ReportConfig, skipEmailValidation bool, skipSchedularValidation bool) error {
	if len(req.Metrics) == 0 {
		return errors.Wrap(apiutil.ErrValidation, errMetricsNotProvided)
	}
	for i, metric := range req.Metrics {
		if err := metric.Validate(); err != nil {
			return errors.Wrap(apiutil.ErrValidation, fmt.Errorf(errInvalidMetric, i+1, err))
		}
	}

	if req.Config == nil {
		return errMissingReportConfig
	}
	if err := req.Config.Validate(); err != nil {
		return errors.Wrap(apiutil.ErrValidation, err)
	}

	if skipEmailValidation {
		return nil
	}
	if req.Email == nil {
		return errMissingReportEmailConfig
	}
	if err := req.Email.Validate(); err != nil {
		return errors.Wrap(apiutil.ErrValidation, err)
	}

	if skipSchedularValidation {
		return nil
	}

	return validateScheduler(req.Schedule)
}

func validateScheduler(sch re.Schedule) error {
	if sch.Recurring != re.None && sch.RecurringPeriod < 1 {
		return errInvalidRecurringPeriod
	}
	return nil
}
