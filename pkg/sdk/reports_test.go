// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/absmach/magistrala/pkg/sdk"
	"github.com/stretchr/testify/assert"
)

const reportConfigID = "report-config-1"

var testReportConfig = sdk.ReportConfig{
	ID:          reportConfigID,
	Name:        "daily-report",
	Description: "Daily temperature report",
	DomainID:    domainID,
	Status:      "enabled",
}

func TestAddReportConfig(t *testing.T) {
	cases := []struct {
		desc    string
		cfg     sdk.ReportConfig
		token   string
		handler http.HandlerFunc
		wantErr bool
		resp    sdk.ReportConfig
	}{
		{
			desc:  "add report config successfully",
			cfg:   sdk.ReportConfig{Name: "daily-report", Description: "desc"},
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, fmt.Sprintf("/%s/reports/configs", domainID), r.URL.Path)
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(testReportConfig)
			},
			resp: testReportConfig,
		},
		{
			desc:  "add report config with empty token",
			cfg:   sdk.ReportConfig{Name: "daily-report"},
			token: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantErr: true,
		},
		{
			desc:  "add report config with bad request",
			cfg:   sdk.ReportConfig{},
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			mgsdk := sdk.NewSDK(sdk.Config{ReportsURL: server.URL})
			result, err := mgsdk.AddReportConfig(context.Background(), tc.cfg, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.Equal(t, tc.resp, result)
			}
		})
	}
}

func TestViewReportConfig(t *testing.T) {
	cases := []struct {
		desc    string
		id      string
		token   string
		handler http.HandlerFunc
		wantErr bool
		resp    sdk.ReportConfig
	}{
		{
			desc:  "view report config successfully",
			id:    reportConfigID,
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, fmt.Sprintf("/%s/reports/configs/%s", domainID, reportConfigID), r.URL.Path)
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(testReportConfig)
			},
			resp: testReportConfig,
		},
		{
			desc:  "view report config with empty token",
			id:    reportConfigID,
			token: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantErr: true,
		},
		{
			desc:  "view non-existent report config",
			id:    "non-existent",
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			mgsdk := sdk.NewSDK(sdk.Config{ReportsURL: server.URL})
			result, err := mgsdk.ViewReportConfig(context.Background(), tc.id, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.Equal(t, tc.resp, result)
			}
		})
	}
}

func TestUpdateReportConfig(t *testing.T) {
	updated := testReportConfig
	updated.Description = "updated description"

	cases := []struct {
		desc    string
		cfg     sdk.ReportConfig
		token   string
		handler http.HandlerFunc
		wantErr bool
		resp    sdk.ReportConfig
	}{
		{
			desc:  "update report config successfully",
			cfg:   testReportConfig,
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPatch, r.Method)
				assert.Equal(t, fmt.Sprintf("/%s/reports/configs/%s", domainID, reportConfigID), r.URL.Path)
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(updated)
			},
			resp: updated,
		},
		{
			desc:  "update report config with empty token",
			cfg:   testReportConfig,
			token: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			mgsdk := sdk.NewSDK(sdk.Config{ReportsURL: server.URL})
			result, err := mgsdk.UpdateReportConfig(context.Background(), tc.cfg, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.Equal(t, tc.resp, result)
			}
		})
	}
}

func TestUpdateReportSchedule(t *testing.T) {
	updated := testReportConfig
	updated.Schedule = map[string]any{"cron": "0 9 * * *"}

	cases := []struct {
		desc    string
		cfg     sdk.ReportConfig
		token   string
		handler http.HandlerFunc
		wantErr bool
		resp    sdk.ReportConfig
	}{
		{
			desc:  "update report schedule successfully",
			cfg:   sdk.ReportConfig{ID: reportConfigID, Schedule: map[string]any{"cron": "0 9 * * *"}},
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPatch, r.Method)
				assert.Equal(t, fmt.Sprintf("/%s/reports/configs/%s/schedule", domainID, reportConfigID), r.URL.Path)
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(updated)
			},
			resp: updated,
		},
		{
			desc:  "update report schedule with empty token",
			cfg:   sdk.ReportConfig{ID: reportConfigID},
			token: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			mgsdk := sdk.NewSDK(sdk.Config{ReportsURL: server.URL})
			result, err := mgsdk.UpdateReportSchedule(context.Background(), tc.cfg, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.Equal(t, tc.resp, result)
			}
		})
	}
}

func TestRemoveReportConfig(t *testing.T) {
	cases := []struct {
		desc    string
		id      string
		token   string
		handler http.HandlerFunc
		wantErr bool
	}{
		{
			desc:  "remove report config successfully",
			id:    reportConfigID,
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, fmt.Sprintf("/%s/reports/configs/%s", domainID, reportConfigID), r.URL.Path)
				w.WriteHeader(http.StatusNoContent)
			},
		},
		{
			desc:  "remove report config with empty token",
			id:    reportConfigID,
			token: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantErr: true,
		},
		{
			desc:  "remove non-existent report config",
			id:    "non-existent",
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			mgsdk := sdk.NewSDK(sdk.Config{ReportsURL: server.URL})
			err := mgsdk.RemoveReportConfig(context.Background(), tc.id, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
		})
	}
}

func TestListReportsConfig(t *testing.T) {
	page := sdk.ReportConfigPage{
		Total:         2,
		Offset:        0,
		Limit:         10,
		ReportConfigs: []sdk.ReportConfig{testReportConfig, {ID: "report-2", Name: "weekly"}},
	}

	cases := []struct {
		desc    string
		pm      sdk.PageMetadata
		token   string
		checkQ  func(t *testing.T, r *http.Request)
		wantErr bool
		resp    sdk.ReportConfigPage
	}{
		{
			desc:  "list reports config successfully",
			pm:    sdk.PageMetadata{Offset: 0, Limit: 10},
			token: validToken,
			checkQ: func(t *testing.T, r *http.Request) {
				assert.Equal(t, "10", r.URL.Query().Get("limit"))
			},
			resp: page,
		},
		{
			desc: "list reports config with filters",
			pm: sdk.PageMetadata{
				Limit:  10,
				Name:   "daily",
				Status: "enabled",
				Dir:    "desc",
				Order:  "created_at",
			},
			token: validToken,
			checkQ: func(t *testing.T, r *http.Request) {
				q := r.URL.Query()
				assert.Equal(t, "enabled", q.Get("status"))
				assert.Equal(t, "daily", q.Get("name"))
				assert.Equal(t, "desc", q.Get("dir"))
				assert.Equal(t, "created_at", q.Get("order"))
			},
			resp: page,
		},
		{
			desc:  "list reports config with empty metadata excludes filter params",
			pm:    sdk.PageMetadata{},
			token: validToken,
			checkQ: func(t *testing.T, r *http.Request) {
				rawQ := r.URL.RawQuery
				assert.NotContains(t, rawQ, "status=")
				assert.NotContains(t, rawQ, "dir=")
				assert.NotContains(t, rawQ, "order=")
			},
			resp: sdk.ReportConfigPage{},
		},
		{
			desc:    "list reports config with empty token",
			pm:      sdk.PageMetadata{Limit: 10},
			token:   "",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, fmt.Sprintf("/%s/reports/configs", domainID), r.URL.Path)
				if r.Header.Get("Authorization") == "" || r.Header.Get("Authorization") == "Bearer " {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				if tc.checkQ != nil {
					tc.checkQ(t, r)
				}
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(tc.resp)
			}))
			defer server.Close()

			mgsdk := sdk.NewSDK(sdk.Config{ReportsURL: server.URL})
			result, err := mgsdk.ListReportsConfig(context.Background(), tc.pm, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.Equal(t, tc.resp, result)
			}
		})
	}
}

func TestEnableReportConfig(t *testing.T) {
	enabled := testReportConfig
	enabled.Status = "enabled"

	cases := []struct {
		desc    string
		id      string
		token   string
		handler http.HandlerFunc
		wantErr bool
		resp    sdk.ReportConfig
	}{
		{
			desc:  "enable report config successfully",
			id:    reportConfigID,
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, fmt.Sprintf("/%s/reports/configs/%s/enable", domainID, reportConfigID), r.URL.Path)
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(enabled)
			},
			resp: enabled,
		},
		{
			desc:  "enable report config with empty token",
			id:    reportConfigID,
			token: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			mgsdk := sdk.NewSDK(sdk.Config{ReportsURL: server.URL})
			result, err := mgsdk.EnableReportConfig(context.Background(), tc.id, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.Equal(t, tc.resp, result)
			}
		})
	}
}

func TestDisableReportConfig(t *testing.T) {
	disabled := testReportConfig
	disabled.Status = "disabled"

	cases := []struct {
		desc    string
		id      string
		token   string
		handler http.HandlerFunc
		wantErr bool
		resp    sdk.ReportConfig
	}{
		{
			desc:  "disable report config successfully",
			id:    reportConfigID,
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, fmt.Sprintf("/%s/reports/configs/%s/disable", domainID, reportConfigID), r.URL.Path)
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(disabled)
			},
			resp: disabled,
		},
		{
			desc:  "disable report config with empty token",
			id:    reportConfigID,
			token: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			mgsdk := sdk.NewSDK(sdk.Config{ReportsURL: server.URL})
			result, err := mgsdk.DisableReportConfig(context.Background(), tc.id, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.Equal(t, tc.resp, result)
			}
		})
	}
}

func TestUpdateReportTemplate(t *testing.T) {
	cases := []struct {
		desc    string
		cfg     sdk.ReportConfig
		token   string
		handler http.HandlerFunc
		wantErr bool
	}{
		{
			desc: "update report template successfully",
			cfg: sdk.ReportConfig{
				ID:             reportConfigID,
				ReportTemplate: sdk.ReportTemplate{Header: "Header text", Footer: "Footer text"},
			},
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPut, r.Method)
				assert.Equal(t, fmt.Sprintf("/%s/reports/configs/%s/template", domainID, reportConfigID), r.URL.Path)
				w.WriteHeader(http.StatusOK)
			},
		},
		{
			desc:  "update report template with empty token",
			cfg:   sdk.ReportConfig{ID: reportConfigID},
			token: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			mgsdk := sdk.NewSDK(sdk.Config{ReportsURL: server.URL})
			err := mgsdk.UpdateReportTemplate(context.Background(), tc.cfg, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
		})
	}
}

func TestViewReportTemplate(t *testing.T) {
	tmpl := sdk.ReportTemplate{Header: "Header", Footer: "Footer"}

	cases := []struct {
		desc    string
		id      string
		token   string
		handler http.HandlerFunc
		wantErr bool
		resp    sdk.ReportTemplate
	}{
		{
			desc:  "view report template successfully",
			id:    reportConfigID,
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, fmt.Sprintf("/%s/reports/configs/%s/template", domainID, reportConfigID), r.URL.Path)
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(tmpl)
			},
			resp: tmpl,
		},
		{
			desc:  "view report template with empty token",
			id:    reportConfigID,
			token: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			mgsdk := sdk.NewSDK(sdk.Config{ReportsURL: server.URL})
			result, err := mgsdk.ViewReportTemplate(context.Background(), tc.id, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.Equal(t, tc.resp, result)
			}
		})
	}
}

func TestDeleteReportTemplate(t *testing.T) {
	cases := []struct {
		desc    string
		id      string
		token   string
		handler http.HandlerFunc
		wantErr bool
	}{
		{
			desc:  "delete report template successfully",
			id:    reportConfigID,
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, fmt.Sprintf("/%s/reports/configs/%s/template", domainID, reportConfigID), r.URL.Path)
				w.WriteHeader(http.StatusNoContent)
			},
		},
		{
			desc:  "delete report template with empty token",
			id:    reportConfigID,
			token: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			mgsdk := sdk.NewSDK(sdk.Config{ReportsURL: server.URL})
			err := mgsdk.DeleteReportTemplate(context.Background(), tc.id, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
		})
	}
}

func TestGenerateReport(t *testing.T) {
	reportPage := sdk.ReportPage{Total: 1}

	cases := []struct {
		desc    string
		cfg     sdk.ReportConfig
		action  sdk.ReportAction
		token   string
		handler http.HandlerFunc
		wantErr bool
		resp    sdk.ReportPage
	}{
		{
			desc:   "generate report successfully",
			cfg:    testReportConfig,
			action: sdk.ViewReportAction,
			token:  validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, fmt.Sprintf("/%s/reports", domainID), r.URL.Path)
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(reportPage)
			},
			resp: reportPage,
		},
		{
			desc:   "generate report with download action",
			cfg:    testReportConfig,
			action: sdk.DownloadReportAction,
			token:  validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(reportPage)
			},
			resp: reportPage,
		},
		{
			desc:   "generate report with empty token",
			cfg:    testReportConfig,
			action: sdk.ViewReportAction,
			token:  "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			mgsdk := sdk.NewSDK(sdk.Config{ReportsURL: server.URL})
			result, err := mgsdk.GenerateReport(context.Background(), tc.cfg, tc.action, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.Equal(t, tc.resp, result)
			}
		})
	}
}

