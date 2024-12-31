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
		INSERT INTO rules (id, domain_id, input_channel, input_topic, logic_type, logic_value,
			output_channel, output_topic, recurring_time, recurring_type, recurring_period, status)
		VALUES (:id, :domain_id, :input_channel, :input_topic, :logic_type, :logic_value,
			:output_channel, :output_topic, :recurring_time, :recurring_type, :recurring_period, :status)
		RETURNING id;
	`

	viewRuleQuery = `
		SELECT id, domain_id, input_channel, input_topic, logic_type, logic_value, output_channel, 
			output_topic, recurring_time, recurring_type, recurring_period, status
		FROM rules
		WHERE id = $1;
	`

	updateRuleQuery = `
		UPDATE rules
		SET input_channel = :input_channel, input_topic = :input_topic, logic_type = :logic_type, 
			logic_value = :logic_value, output_channel = :output_channel, output_topic = :output_topic, 
			recurring_time = :recurring_time, recurring_type = :recurring_type, 
			recurring_period = :recurring_period, status = :status
		WHERE id = :id;
	`

	removeRuleQuery = `
		DELETE FROM rules
		WHERE id = $1;
	`

	listRulesQuery = `
		SELECT id, domain_id, input_channel, input_topic, logic_type, logic_value, output_channel, 
			output_topic, recurring_time, recurring_type, recurring_period, status
		FROM rules r %s %s; 
	`

	totalQuery = `SELECT COUNT(*) FROM rules r %s;`
)

type PostgresRepository struct {
	DB postgres.Database
}

func NewRepository(db postgres.Database) re.Repository {
	return &PostgresRepository{DB: db}
}

func (repo *PostgresRepository) AddRule(ctx context.Context, r re.Rule) (re.Rule, error) {
	dbr := ruleToDb(r)
	_, err := repo.DB.NamedExecContext(ctx, addRuleQuery, dbr)
	if err != nil {
		return re.Rule{}, err
	}
	return r, nil
}

func (repo *PostgresRepository) ViewRule(ctx context.Context, id string) (re.Rule, error) {
	row := repo.DB.QueryRowxContext(ctx, viewRuleQuery, id)
	if err := row.Err(); err != nil {
		return re.Rule{}, err
	}
	var dbr dbRule
	err := row.StructScan(&dbr)
	if err != nil {
		return re.Rule{}, err
	}
	ret := dbToRule(dbr)

	return ret, nil
}

func (repo *PostgresRepository) UpdateRule(ctx context.Context, r re.Rule) (re.Rule, error) {
	dbr := ruleToDb(r)
	result, err := repo.DB.NamedExecContext(ctx, updateRuleQuery, dbr)
	if err != nil {
		return re.Rule{}, err
	}

	if _, err := result.RowsAffected(); err != nil {
		return re.Rule{}, repoerr.ErrNotFound
	}

	return r, nil
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
		pgData += " OFFEST :offset"
	}
	pq := pageQuery(pm)
	q := fmt.Sprintf(listRulesQuery, pq, pgData)
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
		rules = append(rules, dbToRule(r))
	}

	cq := fmt.Sprintf(totalQuery, pq)

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

func pageQuery(pm re.PageMeta) string {
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

	var q string
	if len(query) > 0 {
		q = fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
	}

	return q
}
