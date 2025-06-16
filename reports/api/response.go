// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"
	"time"

	"github.com/absmach/magistrala/reports"
	"github.com/absmach/supermq"
)

var (
	_ supermq.Response = (*addReportConfigRes)(nil)
	_ supermq.Response = (*viewReportConfigRes)(nil)
	_ supermq.Response = (*updateReportConfigRes)(nil)
	_ supermq.Response = (*deleteReportConfigRes)(nil)
	_ supermq.Response = (*listReportsConfigRes)(nil)
)

type pageRes struct {
	Limit  uint64 `json:"limit,omitempty"`
	Offset uint64 `json:"offset"`
	Total  uint64 `json:"total"`
}

type generateReportResp struct {
	Total       uint64            `json:"total"`
	From        time.Time         `json:"from,omitempty"`
	To          time.Time         `json:"to,omitempty"`
	Aggregation reports.AggConfig `json:"aggregation,omitempty"`
	Reports     []reports.Report  `json:"reports,omitempty"`
}

func (res generateReportResp) Code() int {
	return http.StatusCreated
}

func (res generateReportResp) Headers() map[string]string {
	return map[string]string{}
}

func (res generateReportResp) Empty() bool {
	return false
}

type addReportConfigRes struct {
	reports.ReportConfig `json:",inline"`
	created              bool
}

func (res addReportConfigRes) Code() int {
	if res.created {
		return http.StatusCreated
	}
	return http.StatusOK
}

func (res addReportConfigRes) Headers() map[string]string {
	if res.created {
		return map[string]string{}
	}
	return map[string]string{}
}

func (res addReportConfigRes) Empty() bool {
	return false
}

type viewReportConfigRes struct {
	reports.ReportConfig `json:",inline"`
}

func (res viewReportConfigRes) Code() int {
	return http.StatusOK
}

func (res viewReportConfigRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewReportConfigRes) Empty() bool {
	return false
}

type updateReportConfigRes struct {
	reports.ReportConfig `json:",inline"`
}

func (res updateReportConfigRes) Code() int {
	return http.StatusOK
}

func (res updateReportConfigRes) Headers() map[string]string {
	return map[string]string{}
}

func (res updateReportConfigRes) Empty() bool {
	return false
}

type deleteReportConfigRes struct {
	deleted bool
}

func (res deleteReportConfigRes) Code() int {
	if res.deleted {
		return http.StatusNoContent
	}
	return http.StatusOK
}

func (res deleteReportConfigRes) Headers() map[string]string {
	return map[string]string{}
}

func (res deleteReportConfigRes) Empty() bool {
	return true
}

type listReportsConfigRes struct {
	pageRes
	ReportConfigs []reports.ReportConfig `json:"report_configs"`
}

func (res listReportsConfigRes) Code() int {
	return http.StatusOK
}

func (res listReportsConfigRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listReportsConfigRes) Empty() bool {
	return false
}

type downloadReportResp struct {
	File reports.ReportFile
}

func (res downloadReportResp) Code() int {
	return http.StatusOK
}

func (res downloadReportResp) Headers() map[string]string {
	return map[string]string{}
}

func (res downloadReportResp) Empty() bool {
	return false
}

type emailReportResp struct{}

func (res emailReportResp) Code() int {
	return http.StatusOK
}

func (res emailReportResp) Headers() map[string]string {
	return map[string]string{}
}

func (res emailReportResp) Empty() bool {
	return true
}
