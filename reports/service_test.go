// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package reports_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/0x6flab/namegenerator"
	"github.com/absmach/magistrala/internal/testsutil"
	pkglog "github.com/absmach/magistrala/pkg/logger"
	pkgSch "github.com/absmach/magistrala/pkg/schedule"
	remocks "github.com/absmach/magistrala/re/mocks"
	readmocks "github.com/absmach/magistrala/readers/mocks"
	"github.com/absmach/magistrala/reports"
	"github.com/absmach/magistrala/reports/mocks"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	namegen  = namegenerator.NewGenerator()
	userID   = testsutil.GenerateUUID(&testing.T{})
	domainID = testsutil.GenerateUUID(&testing.T{})
	now      = time.Now().UTC()
	schedule = pkgSch.Schedule{
		StartDateTime:   &now,
		Recurring:       pkgSch.Daily,
		RecurringPeriod: 1,
		Time:            time.Now().Add(-time.Hour),
	}
	reportName = namegen.Generate()
	rptConfig  = reports.ReportConfig{
		ID:        testsutil.GenerateUUID(&testing.T{}),
		Name:      reportName,
		DomainID:  domainID,
		Status:    reports.EnabledStatus,
		Schedule:  schedule,
		CreatedBy: userID,
		UpdatedBy: userID,
		UpdatedAt: time.Now(),
	}
)

func newService(runInfo chan pkglog.RunInfo) (reports.Service, *mocks.Repository, *remocks.Ticker) {
	repo := new(mocks.Repository)
	mockTicker := new(remocks.Ticker)
	idProvider := uuid.NewMock()
	readersSvc := new(readmocks.ReadersServiceClient)
	e := new(remocks.Emailer)
	return reports.NewService(repo, runInfo, idProvider, mockTicker, e, readersSvc), repo, mockTicker
}

func TestAddReportConfig(t *testing.T) {
	svc, repo, _ := newService(make(chan pkglog.RunInfo))

	cases := []struct {
		desc    string
		session authn.Session
		cfg     reports.ReportConfig
		res     reports.ReportConfig
		err     error
	}{
		{
			desc: "Add report config successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			cfg: reports.ReportConfig{
				Name:     reportName,
				Schedule: schedule,
			},
			res: rptConfig,
			err: nil,
		},
		{
			desc: "Add report config with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			cfg: reports.ReportConfig{
				Name:     reportName,
				Schedule: schedule,
			},
			err: repoerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("AddReportConfig", mock.Anything, mock.Anything).Return(tc.res, tc.err)
			res, err := svc.AddReportConfig(context.Background(), tc.session, tc.cfg)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.NotEmpty(t, res.ID, "expected non-empty result in ID")
				assert.Equal(t, tc.cfg.Name, res.Name)
				assert.Equal(t, tc.cfg.Schedule, res.Schedule)
			}
			defer repoCall.Unset()
		})
	}
}

func TestViewReportConfig(t *testing.T) {
	svc, repo, _ := newService(make(chan pkglog.RunInfo))

	cases := []struct {
		desc    string
		session authn.Session
		id      string
		res     reports.ReportConfig
		err     error
	}{
		{
			desc: "view report config successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:  rptConfig.ID,
			res: rptConfig,
			err: nil,
		},
		{
			desc: "view report config with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:  rptConfig.ID,
			err: svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("ViewReportConfig", mock.Anything, mock.Anything).Return(tc.res, tc.err)
			res, err := svc.ViewReportConfig(context.Background(), tc.session, tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.res, res)
			}
			defer repoCall.Unset()
		})
	}
}

func TestUpdateReportConfig(t *testing.T) {
	svc, repo, _ := newService(make(chan pkglog.RunInfo))

	newName := namegen.Generate()
	now := time.Now().Add(time.Hour)
	cases := []struct {
		desc    string
		session authn.Session
		cfg     reports.ReportConfig
		res     reports.ReportConfig
		err     error
	}{
		{
			desc: "update report config successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			cfg: reports.ReportConfig{
				Name:     newName,
				ID:       rptConfig.ID,
				Schedule: schedule,
			},
			res: reports.ReportConfig{
				Name:      newName,
				ID:        rptConfig.ID,
				DomainID:  rptConfig.DomainID,
				Status:    rptConfig.Status,
				Schedule:  rptConfig.Schedule,
				UpdatedAt: now,
				UpdatedBy: userID,
			},
			err: nil,
		},
		{
			desc: "update report config with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			cfg: reports.ReportConfig{
				Name:     rptConfig.Name,
				ID:       rptConfig.ID,
				Schedule: schedule,
			},
			err: svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("UpdateReportConfig", mock.Anything, mock.Anything).Return(tc.res, tc.err)
			res, err := svc.UpdateReportConfig(context.Background(), tc.session, tc.cfg)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.res, res)
			}
			defer repoCall.Unset()
		})
	}
}

func TestListReportsConfig(t *testing.T) {
	svc, repo, _ := newService(make(chan pkglog.RunInfo))
	numConfigs := 50
	now := time.Now().Add(time.Hour)
	var configs []reports.ReportConfig
	for i := 0; i < numConfigs; i++ {
		c := reports.ReportConfig{
			ID:        testsutil.GenerateUUID(t),
			Name:      namegen.Generate(),
			DomainID:  domainID,
			Status:    reports.EnabledStatus,
			CreatedAt: now,
			CreatedBy: userID,
			Schedule:  schedule,
		}
		configs = append(configs, c)
	}

	cases := []struct {
		desc     string
		session  authn.Session
		pageMeta reports.PageMeta
		res      reports.ReportConfigPage
		err      error
	}{
		{
			desc: "list report configs successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			pageMeta: reports.PageMeta{},
			res: reports.ReportConfigPage{
				PageMeta: reports.PageMeta{
					Total:  uint64(numConfigs),
					Offset: 0,
					Limit:  10,
				},
				ReportConfigs: configs[0:10],
			},
			err: nil,
		},
		{
			desc: "list report configs successfully with limit",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			pageMeta: reports.PageMeta{
				Limit: 100,
			},
			res: reports.ReportConfigPage{
				PageMeta: reports.PageMeta{
					Total:  uint64(numConfigs),
					Offset: 0,
					Limit:  100,
				},
				ReportConfigs: configs[0:numConfigs],
			},
			err: nil,
		},
		{
			desc: "list report configs successfully with offset",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			pageMeta: reports.PageMeta{
				Offset: 20,
				Limit:  10,
			},
			res: reports.ReportConfigPage{
				PageMeta: reports.PageMeta{
					Total:  uint64(numConfigs),
					Offset: 20,
					Limit:  10,
				},
				ReportConfigs: configs[20:30],
			},
			err: nil,
		},
		{
			desc: "list report configs with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			pageMeta: reports.PageMeta{},
			err:      svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("ListReportsConfig", mock.Anything, mock.Anything).Return(tc.res, tc.err)
			res, err := svc.ListReportsConfig(context.Background(), tc.session, tc.pageMeta)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.res, res)
			}
			defer repoCall.Unset()
		})
	}
}

func TestRemoveReportConfig(t *testing.T) {
	svc, repo, _ := newService(make(chan pkglog.RunInfo))

	cases := []struct {
		desc    string
		session authn.Session
		id      string
		err     error
	}{
		{
			desc: "remove report config successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:  rptConfig.ID,
			err: nil,
		},
		{
			desc: "remove report config with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:  rptConfig.ID,
			err: svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("RemoveReportConfig", mock.Anything, mock.Anything).Return(tc.err)
			err := svc.RemoveReportConfig(context.Background(), tc.session, tc.id)

			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			defer repoCall.Unset()
		})
	}
}

func TestEnableReportConfig(t *testing.T) {
	svc, repo, _ := newService(make(chan pkglog.RunInfo))

	cases := []struct {
		desc    string
		session authn.Session
		id      string
		status  reports.Status
		res     reports.ReportConfig
		err     error
	}{
		{
			desc: "enable report config successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:     rptConfig.ID,
			status: reports.EnabledStatus,
			res:    rptConfig,
			err:    nil,
		},
		{
			desc: "enable report config with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:     rptConfig.ID,
			status: reports.EnabledStatus,
			err:    svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("UpdateReportConfigStatus", context.Background(), mock.Anything).Return(tc.res, tc.err)
			res, err := svc.EnableReportConfig(context.Background(), tc.session, tc.id)

			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.res, res)
			}
			defer repoCall.Unset()
		})
	}
}

func TestDisableReportConfig(t *testing.T) {
	svc, repo, _ := newService(make(chan pkglog.RunInfo))

	cases := []struct {
		desc    string
		session authn.Session
		id      string
		status  reports.Status
		res     reports.ReportConfig
		err     error
	}{
		{
			desc: "disable report config successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:     rptConfig.ID,
			status: reports.DisabledStatus,
			res: reports.ReportConfig{
				ID:        rptConfig.ID,
				Name:      rptConfig.Name,
				DomainID:  rptConfig.DomainID,
				Status:    reports.DisabledStatus,
				Schedule:  schedule,
				UpdatedBy: userID,
				UpdatedAt: time.Now(),
			},
			err: nil,
		},
		{
			desc: "disable report config with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:     rptConfig.ID,
			status: reports.DisabledStatus,
			err:    svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("UpdateReportConfigStatus", mock.Anything, mock.Anything).Return(tc.res, tc.err)
			res, err := svc.DisableReportConfig(context.Background(), tc.session, tc.id)

			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.res, res)
			}
			defer repoCall.Unset()
		})
	}
}
