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
	INSERT INTO rules (id, name, domain_id, tags, metadata, input_channel, input_topic, logic_type, logic_output, logic_value,
		output_channel, output_topic, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status)
	VALUES (:id, :name, :domain_id, :tags, :metadata, :input_channel, :input_topic, :logic_type, :logic_output, :logic_value,
		:output_channel, :output_topic, :start_datetime, :time, :recurring, :recurring_period, :created_at, :created_by, :updated_at, :updated_by, :status)
	RETURNING id, name, domain_id, tags, metadata, input_channel, input_topic, logic_type, logic_output, logic_value,
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
		SELECT id, name, domain_id, tags, metadata, input_channel, input_topic, logic_type, logic_output, logic_value, output_channel,
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
	RETURNING id, name, domain_id, tags, metadata, input_channel, input_topic, logic_type, logic_output, logic_value,
			output_channel, output_topic, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;`

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
		RETURNING id, name, domain_id, tags, metadata, input_channel, input_topic, logic_type, logic_output, logic_value,
			output_channel, output_topic, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;
	`, upq)

	return repo.update(ctx, r, q)
}

func (repo *PostgresRepository) UpdateRuleTags(ctx context.Context, r re.Rule) (re.Rule, error) {
	q := `UPDATE rules SET tags = :tags, updated_at = :updated_at, updated_by = :updated_by
	WHERE id = :id AND status = :status
	RETURNING id, name, domain_id, tags, metadata, input_channel, input_topic, logic_type, logic_output, logic_value,
		output_channel, output_topic, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;`
	r.Status = re.EnabledStatus

	return repo.update(ctx, r, q)
}

func (repo *PostgresRepository) UpdateRuleSchedule(ctx context.Context, r re.Rule) (re.Rule, error) {
	q := `
		UPDATE rules
		SET start_datetime = :start_datetime, time = :time, recurring = :recurring,
			recurring_period = :recurring_period, updated_at = :updated_at, updated_by = :updated_by WHERE id = :id
		RETURNING id, name, domain_id, tags, metadata, input_channel, input_topic, logic_type, logic_output, logic_value,
			output_channel, output_topic, start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status;
	`
	return repo.update(ctx, r, q)
}

func (repo *PostgresRepository) update(ctx context.Context, r re.Rule, query string) (re.Rule, error) {
	dbr, err := ruleToDb(r)
	if err != nil {
		return re.Rule{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	row, err := repo.DB.NamedQueryContext(ctx, query, dbr)
	if err != nil {
		return re.Rule{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()

	var dbRule dbRule
	if row.Next() {
		if err := row.StructScan(&dbRule); err != nil {
			return re.Rule{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
		}
		rule, err := dbToRule(dbRule)
		if err != nil {
			return re.Rule{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
		}
		return rule, nil
	}

	return re.Rule{}, repoerr.ErrNotFound
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
		SELECT id, name, domain_id, tags, input_channel, input_topic, logic_type, logic_output, logic_value, output_channel,
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
		RETURNING id, name, domain_id, tags, metadata, input_channel, input_topic, logic_type, logic_output, logic_value,
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
