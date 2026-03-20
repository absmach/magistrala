// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/absmach/supermq/pkg/errors"
)

const (
	reportsEndpoint        = "reports"
	configsEndpointReports = "configs"
)

// ReportConfig represents a report configuration.
type ReportConfig struct {
	ID             string         `json:"id,omitempty"`
	Name           string         `json:"name,omitempty"`
	Description    string         `json:"description,omitempty"`
	DomainID       string         `json:"domain_id,omitempty"`
	Schedule       any            `json:"schedule,omitempty"`
	Config         any            `json:"config,omitempty"`
	Email          any            `json:"email,omitempty"`
	Metrics        any            `json:"metrics,omitempty"`
	ReportTemplate ReportTemplate `json:"report_template,omitempty"`
	Status         string         `json:"status,omitempty"`
	CreatedAt      time.Time      `json:"created_at,omitempty"`
	CreatedBy      string         `json:"created_by,omitempty"`
	UpdatedAt      time.Time      `json:"updated_at,omitempty"`
	UpdatedBy      string         `json:"updated_by,omitempty"`
}

type ReportTemplate any

type ReportFile struct {
	Name   string
	Format string
	Data   []byte
}

type ReportPage struct {
	Total       uint64    `json:"total"`
	From        time.Time `json:"from,omitempty"`
	To          time.Time `json:"to,omitempty"`
	Aggregation any       `json:"aggregation,omitempty"`
	Reports     any       `json:"reports,omitempty"`
	File        any       `json:"file,omitempty"`
}

type ReportConfigPage struct {
	Total         uint64         `json:"total"`
	Offset        uint64         `json:"offset"`
	Limit         uint64         `json:"limit"`
	ReportConfigs []ReportConfig `json:"report_configs"`
}

type ReportAction string

const (
	ViewReportAction     ReportAction = "view"
	DownloadReportAction ReportAction = "download"
	EmailReportAction    ReportAction = "email"
)

func (sdk mgSDK) AddReportConfig(ctx context.Context, cfg ReportConfig, domainID, token string) (ReportConfig, errors.SDKError) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return ReportConfig{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s", sdk.reportsURL, domainID, reportsEndpoint, configsEndpointReports)

	_, body, sdkerr := sdk.processRequest(ctx, http.MethodPost, url, token, data, nil, http.StatusCreated, http.StatusOK)
	if sdkerr != nil {
		return ReportConfig{}, sdkerr
	}

	var rc ReportConfig
	if err := json.Unmarshal(body, &rc); err != nil {
		return ReportConfig{}, errors.NewSDKError(err)
	}

	return rc, nil
}

func (sdk mgSDK) ViewReportConfig(ctx context.Context, id, domainID, token string) (ReportConfig, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.reportsURL, domainID, reportsEndpoint, configsEndpointReports, id)

	_, body, sdkerr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return ReportConfig{}, sdkerr
	}

	var rc ReportConfig
	if err := json.Unmarshal(body, &rc); err != nil {
		return ReportConfig{}, errors.NewSDKError(err)
	}

	return rc, nil
}

func (sdk mgSDK) UpdateReportConfig(ctx context.Context, cfg ReportConfig, domainID, token string) (ReportConfig, errors.SDKError) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return ReportConfig{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.reportsURL, domainID, reportsEndpoint, configsEndpointReports, cfg.ID)

	_, body, sdkerr := sdk.processRequest(ctx, http.MethodPatch, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return ReportConfig{}, sdkerr
	}

	var rc ReportConfig
	if err := json.Unmarshal(body, &rc); err != nil {
		return ReportConfig{}, errors.NewSDKError(err)
	}

	return rc, nil
}

func (sdk mgSDK) UpdateReportSchedule(ctx context.Context, cfg ReportConfig, domainID, token string) (ReportConfig, errors.SDKError) {
	data, err := json.Marshal(map[string]any{"schedule": cfg.Schedule})
	if err != nil {
		return ReportConfig{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s/schedule", sdk.reportsURL, domainID, reportsEndpoint, configsEndpointReports, cfg.ID)

	_, body, sdkerr := sdk.processRequest(ctx, http.MethodPatch, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return ReportConfig{}, sdkerr
	}

	var rc ReportConfig
	if err := json.Unmarshal(body, &rc); err != nil {
		return ReportConfig{}, errors.NewSDKError(err)
	}

	return rc, nil
}

func (sdk mgSDK) RemoveReportConfig(ctx context.Context, id, domainID, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.reportsURL, domainID, reportsEndpoint, configsEndpointReports, id)

	_, _, sdkerr := sdk.processRequest(ctx, http.MethodDelete, url, token, nil, nil, http.StatusNoContent, http.StatusOK)
	return sdkerr
}

func (sdk mgSDK) ListReportsConfig(ctx context.Context, pm PageMetadata, domainID, token string) (ReportConfigPage, errors.SDKError) {
	endpoint := fmt.Sprintf("%s/%s/%s", domainID, reportsEndpoint, configsEndpointReports)
	url, err := sdk.withQueryParams(sdk.reportsURL, endpoint, pm)
	if err != nil {
		return ReportConfigPage{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return ReportConfigPage{}, sdkerr
	}

	var rcp ReportConfigPage
	if err := json.Unmarshal(body, &rcp); err != nil {
		return ReportConfigPage{}, errors.NewSDKError(err)
	}

	return rcp, nil
}

func (sdk mgSDK) EnableReportConfig(ctx context.Context, id, domainID, token string) (ReportConfig, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s/%s/enable", sdk.reportsURL, domainID, reportsEndpoint, configsEndpointReports, id)

	_, body, sdkerr := sdk.processRequest(ctx, http.MethodPost, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return ReportConfig{}, sdkerr
	}

	var rc ReportConfig
	if err := json.Unmarshal(body, &rc); err != nil {
		return ReportConfig{}, errors.NewSDKError(err)
	}

	return rc, nil
}

func (sdk mgSDK) DisableReportConfig(ctx context.Context, id, domainID, token string) (ReportConfig, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s/%s/disable", sdk.reportsURL, domainID, reportsEndpoint, configsEndpointReports, id)

	_, body, sdkerr := sdk.processRequest(ctx, http.MethodPost, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return ReportConfig{}, sdkerr
	}

	var rc ReportConfig
	if err := json.Unmarshal(body, &rc); err != nil {
		return ReportConfig{}, errors.NewSDKError(err)
	}

	return rc, nil
}

func (sdk mgSDK) UpdateReportTemplate(ctx context.Context, cfg ReportConfig, domainID, token string) errors.SDKError {
	data, err := json.Marshal(cfg)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s/template", sdk.reportsURL, domainID, reportsEndpoint, configsEndpointReports, cfg.ID)

	_, _, sdkerr := sdk.processRequest(ctx, http.MethodPut, url, token, data, nil, http.StatusNoContent)
	return sdkerr
}

func (sdk mgSDK) ViewReportTemplate(ctx context.Context, id, domainID, token string) (ReportTemplate, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s/%s/template", sdk.reportsURL, domainID, reportsEndpoint, configsEndpointReports, id)

	_, body, sdkerr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return "", sdkerr
	}

	var rt ReportTemplate
	if err := json.Unmarshal(body, &rt); err != nil {
		return "", errors.NewSDKError(err)
	}

	return rt, nil
}

func (sdk mgSDK) DeleteReportTemplate(ctx context.Context, id, domainID, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s/%s/%s/template", sdk.reportsURL, domainID, reportsEndpoint, configsEndpointReports, id)

	_, _, sdkerr := sdk.processRequest(ctx, http.MethodDelete, url, token, nil, nil, http.StatusNoContent, http.StatusOK)
	return sdkerr
}

func (sdk mgSDK) GenerateReport(
	ctx context.Context,
	config ReportConfig,
	action ReportAction,
	domainID,
	token string,
) (ReportPage, *ReportFile, errors.SDKError) {
	data, err := json.Marshal(config)
	if err != nil {
		return ReportPage{}, nil, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s?action=%s",
		sdk.reportsURL,
		domainID,
		reportsEndpoint,
		action,
	)

	headers, body, sdkerr := sdk.processRequest(
		ctx,
		http.MethodPost,
		url,
		token,
		data,
		nil,
		http.StatusOK,
	)
	if sdkerr != nil {
		return ReportPage{}, nil, sdkerr
	}

	// ✅ Handle Download Action
	if action == DownloadReportAction {
		file := &ReportFile{
			Name:   extractFilename(headers.Get("Content-Disposition")),
			Format: "pdf",
			Data:   body,
		}
		return ReportPage{}, file, nil
	}

	// ✅ Handle JSON response (view/email)
	var rp ReportPage
	if err := json.Unmarshal(body, &rp); err != nil {
		return ReportPage{}, nil, errors.NewSDKError(err)
	}

	return rp, nil, nil
}

func extractFilename(contentDisposition string) string {
	const prefix = "filename="
	if idx := strings.Index(contentDisposition, prefix); idx != -1 {
		return strings.Trim(contentDisposition[idx+len(prefix):], `"`)
	}
	return "report"
}
