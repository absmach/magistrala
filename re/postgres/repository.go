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
	api "github.com/absmach/supermq/api/http"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/postgres"
)

type PostgresRepository struct {
	db postgres.Database
}

func NewRepository(db postgres.Database) re.Repository {
	return &PostgresRepository{db: db}
}

func (repo *PostgresRepository) AddRule(ctx context.Context, r re.Rule) (re.Rule, error) {
	q := `
	INSERT INTO rules (id, name, domain_id, tags, metadata, input_channel, input_topic, logic_type, logic_value,
		outputs, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status)
	VALUES (:id, :name, :domain_id, :tags, :metadata, :input_channel, :input_topic, :logic_type, :logic_value,
		:outputs, :start_datetime, :time, :recurring, :recurring_period, :created_at, :created_by, :updated_at, :updated_by, :status)
	RETURNING id, name, domain_id, tags, metadata, input_channel, input_topic, logic_type, logic_value,
		outputs, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;
`
	dbr, err := ruleToDb(r)
	if err != nil {
		return re.Rule{}, err
	}
	row, err := repo.db.NamedQueryContext(ctx, q, dbr)
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
		SELECT id, name, domain_id, tags, metadata, input_channel, input_topic, logic_type, logic_value, outputs,
			start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status
		FROM rules
		WHERE id = $1;
	`
	row := repo.db.QueryRowxContext(ctx, q, id)
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
	RETURNING id, name, domain_id, tags, metadata, input_channel, input_topic, logic_type, logic_value,
			outputs, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;`

	return repo.update(ctx, r, q)
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
	query = append(query, "input_channel = :input_channel,")
	query = append(query, "input_topic = :input_topic,")
	if r.Outputs != nil {
		query = append(query, "outputs = :outputs, ")
	}
	if r.Logic.Value != "" {
		query = append(query, "logic_type = :logic_type,")
		query = append(query, "logic_value = :logic_value,")
	}

	if len(query) > 0 {
		upq = strings.Join(query, " ")
	}

	q := fmt.Sprintf(`
		UPDATE rules
		SET %s updated_at = :updated_at, updated_by = :updated_by WHERE id = :id
		RETURNING id, name, domain_id, tags, metadata, input_channel, input_topic, logic_type, logic_value,
			outputs, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;
	`, upq)

	return repo.update(ctx, r, q)
}

func (repo *PostgresRepository) UpdateRuleTags(ctx context.Context, r re.Rule) (re.Rule, error) {
	q := `UPDATE rules SET tags = :tags, updated_at = :updated_at, updated_by = :updated_by
	WHERE id = :id AND status = :status
	RETURNING id, name, domain_id, tags, metadata, input_channel, input_topic, logic_type, logic_value,
		outputs, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;`
	r.Status = re.EnabledStatus

	return repo.update(ctx, r, q)
}

func (repo *PostgresRepository) UpdateRuleSchedule(ctx context.Context, r re.Rule) (re.Rule, error) {
	q := `
		UPDATE rules
		SET start_datetime = :start_datetime, time = :time, recurring = :recurring,
			recurring_period = :recurring_period, updated_at = :updated_at, updated_by = :updated_by WHERE id = :id
		RETURNING id, name, domain_id, tags, metadata, input_channel, input_topic, logic_type, logic_value,
			outputs, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;
	`
	return repo.update(ctx, r, q)
}

func (repo *PostgresRepository) update(ctx context.Context, r re.Rule, query string) (re.Rule, error) {
	dbr, err := ruleToDb(r)
	if err != nil {
		return re.Rule{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	row, err := repo.db.NamedQueryContext(ctx, query, dbr)
	if err != nil {
		return re.Rule{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()
	if !row.Next() {
		return re.Rule{}, repoerr.ErrNotFound
	}
	var dbRule dbRule
	if err := row.StructScan(&dbRule); err != nil {
		return re.Rule{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
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
	result, err := repo.db.ExecContext(ctx, q, id)
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

	dir := api.DescDir
	if pm.Dir == api.AscDir {
		dir = api.AscDir
	}

	orderClause := ""

	switch pm.Order {
	case api.NameKey:
		orderClause = fmt.Sprintf("ORDER BY name %s, id %s", dir, dir)
	case api.CreatedAtOrder:
		orderClause = fmt.Sprintf("ORDER BY created_at %s, id %s", dir, dir)
	case api.UpdatedAtOrder:
		orderClause = fmt.Sprintf("ORDER BY COALESCE(updated_at, created_at) %s, id %s", dir, dir)
	default:
		orderClause = fmt.Sprintf("ORDER BY COALESCE(updated_at, created_at) %s, id %s", dir, dir)
	}

	q := fmt.Sprintf(`
		SELECT id, name, domain_id, tags, input_channel, input_topic, logic_type, logic_value, outputs,
			start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status
		FROM rules r %s %s %s;
	`, pq, orderClause, pgData)
	rows, err := repo.db.NamedQueryContext(ctx, q, pm)
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

	total, err := postgres.Total(ctx, repo.db, cq, pm)
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
		RETURNING id, name, domain_id, tags, metadata, input_channel, input_topic, logic_type, logic_value,
			outputs, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;
	`
	dbr := dbRule{
		ID:        id,
		UpdatedAt: time.Now().UTC(),
		Time:      sql.NullTime{Time: due},
	}
	if !due.IsZero() {
		dbr.Time.Valid = true
	}
	row, err := repo.db.NamedQueryContext(ctx, q, dbr)
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
	if pm.Status != re.AllStatus {
		query = append(query, "r.status = :status")
	}
	if pm.Domain != "" {
		query = append(query, "r.domain_id = :domain_id")
	}
	if pm.Tag != "" {
		query = append(query, "EXISTS (SELECT 1 FROM unnest(tags) AS tag WHERE tag ILIKE '%' || :tag || '%')")
	}
	if pm.ScheduledBefore != nil {
		query = append(query, "r.time < :scheduled_before")
	}
	if pm.ScheduledAfter != nil {
		query = append(query, "r.time > :scheduled_after")
	}
	if pm.Name != "" {
		query = append(query, "r.name ILIKE '%' || :name || '%'")
	}
	if pm.Scheduled != nil && !*pm.Scheduled {
		query = append(query, "r.time IS NULL")
	}

	var q string
	if len(query) > 0 {
		q = fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
	}

	return q
}

func (repo *PostgresRepository) AddLog(ctx context.Context, log re.RuleLog) error {
	q := `
		INSERT INTO rule_executions (id, rule_id, level, message, error, exec_time, created_at)
		VALUES (:id, :rule_id, :level, :message, :error, :exec_time, :created_at);
	`
	dbLog := dbRuleLog{
		ID:        log.ID,
		RuleID:    log.RuleID,
		Level:     log.Level,
		Message:   log.Message,
		Error:     toNullString(log.Error),
		ExecTime:  log.ExecTime,
		CreatedAt: log.CreatedAt,
	}

	row, err := repo.db.NamedQueryContext(ctx, q, dbLog)
	if err != nil {
		return postgres.HandleError(repoerr.ErrCreateEntity, err)
	}
	defer row.Close()

	return nil
}

func (repo *PostgresRepository) ListLogs(ctx context.Context, pm re.LogPageMeta) (re.LogPage, error) {
	var conditions []string
	params := make(map[string]any)

	if pm.RuleID != "" {
		conditions = append(conditions, "rl.rule_id = :rule_id")
		params["rule_id"] = pm.RuleID
	}
	if pm.DomainID != "" {
		conditions = append(conditions, "r.domain_id = :domain_id")
		params["domain_id"] = pm.DomainID
	}
	if pm.Level != "" {
		conditions = append(conditions, "rl.level = :level")
		params["level"] = pm.Level
	}
	if pm.FromTime != nil {
		conditions = append(conditions, "rl.created_at >= :from_time")
		params["from_time"] = pm.FromTime
	}
	if pm.ToTime != nil {
		conditions = append(conditions, "rl.created_at <= :to_time")
		params["to_time"] = pm.ToTime
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	params["limit"] = pm.Limit
	params["offset"] = pm.Offset

	order := "rl.created_at"
	if pm.Order != "" {
		if pm.Order == "name" {
			order = "r.name"
		} else {
			order = "rl." + pm.Order
		}
	}

	dir := "DESC"
	if pm.Dir != "" {
		dir = strings.ToUpper(pm.Dir)
	}

	q := fmt.Sprintf(`
		SELECT rl.id, rl.rule_id, rl.level, rl.message, rl.error, rl.exec_time, rl.created_at
		FROM rule_executions rl
		INNER JOIN rules r ON rl.rule_id = r.id
		%s
		ORDER BY %s %s
		LIMIT :limit OFFSET :offset;
	`, whereClause, order, dir)

	rows, err := repo.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return re.LogPage{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	var logs []re.RuleLog
	for rows.Next() {
		var log dbRuleLog
		if err := rows.StructScan(&log); err != nil {
			return re.LogPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		logs = append(logs, dbToRuleLog(log))
	}

	cq := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM rule_executions rl
		INNER JOIN rules r ON rl.rule_id = r.id
		%s;
	`, whereClause)
	total, err := postgres.Total(ctx, repo.db, cq, params)
	if err != nil {
		return re.LogPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	return re.LogPage{
		Total:  total,
		Offset: pm.Offset,
		Limit:  pm.Limit,
		Logs:   logs,
	}, nil
}
