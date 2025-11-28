// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/0x6flab/namegenerator"
	"github.com/absmach/magistrala/pkg/schedule"
	"github.com/absmach/magistrala/reports"
	"github.com/absmach/magistrala/reports/postgres"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	namegen    = namegenerator.NewGenerator()
	idProvider = uuid.New()
)

func generateUUID(t *testing.T) string {
	id, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("generate uuid unexpected error: %s", err))
	return id
}

func TestAddReportConfig(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM report_config")
		require.Nil(t, err, fmt.Sprintf("clean report_config unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	reportConfig := reports.ReportConfig{
		ID:          generateUUID(t),
		Name:        namegen.Generate(),
		Description: namegen.Generate(),
		DomainID:    generateUUID(t),
		Config: &reports.MetricConfig{
			From:  "now-1h",
			To:    "now",
			Title: "Test Report",
		},
		Metrics: []reports.ReqMetric{
			{
				ChannelID: generateUUID(t),
				Name:      "temperature",
			},
		},
		Email: &reports.EmailSetting{
			To:      []string{"test@example.com"},
			Subject: "Test Report",
			Content: "Report content",
		},
		Schedule: schedule.Schedule{
			StartDateTime:   time.Now().UTC(),
			Time:            time.Now().UTC().Add(time.Hour),
			Recurring:       schedule.Daily,
			RecurringPeriod: 1,
		},
		Status:    reports.EnabledStatus,
		CreatedAt: time.Now().UTC(),
		CreatedBy: generateUUID(t),
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: generateUUID(t),
	}

	cases := []struct {
		desc   string
		report reports.ReportConfig
		err    error
	}{
		{
			desc:   "add valid report config",
			report: reportConfig,
			err:    nil,
		},
		{
			desc:   "add duplicate report config",
			report: reportConfig,
			err:    repoerr.ErrConflict,
		},
		{
			desc: "add report config with empty ID",
			report: reports.ReportConfig{
				Name:      namegen.Generate(),
				DomainID:  generateUUID(t),
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
			err: repoerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			rpt, err := repo.AddReportConfig(context.Background(), tc.report)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			require.NotEmpty(t, rpt.ID)
			require.Equal(t, tc.report.Name, rpt.Name)
			require.Equal(t, tc.report.DomainID, rpt.DomainID)
			require.Equal(t, tc.report.Status, rpt.Status)
		})
	}
}

func TestViewReportConfig(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM report_config")
		require.Nil(t, err, fmt.Sprintf("clean report_config unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	reportConfig := reports.ReportConfig{
		ID:          generateUUID(t),
		Name:        namegen.Generate(),
		Description: namegen.Generate(),
		DomainID:    generateUUID(t),
		Config: &reports.MetricConfig{
			From:  "now-1h",
			To:    "now",
			Title: "Test Report",
		},
		Metrics: []reports.ReqMetric{
			{
				ChannelID: generateUUID(t),
				Name:      "temperature",
			},
		},
		Email: &reports.EmailSetting{
			To:      []string{"test@example.com"},
			Subject: "Test Report",
		},
		Status:    reports.EnabledStatus,
		CreatedAt: time.Now().UTC(),
		CreatedBy: generateUUID(t),
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: generateUUID(t),
	}

	saved, err := repo.AddReportConfig(context.Background(), reportConfig)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "view existing report config",
			id:   saved.ID,
			err:  nil,
		},
		{
			desc: "view non-existing report config",
			id:   generateUUID(t),
			err:  repoerr.ErrNotFound,
		},
		{
			desc: "view with empty id",
			id:   "",
			err:  repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			rpt, err := repo.ViewReportConfig(context.Background(), tc.id)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			require.Equal(t, saved.ID, rpt.ID)
			require.Equal(t, saved.Name, rpt.Name)
			require.Equal(t, saved.DomainID, rpt.DomainID)
		})
	}
}

func TestUpdateReportConfig(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM report_config")
		require.Nil(t, err, fmt.Sprintf("clean report_config unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	reportConfig := reports.ReportConfig{
		ID:          generateUUID(t),
		Name:        namegen.Generate(),
		Description: namegen.Generate(),
		DomainID:    generateUUID(t),
		Status:      reports.EnabledStatus,
		CreatedAt:   time.Now().UTC(),
		CreatedBy:   generateUUID(t),
		UpdatedAt:   time.Now().UTC(),
		UpdatedBy:   generateUUID(t),
		Metrics: []reports.ReqMetric{
			{
				ChannelID: generateUUID(t),
				Name:      "temperature",
			},
		},
	}

	saved, err := repo.AddReportConfig(context.Background(), reportConfig)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc   string
		report reports.ReportConfig
		err    error
	}{
		{
			desc: "update report name",
			report: reports.ReportConfig{
				ID:        saved.ID,
				Name:      "Updated Name",
				UpdatedAt: time.Now().UTC(),
				UpdatedBy: generateUUID(t),
			},
			err: nil,
		},
		{
			desc: "update report description",
			report: reports.ReportConfig{
				ID:          saved.ID,
				Description: "Updated Description",
				UpdatedAt:   time.Now().UTC(),
				UpdatedBy:   generateUUID(t),
			},
			err: nil,
		},
		{
			desc: "update non-existing report",
			report: reports.ReportConfig{
				ID:        generateUUID(t),
				Name:      "New Name",
				UpdatedAt: time.Now().UTC(),
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			rpt, err := repo.UpdateReportConfig(context.Background(), tc.report)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			require.Equal(t, tc.report.ID, rpt.ID)
			if tc.report.Name != "" {
				require.Equal(t, tc.report.Name, rpt.Name)
			}
			if tc.report.Description != "" {
				require.Equal(t, tc.report.Description, rpt.Description)
			}
		})
	}
}

func TestUpdateReportConfigStatus(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM report_config")
		require.Nil(t, err, fmt.Sprintf("clean report_config unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	reportConfig := reports.ReportConfig{
		ID:        generateUUID(t),
		Name:      namegen.Generate(),
		DomainID:  generateUUID(t),
		Status:    reports.EnabledStatus,
		CreatedAt: time.Now().UTC(),
		CreatedBy: generateUUID(t),
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: generateUUID(t),
	}

	saved, err := repo.AddReportConfig(context.Background(), reportConfig)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc   string
		report reports.ReportConfig
		err    error
	}{
		{
			desc: "disable report",
			report: reports.ReportConfig{
				ID:        saved.ID,
				Status:    reports.DisabledStatus,
				UpdatedAt: time.Now().UTC(),
				UpdatedBy: generateUUID(t),
			},
			err: nil,
		},
		{
			desc: "enable report",
			report: reports.ReportConfig{
				ID:        saved.ID,
				Status:    reports.EnabledStatus,
				UpdatedAt: time.Now().UTC(),
				UpdatedBy: generateUUID(t),
			},
			err: nil,
		},
		{
			desc: "update status of non-existing report",
			report: reports.ReportConfig{
				ID:        generateUUID(t),
				Status:    reports.DisabledStatus,
				UpdatedAt: time.Now().UTC(),
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			rpt, err := repo.UpdateReportConfigStatus(context.Background(), tc.report)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			require.Equal(t, tc.report.Status, rpt.Status)
		})
	}
}

func TestRemoveReportConfig(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM report_config")
		require.Nil(t, err, fmt.Sprintf("clean report_config unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	reportConfig := reports.ReportConfig{
		ID:        generateUUID(t),
		Name:      namegen.Generate(),
		DomainID:  generateUUID(t),
		Status:    reports.EnabledStatus,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	saved, err := repo.AddReportConfig(context.Background(), reportConfig)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "remove existing report",
			id:   saved.ID,
			err:  nil,
		},
		{
			desc: "remove non-existing report",
			id:   generateUUID(t),
			err:  nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.RemoveReportConfig(context.Background(), tc.id)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		})
	}
}

func TestListReportsConfig(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM report_config")
		require.Nil(t, err, fmt.Sprintf("clean report_config unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	domainID := generateUUID(t)

	num := uint64(10)
	for i := uint64(0); i < num; i++ {
		reportConfig := reports.ReportConfig{
			ID:        generateUUID(t),
			Name:      fmt.Sprintf("Report-%d", i),
			DomainID:  domainID,
			Status:    reports.EnabledStatus,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		_, err := repo.AddReportConfig(context.Background(), reportConfig)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}

	cases := []struct {
		desc     string
		pageMeta reports.PageMeta
		size     uint64
		err      error
	}{
		{
			desc: "list all reports",
			pageMeta: reports.PageMeta{
				Domain: domainID,
				Limit:  num,
				Offset: 0,
			},
			size: num,
			err:  nil,
		},
		{
			desc: "list with limit",
			pageMeta: reports.PageMeta{
				Domain: domainID,
				Limit:  5,
				Offset: 0,
			},
			size: 5,
			err:  nil,
		},
		{
			desc: "list with offset",
			pageMeta: reports.PageMeta{
				Domain: domainID,
				Limit:  num,
				Offset: 5,
			},
			size: 5,
			err:  nil,
		},
		{
			desc: "list enabled reports",
			pageMeta: reports.PageMeta{
				Domain: domainID,
				Limit:  num,
				Status: reports.EnabledStatus,
			},
			size: num,
			err:  nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			page, err := repo.ListReportsConfig(context.Background(), tc.pageMeta)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			require.Equal(t, tc.size, uint64(len(page.ReportConfigs)))
		})
	}
}

func TestUpdateReportSchedule(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM report_config")
		require.Nil(t, err, fmt.Sprintf("clean report_config unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	reportConfig := reports.ReportConfig{
		ID:        generateUUID(t),
		Name:      namegen.Generate(),
		DomainID:  generateUUID(t),
		Status:    reports.EnabledStatus,
		CreatedAt: time.Now().UTC(),
		CreatedBy: generateUUID(t),
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: generateUUID(t),
		Metrics: []reports.ReqMetric{
			{
				ChannelID: generateUUID(t),
				Name:      "temperature",
			},
		},
		Schedule: schedule.Schedule{
			StartDateTime:   time.Now().UTC(),
			Time:            time.Now().UTC().Add(time.Hour),
			Recurring:       schedule.Daily,
			RecurringPeriod: 1,
		},
	}

	saved, err := repo.AddReportConfig(context.Background(), reportConfig)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	newSchedule := schedule.Schedule{
		StartDateTime:   time.Now().UTC().Add(24 * time.Hour),
		Time:            time.Now().UTC().Add(25 * time.Hour),
		Recurring:       schedule.Weekly,
		RecurringPeriod: 2,
	}

	cases := []struct {
		desc     string
		report   reports.ReportConfig
		expected schedule.Schedule
		err      error
	}{
		{
			desc: "update schedule",
			report: reports.ReportConfig{
				ID:        saved.ID,
				Schedule:  newSchedule,
				UpdatedAt: time.Now().UTC(),
				UpdatedBy: generateUUID(t),
			},
			expected: newSchedule,
			err:      nil,
		},
		{
			desc: "update schedule of non-existing report",
			report: reports.ReportConfig{
				ID:        generateUUID(t),
				Schedule:  newSchedule,
				UpdatedAt: time.Now().UTC(),
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			rpt, err := repo.UpdateReportSchedule(context.Background(), tc.report)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			require.Equal(t, tc.expected.Recurring, rpt.Schedule.Recurring)
			require.Equal(t, tc.expected.RecurringPeriod, rpt.Schedule.RecurringPeriod)
		})
	}
}

func TestUpdateReportDue(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM report_config")
		require.Nil(t, err, fmt.Sprintf("clean report_config unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	reportConfig := reports.ReportConfig{
		ID:        generateUUID(t),
		Name:      namegen.Generate(),
		DomainID:  generateUUID(t),
		Status:    reports.EnabledStatus,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Metrics: []reports.ReqMetric{
			{
				ChannelID: generateUUID(t),
				Name:      "temperature",
			},
		},
	}

	saved, err := repo.AddReportConfig(context.Background(), reportConfig)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	newDue := time.Now().UTC().Add(24 * time.Hour)

	cases := []struct {
		desc string
		id   string
		due  time.Time
		err  error
	}{
		{
			desc: "update due time",
			id:   saved.ID,
			due:  newDue,
			err:  nil,
		},
		{
			desc: "update due time of non-existing report",
			id:   generateUUID(t),
			due:  newDue,
			err:  repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			rpt, err := repo.UpdateReportDue(context.Background(), tc.id, tc.due)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			require.True(t, tc.due.Equal(rpt.Schedule.Time))
		})
	}
}

func TestUpdateReportTemplate(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM report_config")
		require.Nil(t, err, fmt.Sprintf("clean report_config unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	domainID := generateUUID(t)
	reportConfig := reports.ReportConfig{
		ID:        generateUUID(t),
		Name:      namegen.Generate(),
		DomainID:  domainID,
		Status:    reports.EnabledStatus,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Metrics: []reports.ReqMetric{
			{
				ChannelID: generateUUID(t),
				Name:      "temperature",
			},
		},
	}

	saved, err := repo.AddReportConfig(context.Background(), reportConfig)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	template := reports.ReportTemplate("<html><body>Test Template</body></html>")

	cases := []struct {
		desc     string
		domainID string
		reportID string
		template reports.ReportTemplate
		err      error
	}{
		{
			desc:     "update template",
			domainID: domainID,
			reportID: saved.ID,
			template: template,
			err:      nil,
		},
		{
			desc:     "update template for non-existing report",
			domainID: domainID,
			reportID: generateUUID(t),
			template: template,
			err:      nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.UpdateReportTemplate(context.Background(), tc.domainID, tc.reportID, tc.template)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		})
	}
}

func TestViewReportTemplate(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM report_config")
		require.Nil(t, err, fmt.Sprintf("clean report_config unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	domainID := generateUUID(t)
	template := reports.ReportTemplate("<html><body>Test Template</body></html>")

	reportConfig := reports.ReportConfig{
		ID:             generateUUID(t),
		Name:           namegen.Generate(),
		DomainID:       domainID,
		Status:         reports.EnabledStatus,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
		ReportTemplate: template,
		Metrics: []reports.ReqMetric{
			{
				ChannelID: generateUUID(t),
				Name:      "temperature",
			},
		},
	}

	saved, err := repo.AddReportConfig(context.Background(), reportConfig)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc     string
		domainID string
		reportID string
		expected reports.ReportTemplate
		err      error
	}{
		{
			desc:     "view existing template",
			domainID: domainID,
			reportID: saved.ID,
			expected: template,
			err:      nil,
		},
		{
			desc:     "view template for non-existing report",
			domainID: domainID,
			reportID: generateUUID(t),
			err:      repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			tmpl, err := repo.ViewReportTemplate(context.Background(), tc.domainID, tc.reportID)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			require.Equal(t, tc.expected, tmpl)
		})
	}
}

func TestDeleteReportTemplate(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM report_config")
		require.Nil(t, err, fmt.Sprintf("clean report_config unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	domainID := generateUUID(t)
	template := reports.ReportTemplate("<html><body>Test Template</body></html>")

	reportConfig := reports.ReportConfig{
		ID:             generateUUID(t),
		Name:           namegen.Generate(),
		DomainID:       domainID,
		Status:         reports.EnabledStatus,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
		ReportTemplate: template,
		Metrics: []reports.ReqMetric{
			{
				ChannelID: generateUUID(t),
				Name:      "temperature",
			},
		},
	}

	saved, err := repo.AddReportConfig(context.Background(), reportConfig)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc     string
		domainID string
		reportID string
		err      error
	}{
		{
			desc:     "delete existing template",
			domainID: domainID,
			reportID: saved.ID,
			err:      nil,
		},
		{
			desc:     "delete template for non-existing report",
			domainID: domainID,
			reportID: generateUUID(t),
			err:      nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.DeleteReportTemplate(context.Background(), tc.domainID, tc.reportID)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

			if tc.reportID == saved.ID {
				tmpl, err := repo.ViewReportTemplate(context.Background(), tc.domainID, tc.reportID)
				require.Nil(t, err)
				require.Empty(t, tmpl)
			}
		})
	}
}
