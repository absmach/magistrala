// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/absmach/magistrala/re"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/postgres"
)

type PostgresRepository struct {
	DB postgres.Database
}

func NewRepository(db postgres.Database) re.Repository {
	return &PostgresRepository{DB: db}
}

func (repo *PostgresRepository) AddRule(ctx context.Context, r re.Rule) (re.Rule, error) {
	q := `
	INSERT INTO rules (id, name, domain_id, metadata, input_channel, input_topic, logic_type, logic_output, logic_value,
		output_channel, output_topic, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status)
	VALUES (:id, :name, :domain_id, :metadata, :input_channel, :input_topic, :logic_type, :logic_output, :logic_value,
		:output_channel, :output_topic, :start_datetime, :time, :recurring, :recurring_period, :created_at, :created_by, :updated_at, :updated_by, :status)
	RETURNING id, name, domain_id, metadata, input_channel, input_topic, logic_type, logic_output, logic_value,
		output_channel, output_topic, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;
`
	dbr, err := ruleToDb(r)
	if err != nil {
		return re.Rule{}, err
	}
	row, err := repo.DB.NamedQueryContext(ctx, q, dbr)
	if err != nil {
		return re.Rule{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
	}
	defer row.Close()

	var dbRule dbRule
	if row.Next() {
		if err := row.StructScan(&dbRule); err != nil {
			return re.Rule{}, errors.Wrap(repoerr.ErrCreateEntity, err)
		}
	}

	rule, err := dbToRule(dbRule)
	if err != nil {
		return re.Rule{}, errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	return rule, nil
}

func (repo *PostgresRepository) ViewRule(ctx context.Context, id string) (re.Rule, error) {
	q := `
		SELECT id, name, domain_id, metadata, input_channel, input_topic, logic_type, logic_output, logic_value, output_channel, 
			output_topic, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status
		FROM rules
		WHERE id = $1;
	`
	row := repo.DB.QueryRowxContext(ctx, q, id)
	if err := row.Err(); err != nil {
		return re.Rule{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	var dbr dbRule
	if err := row.StructScan(&dbr); err != nil {
		return re.Rule{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	ret, err := dbToRule(dbr)
	if err != nil {
		return re.Rule{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	return ret, nil
}

func (repo *PostgresRepository) UpdateRuleStatus(ctx context.Context, r re.Rule) (re.Rule, error) {
	q := `UPDATE rules 
	SET status = :status, updated_at = :updated_at, updated_by = :updated_by
	WHERE id = :id
	RETURNING id, name, domain_id, metadata, input_channel, input_topic, logic_type, logic_output, logic_value,
			output_channel, output_topic, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;`

	dbr, err := ruleToDb(r)
	if err != nil {
		return re.Rule{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	row, err := repo.DB.NamedQueryContext(ctx, q, dbr)
	if err != nil {
		return re.Rule{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()

	var res dbRule
	if row.Next() {
		if err := row.StructScan(&res); err != nil {
			return re.Rule{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
		}

		rule, err := dbToRule(res)
		if err != nil {
			return re.Rule{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
		}

		return rule, nil
	}

	return re.Rule{}, repoerr.ErrNotFound
}

func (repo *PostgresRepository) UpdateRule(ctx context.Context, r re.Rule) (re.Rule, error) {
	var query []string
	var upq string
	if r.Name != "" {
		query = append(query, "name = :name,")
	}
	if r.Metadata != nil {
		query = append(query, "metadata = :metadata,")
	}
	if r.InputChannel != "" {
		query = append(query, "input_channel = :input_channel,")
	}
	if r.InputTopic != "" {
		query = append(query, "input_topic = :input_topic,")
	}
	if r.OutputChannel != "" {
		query = append(query, "output_channel = :output_channel,")
	}
	if r.OutputTopic != "" {
		query = append(query, "output_topic = :output_topic,")
	}
	if r.Logic.Value != "" {
		query = append(query, "logic_type = :logic_type,")
		query = append(query, "logic_output = :logic_output,")
		query = append(query, "logic_value = :logic_value,")
	}

	if len(query) > 0 {
		upq = strings.Join(query, " ")
	}

	q := fmt.Sprintf(`
		UPDATE rules
		SET %s updated_at = :updated_at, updated_by = :updated_by WHERE id = :id
		RETURNING id, name, domain_id, metadata, input_channel, input_topic, logic_type, logic_output, logic_value,
			output_channel, output_topic, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;
	`, upq)

	dbr, err := ruleToDb(r)
	if err != nil {
		return re.Rule{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}
	row, err := repo.DB.NamedQueryContext(ctx, q, dbr)
	if err != nil {
		return re.Rule{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()

	var dbRule dbRule
	if row.Next() {
		if err := row.StructScan(&dbRule); err != nil {
			return re.Rule{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
		}
	}
	rule, err := dbToRule(dbRule)
	if err != nil {
		return re.Rule{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	return rule, nil
}

func (repo *PostgresRepository) UpdateRuleSchedule(ctx context.Context, r re.Rule) (re.Rule, error) {
	q := `
		UPDATE rules
		SET start_datetime = :start_datetime, time = :time, recurring = :recurring, 
			recurring_period = :recurring_period, updated_at = :updated_at, updated_by = :updated_by WHERE id = :id
		RETURNING id, name, domain_id, metadata, input_channel, input_topic, logic_type, logic_output, logic_value,
			output_channel, output_topic, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;
	`
	dbr, err := ruleToDb(r)
	if err != nil {
		return re.Rule{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}
	row, err := repo.DB.NamedQueryContext(ctx, q, dbr)
	if err != nil {
		return re.Rule{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()

	var dbRule dbRule
	if row.Next() {
		if err := row.StructScan(&dbRule); err != nil {
			return re.Rule{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
		}
	}
	rule, err := dbToRule(dbRule)
	if err != nil {
		return re.Rule{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	return rule, nil
}

func (repo *PostgresRepository) RemoveRule(ctx context.Context, id string) error {
	q := `
	DELETE FROM rules
	WHERE id = $1;
`
	result, err := repo.DB.ExecContext(ctx, q, id)
	if err != nil {
		return postgres.HandleError(repoerr.ErrRemoveEntity, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	if rowsAffected == 0 {
		return repoerr.ErrNotFound
	}

	return nil
}

func (repo *PostgresRepository) ListRules(ctx context.Context, pm re.PageMeta) (re.Page, error) {
	pgData := ""
	if pm.Limit != 0 {
		pgData = "LIMIT :limit"
	}
	if pm.Offset != 0 {
		pgData += " OFFSET :offset"
	}
	pq := pageRulesQuery(pm)
	q := fmt.Sprintf(`
		SELECT id, name, domain_id, input_channel, input_topic, logic_type, logic_output, logic_value, output_channel, 
			output_topic, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status
		FROM rules r %s %s;
	`, pq, pgData)
	rows, err := repo.DB.NamedQueryContext(ctx, q, pm)
	if err != nil {
		return re.Page{}, err
	}
	defer rows.Close()

	var rules []re.Rule
	var r dbRule
	for rows.Next() {
		if err := rows.StructScan(&r); err != nil {
			return re.Page{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}
		ret, err := dbToRule(r)
		if err != nil {
			return re.Page{}, err
		}
		rules = append(rules, ret)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM rules r %s;`, pq)

	total, err := postgres.Total(ctx, repo.DB, cq, pm)
	if err != nil {
		return re.Page{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	ret := re.Page{
		Total:  total,
		Offset: pm.Offset,
		Limit:  pm.Limit,
		Rules:  rules,
	}

	return ret, nil
}

func (repo *PostgresRepository) UpdateRuleDue(ctx context.Context, id string, due time.Time) (re.Rule, error) {
	q := `
		UPDATE rules
		SET time = :time, updated_at = :updated_at WHERE id = :id
		RETURNING id, name, domain_id, metadata, input_channel, input_topic, logic_type, logic_output, logic_value,
			output_channel, output_topic, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;
	`
	dbr := dbRule{
		ID:        id,
		UpdatedAt: time.Now().UTC(),
		Time:      sql.NullTime{Time: due},
	}
	if !due.IsZero() {
		dbr.Time.Valid = true
	}
	row, err := repo.DB.NamedQueryContext(ctx, q, dbr)
	if err != nil {
		return re.Rule{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()

	var dbRule dbRule
	if row.Next() {
		if err := row.StructScan(&dbRule); err != nil {
			return re.Rule{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
		}
	}
	rule, err := dbToRule(dbRule)
	if err != nil {
		return re.Rule{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	return rule, nil
}

func pageRulesQuery(pm re.PageMeta) string {
	var query []string
	if pm.InputChannel != "" {
		query = append(query, "r.input_channel = :input_channel")
	}
	if pm.InputTopic != nil {
		query = append(query, "r.input_topic = :input_topic")
	}
	if pm.OutputChannel != "" {
		query = append(query, "r.output_channel = :output_channel")
	}
	if pm.Status != re.AllStatus {
		query = append(query, "r.status = :status")
	}
	if pm.Domain != "" {
		query = append(query, "r.domain_id = :domain_id")
	}
	if pm.ScheduledBefore != nil {
		query = append(query, "r.time < :scheduled_before")
	}
	if pm.ScheduledAfter != nil {
		query = append(query, "r.time > :scheduled_after")
	}

	var q string
	if len(query) > 0 {
		q = fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
	}

	return q
}

func (repo *PostgresRepository) AddReportConfig(ctx context.Context, cfg re.ReportConfig) (re.ReportConfig, error) {
	q := `
		INSERT INTO report_config (id, name, description, domain_id, config, metrics,
			email, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status)
		VALUES (:id, :name, :description, :domain_id, :config, :metrics,
			:email, :start_datetime, :time, :recurring, :recurring_period, :created_at, :created_by, :updated_at, :updated_by, :status)
		RETURNING id, name, description, domain_id, config, metrics,
			email, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;
	`
	dbr, err := reportToDb(cfg)
	if err != nil {
		return re.ReportConfig{}, err
	}
	row, err := repo.DB.NamedQueryContext(ctx, q, dbr)
	if err != nil {
		return re.ReportConfig{}, err
	}
	defer row.Close()

	var dbReport dbReport
	if row.Next() {
		if err := row.StructScan(&dbReport); err != nil {
			return re.ReportConfig{}, err
		}
	}

	report, err := dbToReport(dbReport)
	if err != nil {
		return re.ReportConfig{}, err
	}

	return report, nil
}

func (repo *PostgresRepository) ViewReportConfig(ctx context.Context, id string) (re.ReportConfig, error) {
	q := `
		SELECT id, name, description, domain_id, config, metrics,
			email, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status
		FROM report_config
		WHERE id = $1;
	`
	row := repo.DB.QueryRowxContext(ctx, q, id)
	if err := row.Err(); err != nil {
		return re.ReportConfig{}, err
	}
	var dbr dbReport
	if err := row.StructScan(&dbr); err != nil {
		return re.ReportConfig{}, err
	}
	rpt, err := dbToReport(dbr)
	if err != nil {
		return re.ReportConfig{}, err
	}

	return rpt, nil
}

func (repo *PostgresRepository) UpdateReportConfigStatus(ctx context.Context, cfg re.ReportConfig) (re.ReportConfig, error) {
	q := `UPDATE report_config SET status = :status, updated_at = :updated_at, updated_by = :updated_by
		WHERE id = :id
        RETURNING id, name, description, domain_id, metrics, email, config,
			start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;`

	dbRpt, err := reportToDb(cfg)
	if err != nil {
		return re.ReportConfig{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	row, err := repo.DB.NamedQueryContext(ctx, q, dbRpt)
	if err != nil {
		return re.ReportConfig{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()

	dbr := dbReport{}
	if row.Next() {
		if err := row.StructScan(&dbr); err != nil {
			return re.ReportConfig{}, err
		}

		res, err := dbToReport(dbr)
		if err != nil {
			return re.ReportConfig{}, err
		}
		return res, err
	}

	return re.ReportConfig{}, repoerr.ErrNotFound
}

func (repo *PostgresRepository) UpdateReportConfig(ctx context.Context, cfg re.ReportConfig) (re.ReportConfig, error) {
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
			email, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;
		`, q)

	dbr, err := reportToDb(cfg)
	if err != nil {
		return re.ReportConfig{}, err
	}
	row, err := repo.DB.NamedQueryContext(ctx, q, dbr)
	if err != nil {
		return re.ReportConfig{}, err
	}
	defer row.Close()

	var dbReport dbReport
	if row.Next() {
		if err := row.StructScan(&dbReport); err != nil {
			return re.ReportConfig{}, err
		}
	}
	rpt, err := dbToReport(dbReport)
	if err != nil {
		return re.ReportConfig{}, err
	}

	return rpt, nil
}

func (repo *PostgresRepository) UpdateReportSchedule(ctx context.Context, cfg re.ReportConfig) (re.ReportConfig, error) {
	q := `
		UPDATE report_config
		SET start_datetime = :start_datetime, time = :time, recurring = :recurring, 
			recurring_period = :recurring_period, updated_at = :updated_at, updated_by = :updated_by WHERE id = :id
		RETURNING id, name, description, domain_id, config, metrics,
			email, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;
	`

	dbr, err := reportToDb(cfg)
	if err != nil {
		return re.ReportConfig{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}
	row, err := repo.DB.NamedQueryContext(ctx, q, dbr)
	if err != nil {
		return re.ReportConfig{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()

	var dbReport dbReport
	if row.Next() {
		if err := row.StructScan(&dbReport); err != nil {
			return re.ReportConfig{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
		}
	}
	report, err := dbToReport(dbReport)
	if err != nil {
		return re.ReportConfig{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
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

func (repo *PostgresRepository) ListReportsConfig(ctx context.Context, pm re.PageMeta) (re.ReportConfigPage, error) {
	listReportsQuery := `
		SELECT id, name, description, domain_id, metrics, email, config,
			start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status
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
		return re.ReportConfigPage{}, err
	}
	defer rows.Close()

	cfgs := []re.ReportConfig{}
	for rows.Next() {
		var r dbReport
		if err := rows.StructScan(&r); err != nil {
			return re.ReportConfigPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}
		rpt, err := dbToReport(r)
		if err != nil {
			return re.ReportConfigPage{}, err
		}
		cfgs = append(cfgs, rpt)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM report_config rc %s;`, pq)

	total, err := postgres.Total(ctx, repo.DB, cq, pm)
	if err != nil {
		return re.ReportConfigPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	pm.Total = total
	ret := re.ReportConfigPage{
		PageMeta:      pm,
		ReportConfigs: cfgs,
	}

	return ret, nil
}

func (repo *PostgresRepository) UpdateReportDue(ctx context.Context, id string, due time.Time) (re.ReportConfig, error) {
	q := `
		UPDATE report_config
		SET time = :time, updated_at = :updated_at WHERE id = :id
		RETURNING id, name, description, domain_id, config, metrics,
			email, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;
	`

	dbr := dbReport{
		ID:        id,
		UpdatedAt: time.Now().UTC(),
		Time:      sql.NullTime{Time: due},
	}
	if !due.IsZero() {
		dbr.Time.Valid = true
	}

	row, err := repo.DB.NamedQueryContext(ctx, q, dbr)
	if err != nil {
		return re.ReportConfig{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()

	var dbReport dbReport
	if row.Next() {
		if err := row.StructScan(&dbReport); err != nil {
			return re.ReportConfig{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
		}
	}
	report, err := dbToReport(dbReport)
	if err != nil {
		return re.ReportConfig{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	return report, nil
}

func pageReportQuery(pm re.PageMeta) string {
	var query []string
	if pm.Status != re.AllStatus {
		query = append(query, "rc.status = :status")
	}
	if pm.Domain != "" {
		query = append(query, "rc.domain_id = :domain_id")
	}
	if pm.ScheduledBefore != nil {
		query = append(query, "rc.time < :scheduled_before")
	}
	if pm.ScheduledAfter != nil {
		query = append(query, "rc.time > :scheduled_after")
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
