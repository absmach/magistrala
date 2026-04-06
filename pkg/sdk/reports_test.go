// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	mglog "github.com/absmach/magistrala/logger"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	authnmocks "github.com/absmach/magistrala/pkg/authn/mocks"
	pkgSch "github.com/absmach/magistrala/pkg/schedule"
	"github.com/absmach/magistrala/pkg/sdk"
	"github.com/absmach/magistrala/reports"
	"github.com/absmach/magistrala/reports/api"
	rmocks "github.com/absmach/magistrala/reports/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	reportConfigID    = "report-config-1"
	reportName        = "daily-report"
	reportUpdatedName = "updated daily-report"
	reportDescription = "Daily temperature report"
	reportUpdatedDesc = "updated Daily temperature report"
	validTemplate     = `<!DOCTYPE html>
<html>
<head>
    <title>{{$.Title}}</title>
    <style>
        body { font-family: Arial, sans-serif; }
        .header { background-color: #f0f0f0; padding: 10px; }
        .content { padding: 20px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>{{$.Title}}</h1>
        <p>Generated on: {{$.GeneratedDate}}</p>
    </div>
    <div class="content">
        <h2>Messages</h2>
        {{range .Messages}}
        <div class="message">
            <p>Time: {{formatTime .Time}}</p>
            <p>Value: {{formatValue .}}</p>
        </div>
        {{end}}
    </div>
</body>
</html>`
)

var (
	now      = time.Now().UTC().Truncate(time.Minute)
	future   = now.Add(1 * time.Hour)
	schedule = pkgSch.Schedule{
		StartDateTime:   future,
		Recurring:       pkgSch.Daily,
		RecurringPeriod: 1,
		Time:            future,
	}
	metrics = []reports.ReqMetric{
		{
			ChannelID: "channel1",
			ClientIDs: []string{"client1"},
			Name:      "metric_name",
		},
	}
	config = reports.MetricConfig{
		From:        "now()-1h",
		To:          "now()",
		Title:       "test_title",
		Aggregation: reports.AggConfig{AggType: reports.AggregationAVG, Interval: "1h"},
	}
	email = reports.EmailSetting{
		To:      []string{"test@example.com"},
		Subject: "Test Report",
	}

	testReportConfig = sdk.ReportConfig{
		ID:          reportConfigID,
		Name:        reportName,
		Description: reportDescription,
		DomainID:    domainID,
		Status:      "enabled",
		Schedule:    schedule,
		Metrics:     metrics,
		Config:      &config,
		Email:       &email,
	}
)

func setupReports() (*httptest.Server, *rmocks.Service, *authnmocks.Authentication) {
	rsvc := new(rmocks.Service)
	log := mglog.NewMock()
	authn := new(authnmocks.Authentication)
	am := smqauthn.NewAuthNMiddleware(authn, smqauthn.WithAllowUnverifiedUser(true))
	mux := chi.NewRouter()
	_ = api.MakeHandler(rsvc, am, mux, log, "")
	return httptest.NewServer(mux), rsvc, authn
}

func TestAddReportConfig(t *testing.T) {
	rs, rsvc, auth := setupReports()
	defer rs.Close()

	conf := sdk.Config{
		ReportsURL: rs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	svcCfg := reports.ReportConfig{
		ID:          reportConfigID,
		Name:        reportName,
		Description: reportDescription,
		DomainID:    domainID,
		Status:      reports.EnabledStatus,
		Schedule:    schedule,
		Metrics: []reports.ReqMetric{
			{
				ChannelID: "channel1",
				ClientIDs: []string{"client1"},
				Name:      "metric_name",
			},
		},
		Config: &reports.MetricConfig{
			From:        "now()-1h",
			To:          "now()",
			Title:       "test_title",
			Aggregation: reports.AggConfig{AggType: reports.AggregationAVG, Interval: "1h"},
		},
		Email: &reports.EmailSetting{
			To:      []string{"test@example.com"},
			Subject: "Test Report",
		},
	}

	cases := []struct {
		desc            string
		cfg             sdk.ReportConfig
		token           string
		session         smqauthn.Session
		svcRes          reports.ReportConfig
		svcErr          error
		authenticateErr error
		wantErr         bool
	}{
		{
			desc:   "add report config successfully",
			cfg:    testReportConfig,
			token:  validToken,
			svcRes: svcCfg,
		},
		{
			desc:    "add report config with empty token",
			cfg:     sdk.ReportConfig{Name: "daily-report"},
			token:   "",
			wantErr: true,
			svcErr:  errors.New("missing or invalid bearer user token"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := rsvc.On("AddReportConfig", mock.Anything, tc.session, mock.Anything).Return(tc.svcRes, tc.svcErr)
			result, err := mgsdk.AddReportConfig(context.Background(), tc.cfg, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.NotEmpty(t, result.ID)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestViewReportConfig(t *testing.T) {
	rs, rsvc, auth := setupReports()
	defer rs.Close()

	conf := sdk.Config{
		ReportsURL: rs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	svcCfg := reports.ReportConfig{
		ID:          reportConfigID,
		Name:        reportName,
		Description: reportDescription,
		DomainID:    domainID,
		Status:      reports.EnabledStatus,
		Metrics:     metrics,
		Config:      &config,
		Email:       &email,
	}

	cases := []struct {
		desc            string
		id              string
		token           string
		session         smqauthn.Session
		svcRes          reports.ReportConfig
		svcErr          error
		authenticateErr error
		wantErr         bool
	}{
		{
			desc:   "view report config successfully",
			id:     reportConfigID,
			token:  validToken,
			svcRes: svcCfg,
		},
		{
			desc:    "view report config with empty token",
			id:      reportConfigID,
			token:   "",
			wantErr: true,
		},
		{
			desc:    "view non-existent report config",
			id:      "non-existent",
			token:   validToken,
			svcErr:  errors.New("not found"),
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := rsvc.On("ViewReportConfig", mock.Anything, tc.session, tc.id, mock.Anything).Return(tc.svcRes, tc.svcErr)
			result, err := mgsdk.ViewReportConfig(context.Background(), tc.id, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.NotEmpty(t, result.ID)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateReportConfig(t *testing.T) {
	rs, rsvc, auth := setupReports()
	defer rs.Close()

	conf := sdk.Config{
		ReportsURL: rs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	updatedConfig := testReportConfig
	updatedConfig.Name = reportUpdatedName
	updatedConfig.Description = reportUpdatedDesc

	svcCfg := reports.ReportConfig{
		ID:          reportConfigID,
		Name:        reportUpdatedName,
		Description: reportUpdatedDesc,
		DomainID:    domainID,
		Status:      reports.EnabledStatus,
		Metrics:     metrics,
		Config:      &config,
		Email:       &email,
	}

	cases := []struct {
		desc            string
		cfg             sdk.ReportConfig
		token           string
		session         smqauthn.Session
		svcRes          reports.ReportConfig
		svcErr          error
		authenticateErr error
		wantErr         bool
	}{
		{
			desc:   "update report config successfully",
			cfg:    updatedConfig,
			token:  validToken,
			svcRes: svcCfg,
		},
		{
			desc:    "update report config with empty token",
			cfg:     sdk.ReportConfig{ID: reportConfigID, Name: "updated-report"},
			token:   "",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := rsvc.On("UpdateReportConfig", mock.Anything, tc.session, mock.Anything).Return(tc.svcRes, tc.svcErr)
			result, err := mgsdk.UpdateReportConfig(context.Background(), tc.cfg, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.NotEmpty(t, result.ID)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateReportSchedule(t *testing.T) {
	rs, rsvc, auth := setupReports()
	defer rs.Close()

	conf := sdk.Config{
		ReportsURL: rs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	svcCfg := reports.ReportConfig{
		ID:     reportConfigID,
		Name:   reportName,
		Status: reports.EnabledStatus,
	}

	cases := []struct {
		desc            string
		cfg             sdk.ReportConfig
		token           string
		session         smqauthn.Session
		svcRes          reports.ReportConfig
		svcErr          error
		authenticateErr error
		wantErr         bool
	}{
		{
			desc:   "update report schedule successfully",
			cfg:    sdk.ReportConfig{ID: reportConfigID, Schedule: map[string]any{"cron": "0 9 * * *"}},
			token:  validToken,
			svcRes: svcCfg,
		},
		{
			desc:    "update report schedule with empty token",
			cfg:     sdk.ReportConfig{ID: reportConfigID},
			token:   "",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := rsvc.On("UpdateReportSchedule", mock.Anything, tc.session, mock.Anything).Return(tc.svcRes, tc.svcErr)
			result, err := mgsdk.UpdateReportSchedule(context.Background(), tc.cfg, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.NotEmpty(t, result.ID)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveReportConfig(t *testing.T) {
	rs, rsvc, auth := setupReports()
	defer rs.Close()

	conf := sdk.Config{
		ReportsURL: rs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		id              string
		token           string
		session         smqauthn.Session
		svcErr          error
		authenticateErr error
		wantErr         bool
	}{
		{
			desc:  "remove report config successfully",
			id:    reportConfigID,
			token: validToken,
		},
		{
			desc:    "remove report config with empty token",
			id:      reportConfigID,
			token:   "",
			wantErr: true,
		},
		{
			desc:    "remove non-existent report config",
			id:      "non-existent",
			token:   validToken,
			svcErr:  errors.New("not found"),
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := rsvc.On("RemoveReportConfig", mock.Anything, tc.session, tc.id).Return(tc.svcErr)
			err := mgsdk.RemoveReportConfig(context.Background(), tc.id, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListReportsConfig(t *testing.T) {
	rs, rsvc, auth := setupReports()
	defer rs.Close()

	conf := sdk.Config{
		ReportsURL: rs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	svcPage := reports.ReportConfigPage{}

	cases := []struct {
		desc            string
		pm              sdk.PageMetadata
		token           string
		session         smqauthn.Session
		svcRes          reports.ReportConfigPage
		svcErr          error
		authenticateErr error
		wantErr         bool
	}{
		{
			desc:   "list reports config successfully",
			pm:     sdk.PageMetadata{Offset: 0, Limit: 10},
			token:  validToken,
			svcRes: svcPage,
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
			token:  validToken,
			svcRes: svcPage,
		},
		{
			desc:   "list reports config with empty metadata excludes filter params",
			pm:     sdk.PageMetadata{},
			token:  validToken,
			svcRes: reports.ReportConfigPage{},
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
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := rsvc.On("ListReportsConfig", mock.Anything, tc.session, mock.Anything).Return(tc.svcRes, tc.svcErr)
			result, err := mgsdk.ListReportsConfig(context.Background(), tc.pm, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.NotNil(t, result)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestEnableReportConfig(t *testing.T) {
	rs, rsvc, auth := setupReports()
	defer rs.Close()

	conf := sdk.Config{
		ReportsURL: rs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	svcCfg := reports.ReportConfig{
		ID:     reportConfigID,
		Status: reports.EnabledStatus,
	}

	cases := []struct {
		desc            string
		id              string
		token           string
		session         smqauthn.Session
		svcRes          reports.ReportConfig
		svcErr          error
		authenticateErr error
		wantErr         bool
	}{
		{
			desc:   "enable report config successfully",
			id:     reportConfigID,
			token:  validToken,
			svcRes: svcCfg,
		},
		{
			desc:    "enable report config with empty token",
			id:      reportConfigID,
			token:   "",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := rsvc.On("EnableReportConfig", mock.Anything, tc.session, tc.id).Return(tc.svcRes, tc.svcErr)
			result, err := mgsdk.EnableReportConfig(context.Background(), tc.id, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.NotEmpty(t, result.ID)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDisableReportConfig(t *testing.T) {
	rs, rsvc, auth := setupReports()
	defer rs.Close()

	conf := sdk.Config{
		ReportsURL: rs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	svcCfg := reports.ReportConfig{
		ID:     reportConfigID,
		Status: reports.DisabledStatus,
	}

	cases := []struct {
		desc            string
		id              string
		token           string
		session         smqauthn.Session
		svcRes          reports.ReportConfig
		svcErr          error
		authenticateErr error
		wantErr         bool
	}{
		{
			desc:   "disable report config successfully",
			id:     reportConfigID,
			token:  validToken,
			svcRes: svcCfg,
		},
		{
			desc:    "disable report config with empty token",
			id:      reportConfigID,
			token:   "",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := rsvc.On("DisableReportConfig", mock.Anything, tc.session, tc.id).Return(tc.svcRes, tc.svcErr)
			result, err := mgsdk.DisableReportConfig(context.Background(), tc.id, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.NotEmpty(t, result.ID)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateReportTemplate(t *testing.T) {
	rs, rsvc, auth := setupReports()
	defer rs.Close()

	conf := sdk.Config{
		ReportsURL: rs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		cfg             sdk.ReportConfig
		token           string
		session         smqauthn.Session
		svcErr          error
		authenticateErr error
		wantErr         bool
	}{
		{
			desc: "update report template successfully",
			cfg: sdk.ReportConfig{
				ID:             reportConfigID,
				ReportTemplate: validTemplate,
			},
			token: validToken,
		},
		{
			desc:    "update report template with empty token",
			cfg:     sdk.ReportConfig{ID: reportConfigID},
			token:   "",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := rsvc.On("UpdateReportTemplate", mock.Anything, tc.session, mock.Anything).Return(tc.svcErr)
			err := mgsdk.UpdateReportTemplate(context.Background(), tc.cfg, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestViewReportTemplate(t *testing.T) {
	rs, rsvc, auth := setupReports()
	defer rs.Close()

	conf := sdk.Config{
		ReportsURL: rs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	svcTmpl := reports.ReportTemplate(validTemplate)

	cases := []struct {
		desc            string
		id              string
		token           string
		session         smqauthn.Session
		svcRes          reports.ReportTemplate
		svcErr          error
		authenticateErr error
		wantErr         bool
	}{
		{
			desc:   "view report template successfully",
			id:     reportConfigID,
			token:  validToken,
			svcRes: svcTmpl,
		},
		{
			desc:    "view report template with empty token",
			id:      reportConfigID,
			token:   "",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := rsvc.On("ViewReportTemplate", mock.Anything, tc.session, tc.id).Return(tc.svcRes, tc.svcErr)
			result, err := mgsdk.ViewReportTemplate(context.Background(), tc.id, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.NotEmpty(t, result)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDeleteReportTemplate(t *testing.T) {
	rs, rsvc, auth := setupReports()
	defer rs.Close()

	conf := sdk.Config{
		ReportsURL: rs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		id              string
		token           string
		session         smqauthn.Session
		svcErr          error
		authenticateErr error
		wantErr         bool
	}{
		{
			desc:  "delete report template successfully",
			id:    reportConfigID,
			token: validToken,
		},
		{
			desc:    "delete report template with empty token",
			id:      reportConfigID,
			token:   "",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := rsvc.On("DeleteReportTemplate", mock.Anything, tc.session, tc.id).Return(tc.svcErr)
			err := mgsdk.DeleteReportTemplate(context.Background(), tc.id, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestGenerateReport(t *testing.T) {
	rs, rsvc, auth := setupReports()
	defer rs.Close()

	conf := sdk.Config{
		ReportsURL: rs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	svcPage := reports.ReportPage{}

	config := sdk.ReportConfig{
		ID:             reportConfigID,
		Name:           reportName,
		Description:    reportDescription,
		DomainID:       domainID,
		Metrics:        metrics,
		Config:         &config,
		ReportTemplate: reports.ReportTemplate(validTemplate),
	}

	cases := []struct {
		desc            string
		cfg             sdk.ReportConfig
		action          sdk.ReportAction
		token           string
		session         smqauthn.Session
		svcRes          reports.ReportPage
		svcErr          error
		authenticateErr error
		wantErr         bool
	}{
		{
			desc:   "generate report successfully",
			cfg:    config,
			action: sdk.ViewReportAction,
			token:  validToken,
			svcRes: svcPage,
		},
		{
			desc:   "generate report with download action",
			cfg:    config,
			action: sdk.DownloadReportAction,
			token:  validToken,
			svcRes: svcPage,
		},
		{
			desc:    "generate report with empty token",
			cfg:     config,
			action:  sdk.ViewReportAction,
			token:   "",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{
					DomainUserID: domainID + "_" + validID,
					UserID:       validID,
					DomainID:     domainID,
				}
			}

			authCall := auth.On(
				"Authenticate",
				mock.Anything,
				tc.token,
			).Return(tc.session, tc.authenticateErr)

			svcCall := rsvc.On(
				"GenerateReport",
				mock.Anything,
				tc.session,
				mock.Anything,
				mock.Anything,
			).Return(tc.svcRes, tc.svcErr)

			page, file, err := mgsdk.GenerateReport(
				context.Background(),
				tc.cfg,
				tc.action,
				domainID,
				tc.token,
			)

			assert.Equal(t, tc.wantErr, err != nil)

			if !tc.wantErr {
				if tc.action == sdk.DownloadReportAction {
					// download should return file
					assert.NotNil(t, file)
				} else {
					// view/email should return page
					assert.Equal(t, tc.svcRes.Total, page.Total)
				}
			}

			svcCall.Unset()
			authCall.Unset()
		})
	}
}
