// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"

	"github.com/absmach/magistrala/pkg/schedule"
	"github.com/absmach/magistrala/reports"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
)

const (
	maxLimitSize = 1000
	MaxNameSize  = 1024
	MaxTitleSize = 37

	errInvalidMetric = "invalid metric[%d]: %w"
)

var (
	errInvalidReportAction      = errors.New("invalid report action")
	errMetricsNotProvided       = errors.New("metrics not provided")
	errMissingReportConfig      = errors.New("missing report config")
	errMissingReportEmailConfig = errors.New("missing report email config")
	errInvalidRecurringPeriod   = errors.New("invalid recurring period")
	errTitleSize                = errors.New("invalid title size")
)

type addReportConfigReq struct {
	reports.ReportConfig `json:",inline"`
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
	reports.PageMeta `json:",inline"`
}

func (req listReportsConfigReq) validate() error {
	if req.Limit > maxLimitSize {
		return svcerr.ErrMalformedEntity
	}
	return nil
}

type updateReportConfigReq struct {
	reports.ReportConfig `json:",inline"`
}

func (req updateReportConfigReq) validate() error {
	if req.ID == "" {
		return apiutil.ErrMissingID
	}
	return validateReportConfig(req.ReportConfig, false, false)
}

type updateReportScheduleReq struct {
	id       string
	Schedule schedule.Schedule `json:"schedule,omitempty"`
}

func (req updateReportScheduleReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
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
	reports.ReportConfig
	action reports.ReportAction
}

func (req generateReportReq) validate() error {
	if len(req.Config.Title) > MaxTitleSize {
		return errors.Wrap(apiutil.ErrValidation, errTitleSize)
	}

	switch req.action {
	case reports.ViewReport, reports.DownloadReport:
		return validateReportConfig(req.ReportConfig, true, true)
	case reports.EmailReport:
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

func validateReportConfig(req reports.ReportConfig, skipEmailValidation bool, skipSchedularValidation bool) error {
	if len(req.Metrics) == 0 {
		return errors.Wrap(apiutil.ErrValidation, errMetricsNotProvided)
	}
	for i, metric := range req.Metrics {
		if err := metric.Validate(); err != nil {
			return errors.Wrap(apiutil.ErrValidation, fmt.Errorf(errInvalidMetric, i+1, err))
		}
	}

	if req.Config == nil {
		return errors.Wrap(errMissingReportConfig, apiutil.ErrValidation)
	}
	if err := req.Config.Validate(); err != nil {
		return errors.Wrap(err, apiutil.ErrValidation)
	}

	if skipEmailValidation {
		return nil
	}
	if req.Email == nil {
		return errors.Wrap(errMissingReportEmailConfig, apiutil.ErrValidation)
	}
	if err := req.Email.Validate(); err != nil {
		return errors.Wrap(apiutil.ErrValidation, err)
	}

	if skipSchedularValidation {
		return nil
	}

	return validateScheduler(req.Schedule)
}

func validateScheduler(sch schedule.Schedule) error {
	if sch.Recurring != schedule.None && sch.RecurringPeriod < 1 {
		return errors.Wrap(apiutil.ErrValidation, errInvalidRecurringPeriod)
	}
	return nil
}
