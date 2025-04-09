// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/absmach/magistrala/re"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/postgres"
)

// SQL Queries as Strings.
const (
	addRuleQuery = `
		INSERT INTO rules (id, name, domain_id, metadata, input_channel, input_topic, logic_type, logic_value,
			output_channel, output_topic, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status)
		VALUES (:id, :name, :domain_id, :metadata, :input_channel, :input_topic, :logic_type, :logic_value,
			:output_channel, :output_topic, :start_datetime, :time, :recurring, :recurring_period, :created_at, :created_by, :updated_at, :updated_by, :status)
		RETURNING id, name, domain_id, metadata, input_channel, input_topic, logic_type, logic_value,
			output_channel, output_topic, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;
	`

	viewRuleQuery = `
		SELECT id, name, domain_id, metadata, input_channel, input_topic, logic_type, logic_value, output_channel, 
			output_topic, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status
		FROM rules
		WHERE id = $1;
	`

	updateRuleQuery = `
		UPDATE rules
		SET name = :name, metadata = :metadata, input_channel = :input_channel, input_topic = :input_topic, logic_type = :logic_type, 
			logic_value = :logic_value, output_channel = :output_channel, output_topic = :output_topic, 
			start_datetime = :start_datetime, time = :time, recurring = :recurring, 
			recurring_period = :recurring_period, updated_at = :updated_at, updated_by = :updated_by, status = :status
		WHERE id = :id
		RETURNING id, name, domain_id, metadata, input_channel, input_topic, logic_type, logic_value,
			output_channel, output_topic, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;
	`

	removeRuleQuery = `
		DELETE FROM rules
		WHERE id = $1;
	`

	updateRuleStatusQuery = `
		UPDATE rules
		SET status = $2
		WHERE id = $1
		RETURNING id, name, domain_id, metadata, input_channel, input_topic, logic_type, logic_value,
			output_channel, output_topic, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;  
	`

	listRulesQuery = `
		SELECT id, name, domain_id, input_channel, input_topic, logic_type, logic_value, output_channel, 
			output_topic, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status
		FROM rules r %s %s;
	`

	totalRuleQuery = `SELECT COUNT(*) FROM rules r %s;`

	addReportQuery = `
		INSERT INTO report_config (id, name, domain_id, channel_ids, client_ids, aggregation, metrics,
			"to", "from", start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status, subject, limit)
		VALUES (:id, :name, :domain_id, :channel_ids, :client_ids, :aggregation, :metrics,
			:to, :from, :start_datetime, :time, :recurring, :recurring_period, :created_at, :created_by, :updated_at, :updated_by, :status, :subject, :limit)
		RETURNING id, name, domain_id, channel_ids, client_ids, aggregation, metrics,
			"to", "from", start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status, subject, limit;
	`

	viewReportQuery = `
		SELECT id, name, domain_id, channel_ids, client_ids, aggregation, metrics,
			"to", "from", start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status, subject, limit
		FROM report_config
		WHERE id = $1;
	`

	updateReportQuery = `
    UPDATE report_config
    SET name = :name, channel_ids = :channel_ids, client_ids = :client_ids, aggregation = :aggregation, metrics = :metrics,
        start_datetime = :start_datetime, time = :time, recurring = :recurring, 
        recurring_period = :recurring_period, updated_at = :updated_at, updated_by = :updated_by, status = :status,
        "to" = :to, "from" = :from, subject = :subject, limit = :limit
    WHERE id = :id
    RETURNING id, name, domain_id, channel_ids, client_ids, aggregation, metrics,
        "to", "from", start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status, subject, limit;
`

	removeReportQuery = `
		DELETE FROM report_config
		WHERE id = $1;
	`

	updateReportStatusQuery = `
		UPDATE report_config
		SET status = $2
		WHERE id = $1
		RETURNING id, name, domain_id, channel_ids, client_ids, aggregation, metrics, "to", "from", subject,
			start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status, limit; 
	`

	listReportsQuery = `
		SELECT id, name, domain_id, channel_ids, client_ids, aggregation, metrics, "to", "from", subject,
			start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status, limit
		FROM report_config rc %s %s;
	`

	totalReportQuery = `SELECT COUNT(*) FROM report_config rc %s;`
)

type PostgresRepository struct {
	DB postgres.Database
}

func NewRepository(db postgres.Database) re.Repository {
	return &PostgresRepository{DB: db}
}

func (repo *PostgresRepository) AddRule(ctx context.Context, r re.Rule) (re.Rule, error) {
	dbr, err := ruleToDb(r)
	if err != nil {
		return re.Rule{}, err
	}
	row, err := repo.DB.NamedQueryContext(ctx, addRuleQuery, dbr)
	if err != nil {
		return re.Rule{}, err
	}
	defer row.Close()

	var dbRule dbRule
	if row.Next() {
		if err := row.StructScan(&dbRule); err != nil {
			return re.Rule{}, err
		}
	}

	rule, err := dbToRule(dbRule)
	if err != nil {
		return re.Rule{}, err
	}

	return rule, nil
}

func (repo *PostgresRepository) ViewRule(ctx context.Context, id string) (re.Rule, error) {
	row := repo.DB.QueryRowxContext(ctx, viewRuleQuery, id)
	if err := row.Err(); err != nil {
		return re.Rule{}, err
	}
	var dbr dbRule
	if err := row.StructScan(&dbr); err != nil {
		return re.Rule{}, err
	}
	ret, err := dbToRule(dbr)
	if err != nil {
		return re.Rule{}, err
	}

	return ret, nil
}

func (repo *PostgresRepository) UpdateRuleStatus(ctx context.Context, id string, status re.Status) (re.Rule, error) {
	row := repo.DB.QueryRowxContext(ctx, updateRuleStatusQuery, id, status)
	if err := row.Err(); err != nil {
		return re.Rule{}, err
	}

	var dbr dbRule
	if err := row.StructScan(&dbr); err != nil {
		return re.Rule{}, err
	}

	rule, err := dbToRule(dbr)
	if err != nil {
		return re.Rule{}, err
	}

	return rule, nil
}

func (repo *PostgresRepository) UpdateRule(ctx context.Context, r re.Rule) (re.Rule, error) {
	dbr, err := ruleToDb(r)
	if err != nil {
		return re.Rule{}, err
	}
	row, err := repo.DB.NamedQueryContext(ctx, updateRuleQuery, dbr)
	if err != nil {
		return re.Rule{}, err
	}
	defer row.Close()

	var dbRule dbRule
	if row.Next() {
		if err := row.StructScan(&dbRule); err != nil {
			return re.Rule{}, err
		}
	}
	rule, err := dbToRule(dbRule)
	if err != nil {
		return re.Rule{}, err
	}

	return rule, nil
}

func (repo *PostgresRepository) RemoveRule(ctx context.Context, id string) error {
	result, err := repo.DB.ExecContext(ctx, removeRuleQuery, id)
	if err != nil {
		return err
	}

	if _, err := result.RowsAffected(); err != nil {
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
	q := fmt.Sprintf(listRulesQuery, pq, pgData)
	rows, err := repo.DB.NamedQueryContext(ctx, q, pm)
	if err != nil {
		return re.Page{}, err
	}
	defer rows.Close()

	var rules []re.Rule
	for rows.Next() {
		var r dbRule
		if err := rows.StructScan(&r); err != nil {
			return re.Page{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}
		ret, err := dbToRule(r)
		if err != nil {
			return re.Page{}, err
		}
		rules = append(rules, ret)
	}

	cq := fmt.Sprintf(totalRuleQuery, pq)

	total, err := postgres.Total(ctx, repo.DB, cq, pm)
	if err != nil {
		return re.Page{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	pm.Total = total
	ret := re.Page{
		PageMeta: pm,
		Rules:    rules,
	}

	return ret, nil
}

func pageRulesQuery(pm re.PageMeta) string {
	var query []string
	if pm.InputChannel != "" {
		query = append(query, "r.input_channel = :input_channel")
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

	var q string
	if len(query) > 0 {
		q = fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
	}

	return q
}

func (repo *PostgresRepository) AddReportConfig(ctx context.Context, cfg re.ReportConfig) (re.ReportConfig, error) {
	dbr, err := reportToDb(cfg)
	if err != nil {
		return re.ReportConfig{}, err
	}
	row, err := repo.DB.NamedQueryContext(ctx, addReportQuery, dbr)
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
	row := repo.DB.QueryRowxContext(ctx, viewReportQuery, id)
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

func (repo *PostgresRepository) UpdateReportConfigStatus(ctx context.Context, id string, status re.Status) (re.ReportConfig, error) {
	row := repo.DB.QueryRowxContext(ctx, updateReportStatusQuery, id, status)
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

func (repo *PostgresRepository) UpdateReportConfig(ctx context.Context, cfg re.ReportConfig) (re.ReportConfig, error) {
	dbr, err := reportToDb(cfg)
	if err != nil {
		return re.ReportConfig{}, err
	}
	row, err := repo.DB.NamedQueryContext(ctx, updateReportQuery, dbr)
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

func (repo *PostgresRepository) RemoveReportConfig(ctx context.Context, id string) error {
	result, err := repo.DB.ExecContext(ctx, removeReportQuery, id)
	if err != nil {
		return err
	}

	if _, err := result.RowsAffected(); err != nil {
		return repoerr.ErrNotFound
	}

	return nil
}

func (repo *PostgresRepository) ListReportsConfig(ctx context.Context, pm re.PageMeta) (re.ReportConfigPage, error) {
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

	var cfgs []re.ReportConfig
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

	cq := fmt.Sprintf(totalReportQuery, pq)

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

func pageReportQuery(pm re.PageMeta) string {
	var query []string
	if pm.Status != re.AllStatus {
		query = append(query, "rc.status = :status")
	}

	if pm.Domain != "" {
		query = append(query, "rc.domain_id = :domain_id")
	}

	var q string
	if len(query) > 0 {
		q = fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
	}

	return q
}
