// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/reports"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/postgres"
)

type PostgresRepository struct {
	DB postgres.Database
}

func NewRepository(db postgres.Database) reports.Repository {
	return &PostgresRepository{DB: db}
}

func (repo *PostgresRepository) AddReportConfig(ctx context.Context, cfg reports.ReportConfig) (reports.ReportConfig, error) {
	q := `
		INSERT INTO report_config (id, name, description, domain_id, config, metrics,
			email, start_datetime, due, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status)
		VALUES (:id, :name, :description, :domain_id, :config, :metrics,
			:email, :start_datetime, :due, :recurring, :recurring_period, :created_at, :created_by, :updated_at, :updated_by, :status)
		RETURNING id, name, description, domain_id, config, metrics,
			email, start_datetime, due, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;
	`
	dbr, err := reportToDb(cfg)
	if err != nil {
		return reports.ReportConfig{}, err
	}
	row, err := repo.DB.NamedQueryContext(ctx, q, dbr)
	if err != nil {
		return reports.ReportConfig{}, err
	}
	defer row.Close()

	var dbReport dbReport
	if row.Next() {
		if err := row.StructScan(&dbReport); err != nil {
			return reports.ReportConfig{}, err
		}
	}

	report, err := dbToReport(dbReport)
	if err != nil {
		return reports.ReportConfig{}, err
	}

	return report, nil
}

func (repo *PostgresRepository) ViewReportConfig(ctx context.Context, id string) (reports.ReportConfig, error) {
	q := `
		SELECT id, name, description, domain_id, config, metrics,
			email, start_datetime, due, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status
		FROM report_config
		WHERE id = $1;
	`
	row := repo.DB.QueryRowxContext(ctx, q, id)
	if err := row.Err(); err != nil {
		return reports.ReportConfig{}, err
	}
	var dbr dbReport
	if err := row.StructScan(&dbr); err != nil {
		return reports.ReportConfig{}, err
	}
	rpt, err := dbToReport(dbr)
	if err != nil {
		return reports.ReportConfig{}, err
	}

	return rpt, nil
}

func (repo *PostgresRepository) UpdateReportConfigStatus(ctx context.Context, cfg reports.ReportConfig) (reports.ReportConfig, error) {
	q := `UPDATE report_config SET status = :status, updated_at = :updated_at, updated_by = :updated_by
		WHERE id = :id
        RETURNING id, name, description, domain_id, metrics, email, config,
			start_datetime, due, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;`

	dbRpt, err := reportToDb(cfg)
	if err != nil {
		return reports.ReportConfig{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	row, err := repo.DB.NamedQueryContext(ctx, q, dbRpt)
	if err != nil {
		return reports.ReportConfig{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()

	dbr := dbReport{}
	if row.Next() {
		if err := row.StructScan(&dbr); err != nil {
			return reports.ReportConfig{}, err
		}

		res, err := dbToReport(dbr)
		if err != nil {
			return reports.ReportConfig{}, err
		}
		return res, err
	}

	return reports.ReportConfig{}, repoerr.ErrNotFound
}

func (repo *PostgresRepository) UpdateReportConfig(ctx context.Context, cfg reports.ReportConfig) (reports.ReportConfig, error) {
	var query []string

	if cfg.Name != "" {
		query = append(query, "name = :name")
	}

	if cfg.Description != "" {
		query = append(query, "description = :description")
	}

	if len(cfg.Metrics) > 0 {
		query = append(query, "metrics = :metrics")
	}

	if cfg.Email != nil {
		query = append(query, "email = :email")
	}

	if cfg.Config != nil {
		query = append(query, "config = :config")
	}

	var q string
	if len(query) > 0 {
		q = fmt.Sprintf("%s", strings.Join(query, ", "))
	}

	q = fmt.Sprintf(`
		UPDATE report_config
		SET %s,
			updated_at = :updated_at, updated_by = :updated_by
		WHERE id = :id
		RETURNING id, name, description, domain_id, config, metrics,
			email, start_datetime, due, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;
		`, q)

	dbr, err := reportToDb(cfg)
	if err != nil {
		return reports.ReportConfig{}, err
	}
	row, err := repo.DB.NamedQueryContext(ctx, q, dbr)
	if err != nil {
		return reports.ReportConfig{}, err
	}
	defer row.Close()

	var dbReport dbReport
	if row.Next() {
		if err := row.StructScan(&dbReport); err != nil {
			return reports.ReportConfig{}, err
		}
	}
	rpt, err := dbToReport(dbReport)
	if err != nil {
		return reports.ReportConfig{}, err
	}

	return rpt, nil
}

func (repo *PostgresRepository) UpdateReportSchedule(ctx context.Context, cfg reports.ReportConfig) (reports.ReportConfig, error) {
	q := `
		UPDATE report_config
		SET start_datetime = :start_datetime, due = :due, recurring = :recurring,
			recurring_period = :recurring_period, updated_at = :updated_at, updated_by = :updated_by WHERE id = :id
		RETURNING id, name, description, domain_id, config, metrics,
			email, start_datetime, due, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;
	`

	dbr, err := reportToDb(cfg)
	if err != nil {
		return reports.ReportConfig{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}
	row, err := repo.DB.NamedQueryContext(ctx, q, dbr)
	if err != nil {
		return reports.ReportConfig{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()

	var dbReport dbReport
	if row.Next() {
		if err := row.StructScan(&dbReport); err != nil {
			return reports.ReportConfig{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
		}
	}
	report, err := dbToReport(dbReport)
	if err != nil {
		return reports.ReportConfig{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	return report, nil
}

func (repo *PostgresRepository) RemoveReportConfig(ctx context.Context, id string) error {
	q := `
		DELETE FROM report_config
		WHERE id = $1;
	`

	result, err := repo.DB.ExecContext(ctx, q, id)
	if err != nil {
		return err
	}

	if _, err := result.RowsAffected(); err != nil {
		return repoerr.ErrNotFound
	}

	return nil
}

func (repo *PostgresRepository) ListReportsConfig(ctx context.Context, pm reports.PageMeta) (reports.ReportConfigPage, error) {
	listReportsQuery := `
		SELECT id, name, description, domain_id, metrics, email, config,
			start_datetime, due, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status
		FROM report_config rc %s %s;
	`

	pgData := ""
	if pm.Limit != 0 {
		pgData = "LIMIT :limit"
	}
	if pm.Offset != 0 {
		pgData += " OFFSET :offset"
	}
	pq := pageReportQuery(pm)
	q := fmt.Sprintf(listReportsQuery, pq, pgData)
	rows, err := repo.DB.NamedQueryContext(ctx, q, pm)
	if err != nil {
		return reports.ReportConfigPage{}, err
	}
	defer rows.Close()

	cfgs := []reports.ReportConfig{}
	for rows.Next() {
		var r dbReport
		if err := rows.StructScan(&r); err != nil {
			return reports.ReportConfigPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}
		rpt, err := dbToReport(r)
		if err != nil {
			return reports.ReportConfigPage{}, err
		}
		cfgs = append(cfgs, rpt)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM report_config rc %s;`, pq)

	total, err := postgres.Total(ctx, repo.DB, cq, pm)
	if err != nil {
		return reports.ReportConfigPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	pm.Total = total
	ret := reports.ReportConfigPage{
		PageMeta:      pm,
		ReportConfigs: cfgs,
	}

	return ret, nil
}

func (repo *PostgresRepository) UpdateReportDue(ctx context.Context, id string, due time.Time) (reports.ReportConfig, error) {
	q := `
		UPDATE report_config
		SET due = :due, updated_at = :updated_at WHERE id = :id
		RETURNING id, name, description, domain_id, config, metrics,
			email, start_datetime, due, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;
	`

	dbr := dbReport{
		ID:        id,
		UpdatedAt: time.Now().UTC(),
		Due:       sql.NullTime{Time: due},
	}
	if !due.IsZero() {
		dbr.Due.Valid = true
	}

	row, err := repo.DB.NamedQueryContext(ctx, q, dbr)
	if err != nil {
		return reports.ReportConfig{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()

	var dbReport dbReport
	if row.Next() {
		if err := row.StructScan(&dbReport); err != nil {
			return reports.ReportConfig{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
		}
	}
	report, err := dbToReport(dbReport)
	if err != nil {
		return reports.ReportConfig{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	return report, nil
}

func pageReportQuery(pm reports.PageMeta) string {
	var query []string
	if pm.Status != reports.AllStatus {
		query = append(query, "rc.status = :status")
	}
	if pm.Domain != "" {
		query = append(query, "rc.domain_id = :domain_id")
	}
	if pm.ScheduledBefore != nil {
		query = append(query, "rc.due < :scheduled_before")
	}
	if pm.ScheduledAfter != nil {
		query = append(query, "rc.due > :scheduled_after")
	}
	if pm.Name != "" {
		query = append(query, "rc.name ILIKE '%' || :name || '%'")
	}

	var q string
	if len(query) > 0 {
		q = fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
	}

	return q
}
