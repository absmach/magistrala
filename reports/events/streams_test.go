// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/reports"
	"github.com/absmach/magistrala/reports/events"
	"github.com/absmach/magistrala/reports/mocks"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	storeClient  *redis.Client
	storeURL     string
	validSession = authn.Session{
		DomainID: testsutil.GenerateUUID(&testing.T{}),
		UserID:   testsutil.GenerateUUID(&testing.T{}),
	}
	validReportConfig     = generateTestReportConfig(&testing.T{})
	validReportConfigPage = reports.ReportConfigPage{
		PageMeta: reports.PageMeta{
			Limit:  10,
			Offset: 0,
			Total:  1,
		},
		ReportConfigs: []reports.ReportConfig{validReportConfig},
	}
)

func newEventStoreMiddleware(t *testing.T) (*mocks.Service, reports.Service) {
	svc := new(mocks.Service)
	nsvc, err := events.NewEventStoreMiddleware(context.Background(), svc, storeURL)
	require.Nil(t, err, fmt.Sprintf("create events store middleware failed with unexpected error: %s", err))

	return svc, nsvc
}

func TestMain(m *testing.M) {
	code := testsutil.RunRedisTest(m, &storeClient, &storeURL)
	os.Exit(code)
}

func TestAddReportConfig(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc    string
		session authn.Session
		cfg     reports.ReportConfig
		svcRes  reports.ReportConfig
		svcErr  error
		resp    reports.ReportConfig
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			cfg:     validReportConfig,
			svcRes:  validReportConfig,
			svcErr:  nil,
			resp:    validReportConfig,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			cfg:     validReportConfig,
			svcRes:  reports.ReportConfig{},
			svcErr:  svcerr.ErrCreateEntity,
			resp:    reports.ReportConfig{},
			err:     svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("AddReportConfig", validCtx, tc.session, tc.cfg).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.AddReportConfig(validCtx, tc.session, tc.cfg)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestRemoveReportConfig(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		reportID string
		svcErr   error
		err      error
	}{
		{
			desc:     "publish successfully",
			session:  validSession,
			reportID: validReportConfig.ID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:     "failed to publish with service error",
			session:  validSession,
			reportID: validReportConfig.ID,
			svcErr:   svcerr.ErrRemoveEntity,
			err:      svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("RemoveReportConfig", validCtx, tc.session, tc.reportID).Return(tc.svcErr)
			err := nsvc.RemoveReportConfig(validCtx, tc.session, tc.reportID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestViewReportConfig(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc      string
		session   authn.Session
		reportID  string
		withRoles bool
		svcRes    reports.ReportConfig
		svcErr    error
		resp      reports.ReportConfig
		err       error
	}{
		{
			desc:      "view successfully",
			session:   validSession,
			reportID:  validReportConfig.ID,
			withRoles: false,
			svcRes:    validReportConfig,
			svcErr:    nil,
			resp:      validReportConfig,
			err:       nil,
		},
		{
			desc:      "failed with service error",
			session:   validSession,
			reportID:  validReportConfig.ID,
			withRoles: false,
			svcRes:    reports.ReportConfig{},
			svcErr:    svcerr.ErrViewEntity,
			resp:      reports.ReportConfig{},
			err:       svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ViewReportConfig", validCtx, tc.session, tc.reportID, tc.withRoles).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.ViewReportConfig(validCtx, tc.session, tc.reportID, tc.withRoles)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestUpdateReportConfig(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	updatedCfg := validReportConfig
	updatedCfg.Name = "updatedName"

	cases := []struct {
		desc    string
		session authn.Session
		cfg     reports.ReportConfig
		svcRes  reports.ReportConfig
		svcErr  error
		resp    reports.ReportConfig
		err     error
	}{
		{
			desc:    "update successfully",
			session: validSession,
			cfg:     updatedCfg,
			svcRes:  updatedCfg,
			svcErr:  nil,
			resp:    updatedCfg,
			err:     nil,
		},
		{
			desc:    "failed with service error",
			session: validSession,
			cfg:     updatedCfg,
			svcRes:  reports.ReportConfig{},
			svcErr:  svcerr.ErrUpdateEntity,
			resp:    reports.ReportConfig{},
			err:     svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UpdateReportConfig", validCtx, tc.session, tc.cfg).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.UpdateReportConfig(validCtx, tc.session, tc.cfg)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestUpdateReportSchedule(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc    string
		session authn.Session
		cfg     reports.ReportConfig
		svcRes  reports.ReportConfig
		svcErr  error
		resp    reports.ReportConfig
		err     error
	}{
		{
			desc:    "update schedule successfully",
			session: validSession,
			cfg:     validReportConfig,
			svcRes:  validReportConfig,
			svcErr:  nil,
			resp:    validReportConfig,
			err:     nil,
		},
		{
			desc:    "failed with service error",
			session: validSession,
			cfg:     validReportConfig,
			svcRes:  reports.ReportConfig{},
			svcErr:  svcerr.ErrUpdateEntity,
			resp:    reports.ReportConfig{},
			err:     svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UpdateReportSchedule", validCtx, tc.session, tc.cfg).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.UpdateReportSchedule(validCtx, tc.session, tc.cfg)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestListReportsConfig(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		pageMeta reports.PageMeta
		svcRes   reports.ReportConfigPage
		svcErr   error
		resp     reports.ReportConfigPage
		err      error
	}{
		{
			desc:    "list successfully",
			session: validSession,
			pageMeta: reports.PageMeta{
				Limit:  10,
				Offset: 0,
			},
			svcRes: validReportConfigPage,
			svcErr: nil,
			resp:   validReportConfigPage,
			err:    nil,
		},
		{
			desc:    "failed with service error",
			session: validSession,
			pageMeta: reports.PageMeta{
				Limit:  10,
				Offset: 0,
			},
			svcRes: reports.ReportConfigPage{},
			svcErr: svcerr.ErrViewEntity,
			resp:   reports.ReportConfigPage{},
			err:    svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ListReportsConfig", validCtx, tc.session, tc.pageMeta).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.ListReportsConfig(validCtx, tc.session, tc.pageMeta)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestEnableReportConfig(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		reportID string
		svcRes   reports.ReportConfig
		svcErr   error
		resp     reports.ReportConfig
		err      error
	}{
		{
			desc:     "enable successfully",
			session:  validSession,
			reportID: validReportConfig.ID,
			svcRes:   validReportConfig,
			svcErr:   nil,
			resp:     validReportConfig,
			err:      nil,
		},
		{
			desc:     "failed with service error",
			session:  validSession,
			reportID: validReportConfig.ID,
			svcRes:   reports.ReportConfig{},
			svcErr:   svcerr.ErrUpdateEntity,
			resp:     reports.ReportConfig{},
			err:      svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("EnableReportConfig", validCtx, tc.session, tc.reportID).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.EnableReportConfig(validCtx, tc.session, tc.reportID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestDisableReportConfig(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		reportID string
		svcRes   reports.ReportConfig
		svcErr   error
		resp     reports.ReportConfig
		err      error
	}{
		{
			desc:     "disable successfully",
			session:  validSession,
			reportID: validReportConfig.ID,
			svcRes:   validReportConfig,
			svcErr:   nil,
			resp:     validReportConfig,
			err:      nil,
		},
		{
			desc:     "failed with service error",
			session:  validSession,
			reportID: validReportConfig.ID,
			svcRes:   reports.ReportConfig{},
			svcErr:   svcerr.ErrUpdateEntity,
			resp:     reports.ReportConfig{},
			err:      svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("DisableReportConfig", validCtx, tc.session, tc.reportID).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.DisableReportConfig(validCtx, tc.session, tc.reportID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestUpdateReportTemplate(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc    string
		session authn.Session
		cfg     reports.ReportConfig
		svcErr  error
		err     error
	}{
		{
			desc:    "update template successfully",
			session: validSession,
			cfg:     validReportConfig,
			svcErr:  nil,
			err:     nil,
		},
		{
			desc:    "failed with service error",
			session: validSession,
			cfg:     validReportConfig,
			svcErr:  svcerr.ErrUpdateEntity,
			err:     svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UpdateReportTemplate", validCtx, tc.session, tc.cfg).Return(tc.svcErr)
			err := nsvc.UpdateReportTemplate(validCtx, tc.session, tc.cfg)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestViewReportTemplate(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		reportID string
		svcRes   reports.ReportTemplate
		svcErr   error
		resp     reports.ReportTemplate
		err      error
	}{
		{
			desc:     "view template successfully",
			session:  validSession,
			reportID: validReportConfig.ID,
			svcRes:   reports.ReportTemplate("template content"),
			svcErr:   nil,
			resp:     reports.ReportTemplate("template content"),
			err:      nil,
		},
		{
			desc:     "failed with service error",
			session:  validSession,
			reportID: validReportConfig.ID,
			svcRes:   reports.ReportTemplate(""),
			svcErr:   svcerr.ErrViewEntity,
			resp:     reports.ReportTemplate(""),
			err:      svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ViewReportTemplate", validCtx, tc.session, tc.reportID).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.ViewReportTemplate(validCtx, tc.session, tc.reportID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestDeleteReportTemplate(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		reportID string
		svcErr   error
		err      error
	}{
		{
			desc:     "delete template successfully",
			session:  validSession,
			reportID: validReportConfig.ID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:     "failed with service error",
			session:  validSession,
			reportID: validReportConfig.ID,
			svcErr:   svcerr.ErrRemoveEntity,
			err:      svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("DeleteReportTemplate", validCtx, tc.session, tc.reportID).Return(tc.svcErr)
			err := nsvc.DeleteReportTemplate(validCtx, tc.session, tc.reportID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestGenerateReport(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc    string
		session authn.Session
		config  reports.ReportConfig
		action  reports.ReportAction
		svcRes  reports.ReportPage
		svcErr  error
		resp    reports.ReportPage
		err     error
	}{
		{
			desc:    "generate report successfully",
			session: validSession,
			config:  validReportConfig,
			action:  reports.ViewReport,
			svcRes:  reports.ReportPage{},
			svcErr:  nil,
			resp:    reports.ReportPage{},
			err:     nil,
		},
		{
			desc:    "failed with service error",
			session: validSession,
			config:  validReportConfig,
			action:  reports.ViewReport,
			svcRes:  reports.ReportPage{},
			svcErr:  svcerr.ErrViewEntity,
			resp:    reports.ReportPage{},
			err:     svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("GenerateReport", validCtx, tc.session, tc.config, tc.action).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.GenerateReport(validCtx, tc.session, tc.config, tc.action)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestStartScheduler(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	cases := []struct {
		desc   string
		svcErr error
		err    error
	}{
		{
			desc:   "start scheduler successfully",
			svcErr: nil,
			err:    nil,
		},
		{
			desc:   "failed with service error",
			svcErr: svcerr.ErrCreateEntity,
			err:    svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("StartScheduler", context.Background()).Return(tc.svcErr)
			err := nsvc.StartScheduler(context.Background())
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func generateTestReportConfig(t *testing.T) reports.ReportConfig {
	createdAt, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	assert.Nil(t, err, fmt.Sprintf("Unexpected error parsing time: %v", err))
	return reports.ReportConfig{
		ID:        testsutil.GenerateUUID(t),
		Name:      "testreport",
		DomainID:  testsutil.GenerateUUID(t),
		Status:    reports.EnabledStatus,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
}
