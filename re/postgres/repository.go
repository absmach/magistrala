// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	mgPolicies "github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/re"
	api "github.com/absmach/supermq/api/http"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/postgres"
	rolesPostgres "github.com/absmach/supermq/pkg/roles/repo/postgres"
)

const (
	rolesTableNamePrefix = "rules"
	entityTableName      = "rules"
	entityIDColumnName   = "id"
)

type PostgresRepository struct {
	DB postgres.Database
	rolesPostgres.Repository
}

func NewRepository(db postgres.Database) re.Repository {
	rolesRepo := rolesPostgres.NewRepository(db, mgPolicies.RulesType, rolesTableNamePrefix, entityTableName, entityIDColumnName)
	return &PostgresRepository{
		DB:         db,
		Repository: rolesRepo,
	}
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
		SELECT id, name, domain_id, tags, metadata, input_channel, input_topic, logic_type, logic_value, outputs,
			start_datetime, time, recurring, recurring_period, created_at, created_by, updated_at, updated_by, status
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

func (repo *PostgresRepository) RetrieveByIDWithRoles(ctx context.Context, id, memberID string) (re.Rule, error) {
	query := `
	WITH selected_rule AS (
		SELECT
			r.id,
			r.domain_id
		FROM
			rules r
		WHERE
			r.id = :id
		LIMIT 1
	),
	selected_rule_roles AS (
		SELECT
			rr.entity_id AS rule_id,
			rrm.member_id AS member_id,
			rr.id AS role_id,
			rr."name" AS role_name,
			jsonb_agg(DISTINCT rra."action") AS actions,
			'direct' AS access_type,
			'' AS access_provider_id
		FROM
			rules_roles rr
		JOIN
			rules_role_members rrm ON rr.id = rrm.role_id
		JOIN
			rules_role_actions rra ON rr.id = rra.role_id
		JOIN
			selected_rule sr ON sr.id = rr.entity_id
			AND rrm.member_id = :member_id
		GROUP BY
			rr.entity_id, rr.id, rr.name, rrm.member_id
	),
	selected_domain_roles AS (
		SELECT
			sr.id AS rule_id,
			drm.member_id AS member_id,
			dr.id AS role_id,
			dr."name" AS role_name,
			jsonb_agg(DISTINCT all_actions."action") AS actions,
			'domain' AS access_type,
			dr.entity_id AS access_provider_id
		FROM
			domains d
		JOIN
			selected_rule sr ON sr.domain_id = d.id
		JOIN
			domains_roles dr ON dr.entity_id = d.id
		JOIN
			domains_role_members drm ON dr.id = drm.role_id
		JOIN
			domains_role_actions dra ON dr.id = dra.role_id
		JOIN
			domains_role_actions all_actions ON dr.id = all_actions.role_id
		WHERE
			drm.member_id = :member_id
			AND dra."action" LIKE 'rule%'
		GROUP BY
			sr.id, dr.entity_id, dr.id, dr."name", drm.member_id
	),
	all_roles AS (
		SELECT
			srr.rule_id,
			srr.member_id,
			srr.role_id,
			srr.role_name,
			srr.actions,
			srr.access_type,
			srr.access_provider_id
		FROM
			selected_rule_roles srr
		UNION
		SELECT
			sdr.rule_id,
			sdr.member_id,
			sdr.role_id,
			sdr.role_name,
			sdr.actions,
			sdr.access_type,
			sdr.access_provider_id
		FROM
			selected_domain_roles sdr
	),
	final_roles AS (
		SELECT
			ar.rule_id,
			ar.member_id,
			jsonb_agg(
				jsonb_build_object(
					'role_id', ar.role_id,
					'role_name', ar.role_name,
					'actions', ar.actions,
					'access_type', ar.access_type,
					'access_provider_id', ar.access_provider_id
				)
			) AS roles
		FROM all_roles ar
		GROUP BY
			ar.rule_id, ar.member_id
	)
	SELECT
		r2.id,
		r2."name",
		r2.domain_id,
		r2.tags,
		r2.metadata,
		r2.input_channel,
		r2.input_topic,
		r2.outputs,
		r2.status,
		r2.logic_type,
		r2.logic_value,
		r2.time,
		r2.recurring,
		r2.recurring_period,
		r2.start_datetime,
		r2.created_at,
		r2.created_by,
		r2.updated_at,
		r2.updated_by,
		fr.member_id,
		fr.roles
	FROM rules r2
	JOIN final_roles fr ON fr.rule_id = r2.id
	`
	parameters := map[string]any{
		"id":        id,
		"member_id": memberID,
	}
	row, err := repo.DB.NamedQueryContext(ctx, query, parameters)
	if err != nil {
		return re.Rule{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer row.Close()

	dbrule := dbRule{}
	if !row.Next() {
		return re.Rule{}, repoerr.ErrNotFound
	}

	if err := row.StructScan(&dbrule); err != nil {
		return re.Rule{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	r, err := dbToRule(dbrule)
	if err != nil {
		return re.Rule{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	return r, nil
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

	row, err := repo.DB.NamedQueryContext(ctx, query, dbr)
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
