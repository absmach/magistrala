// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/absmach/magistrala/alarms"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/postgres"
	"github.com/jmoiron/sqlx"
)

type repository struct {
	db *sqlx.DB
}

var _ alarms.Repository = (*repository)(nil)

func NewAlarmsRepo(db *sqlx.DB) alarms.Repository {
	return &repository{db: db}
}

func (r *repository) CreateRule(ctx context.Context, rule alarms.Rule) (alarms.Rule, error) {
	query := `INSERT INTO rules (id, name, user_id, domain_id, condition, channel, metadata, created_by, created_at)
				VALUES (:id, :name, :user_id, :domain_id, :condition, :channel, :metadata, :created_by, :created_at)
				RETURNING id, name, user_id, domain_id, condition, channel, metadata, created_by, created_at;`
	dbr, err := toDBRule(rule)
	if err != nil {
		return alarms.Rule{}, errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	row, err := r.db.NamedQueryContext(ctx, query, dbr)
	if err != nil {
		return alarms.Rule{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
	}
	defer row.Close()

	if !row.Next() {
		return alarms.Rule{}, errors.Wrap(repoerr.ErrFailedOpDB, errors.New("no rows returned"))
	}

	dbr = dbRule{}
	if err := row.StructScan(&dbr); err != nil {
		return alarms.Rule{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	return toRule(dbr)
}

func (r *repository) UpdateRule(ctx context.Context, rule alarms.Rule) (alarms.Rule, error) {
	var query []string
	var upq string
	if rule.Name != "" {
		query = append(query, "name = :name,")
	}
	if rule.UserID != "" {
		query = append(query, "user_id = :user_id,")
	}
	if rule.DomainID != "" {
		query = append(query, "domain_id = :domain_id,")
	}
	if rule.Condition != "" {
		query = append(query, "condition = :condition,")
	}
	if rule.Channel != "" {
		query = append(query, "channel = :channel,")
	}
	if rule.Metadata != nil {
		query = append(query, "metadata = :metadata,")
	}
	if len(query) > 0 {
		upq = strings.Join(query, " ")
	}

	q := fmt.Sprintf(`UPDATE rules SET %s updated_by = :updated_by, updated_at = :updated_at WHERE id = :id
		RETURNING id, name, user_id, domain_id, condition, channel, metadata, created_by, created_at, updated_by, updated_at;`, upq)

	dbr, err := toDBRule(rule)
	if err != nil {
		return alarms.Rule{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}
	row, err := r.db.NamedQueryContext(ctx, q, dbr)
	if err != nil {
		return alarms.Rule{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()

	if !row.Next() {
		return alarms.Rule{}, errors.Wrap(repoerr.ErrFailedOpDB, repoerr.ErrNotFound)
	}

	dbr = dbRule{}
	if err := row.StructScan(&dbr); err != nil {
		return alarms.Rule{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	return toRule(dbr)
}

func (r *repository) ViewRule(ctx context.Context, id string) (alarms.Rule, error) {
	query := `SELECT * FROM rules WHERE id = :id;`
	row, err := r.db.NamedQueryContext(ctx, query, map[string]interface{}{"id": id})
	if err != nil {
		return alarms.Rule{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer row.Close()

	if !row.Next() {
		return alarms.Rule{}, errors.Wrap(repoerr.ErrFailedOpDB, repoerr.ErrNotFound)
	}

	dbr := dbRule{}
	if err := row.StructScan(&dbr); err != nil {
		return alarms.Rule{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	rule, err := toRule(dbr)
	if err != nil {
		return alarms.Rule{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	return rule, nil
}

func (r *repository) ListRules(ctx context.Context, pm alarms.PageMetadata) (alarms.RulesPage, error) {
	query, err := pageQuery(pm, "rules")
	if err != nil {
		return alarms.RulesPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	q := fmt.Sprintf(`SELECT * FROM rules %s ORDER BY created_at LIMIT :limit OFFSET :offset;`, query)
	rows, err := r.db.NamedQueryContext(ctx, q, pm)
	if err != nil {
		return alarms.RulesPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	defer rows.Close()

	var rules []alarms.Rule
	for rows.Next() {
		dbr := dbRule{}
		if err := rows.StructScan(&dbr); err != nil {
			return alarms.RulesPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		r, err := toRule(dbr)
		if err != nil {
			return alarms.RulesPage{}, err
		}

		rules = append(rules, r)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM rules %s;`, query)
	total, err := postgres.Total(ctx, r.db, cq, pm)
	if err != nil {
		return alarms.RulesPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	return alarms.RulesPage{
		Total:  total,
		Offset: pm.Offset,
		Limit:  pm.Limit,
		Rules:  rules,
	}, nil
}

func (r *repository) DeleteRule(ctx context.Context, id string) error {
	query := `DELETE FROM rules WHERE id = :id;`
	result, err := r.db.NamedExecContext(ctx, query, map[string]interface{}{"id": id})
	if err != nil {
		return errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	if rowsAffected == 0 {
		return errors.Wrap(repoerr.ErrFailedOpDB, repoerr.ErrNotFound)
	}

	return nil
}

func (r *repository) CreateAlarm(ctx context.Context, alarm alarms.Alarm) (alarms.Alarm, error) {
	query := `INSERT INTO alarms (id, rule_id, message, status, user_id, domain_id, assignee_id, metadata, created_by, created_at)
				VALUES (:id, :rule_id, :message, :status, :user_id, :domain_id, :assignee_id, :metadata, :created_by, :created_at)
				RETURNING id, rule_id, message, status, user_id, domain_id, assignee_id, metadata, created_by, created_at;`
	dba, err := toDBAlarm(alarm)
	if err != nil {
		return alarms.Alarm{}, errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	row, err := r.db.NamedQueryContext(ctx, query, dba)
	if err != nil {
		return alarms.Alarm{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
	}
	defer row.Close()

	if !row.Next() {
		return alarms.Alarm{}, errors.Wrap(repoerr.ErrFailedOpDB, errors.New("no rows returned"))
	}

	dba = dbAlarm{}
	if err := row.StructScan(&dba); err != nil {
		return alarms.Alarm{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	return toAlarm(dba)
}

func (r *repository) UpdateAlarm(ctx context.Context, alarm alarms.Alarm) (alarms.Alarm, error) {
	var query []string
	var upq string
	if alarm.Message != "" {
		query = append(query, "message = :message,")
	}
	if alarm.Status != 0 {
		query = append(query, "status = :status,")
	}
	if alarm.UserID != "" {
		query = append(query, "user_id = :user_id,")
	}
	if alarm.DomainID != "" {
		query = append(query, "domain_id = :domain_id,")
	}
	if alarm.AssigneeID != "" {
		query = append(query, "assignee_id = :assignee_id,")
	}
	if alarm.ResolvedBy != "" {
		query = append(query, "resolved_by = :resolved_by,")
	}
	if !alarm.ResolvedAt.IsZero() {
		query = append(query, "resolved_at = :resolved_at,")
	}
	if alarm.Metadata != nil {
		query = append(query, "metadata = :metadata,")
	}
	if len(query) > 0 {
		upq = strings.Join(query, " ")
	}

	q := fmt.Sprintf(`UPDATE alarms SET %s updated_by = :updated_by, updated_at = :updated_at WHERE id = :id
		RETURNING id, rule_id, message, status, user_id, domain_id, assignee_id, metadata, created_by, created_at, updated_by, updated_at, resolved_by, resolved_at;`, upq)

	dba, err := toDBAlarm(alarm)
	if err != nil {
		return alarms.Alarm{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}
	row, err := r.db.NamedQueryContext(ctx, q, dba)
	if err != nil {
		return alarms.Alarm{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()

	if !row.Next() {
		return alarms.Alarm{}, errors.Wrap(repoerr.ErrFailedOpDB, repoerr.ErrNotFound)
	}

	dba = dbAlarm{}
	if err := row.StructScan(&dba); err != nil {
		return alarms.Alarm{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	return toAlarm(dba)
}

func (r *repository) ViewAlarm(ctx context.Context, id string) (alarms.Alarm, error) {
	query := `SELECT * FROM alarms WHERE id = :id;`
	row, err := r.db.NamedQueryContext(ctx, query, map[string]interface{}{"id": id})
	if err != nil {
		return alarms.Alarm{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer row.Close()

	if !row.Next() {
		return alarms.Alarm{}, errors.Wrap(repoerr.ErrFailedOpDB, repoerr.ErrNotFound)
	}

	dba := dbAlarm{}
	if err := row.StructScan(&dba); err != nil {
		return alarms.Alarm{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	alarm, err := toAlarm(dba)
	if err != nil {
		return alarms.Alarm{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	return alarm, nil
}

func (r *repository) ListAlarms(ctx context.Context, pm alarms.PageMetadata) (alarms.AlarmsPage, error) {
	query, err := pageQuery(pm, "alarms")
	if err != nil {
		return alarms.AlarmsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	q := fmt.Sprintf(`SELECT * FROM alarms %s ORDER BY created_at LIMIT :limit OFFSET :offset;`, query)
	rows, err := r.db.NamedQueryContext(ctx, q, pm)
	if err != nil {
		return alarms.AlarmsPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	defer rows.Close()

	var items []alarms.Alarm
	for rows.Next() {
		dba := dbAlarm{}
		if err := rows.StructScan(&dba); err != nil {
			return alarms.AlarmsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		a, err := toAlarm(dba)
		if err != nil {
			return alarms.AlarmsPage{}, err
		}

		items = append(items, a)
	}

	q = fmt.Sprintf(`SELECT COUNT(*) FROM alarms %s;`, query)
	total, err := postgres.Total(ctx, r.db, q, pm)
	if err != nil {
		return alarms.AlarmsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	return alarms.AlarmsPage{
		Total:  total,
		Offset: pm.Offset,
		Limit:  pm.Limit,
		Alarms: items,
	}, nil
}

func (r *repository) DeleteAlarm(ctx context.Context, id string) error {
	query := `DELETE FROM alarms WHERE id = :id;`
	result, err := r.db.NamedExecContext(ctx, query, map[string]interface{}{"id": id})
	if err != nil {
		return errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	if rowsAffected == 0 {
		return errors.Wrap(repoerr.ErrFailedOpDB, repoerr.ErrNotFound)
	}

	return nil
}

func (r *repository) AssignAlarm(ctx context.Context, alarm alarms.Alarm) error {
	query := `UPDATE alarms SET
				assignee_id = :assignee_id, updated_at = :updated_at, updated_by = :updated_by WHERE id = :id;`
	result, err := r.db.NamedExecContext(ctx, query, map[string]interface{}{
		"id":          alarm.ID,
		"assignee_id": alarm.AssigneeID,
		"updated_at":  alarm.UpdatedAt,
		"updated_by":  alarm.UpdatedBy,
	})
	if err != nil {
		return errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	if rowsAffected == 0 {
		return errors.Wrap(repoerr.ErrFailedOpDB, repoerr.ErrNotFound)
	}

	return nil
}

type dbRule struct {
	ID        string       `db:"id"`
	Name      string       `db:"name"`
	UserID    string       `db:"user_id"`
	DomainID  string       `db:"domain_id"`
	Condition string       `db:"condition"`
	Channel   string       `db:"channel"`
	CreatedAt time.Time    `db:"created_at"`
	CreatedBy string       `db:"created_by"`
	UpdatedAt sql.NullTime `db:"updated_at,omitempty"`
	UpdatedBy *string      `db:"updated_by,omitempty"`
	Metadata  []byte       `db:"metadata,omitempty"`
}

func toDBRule(a alarms.Rule) (dbRule, error) {
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now()
	}
	var updatedBy *string
	if a.UpdatedBy != "" {
		updatedBy = &a.UpdatedBy
	}
	var updatedAt sql.NullTime
	if a.UpdatedAt != (time.Time{}) {
		updatedAt = sql.NullTime{Time: a.UpdatedAt, Valid: true}
	}

	metadata := []byte("{}")
	if len(a.Metadata) > 0 {
		b, err := json.Marshal(a.Metadata)
		if err != nil {
			return dbRule{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
		metadata = b
	}

	return dbRule{
		ID:        a.ID,
		Name:      a.Name,
		UserID:    a.UserID,
		DomainID:  a.DomainID,
		Condition: a.Condition,
		Channel:   a.Channel,
		CreatedAt: a.CreatedAt,
		CreatedBy: a.CreatedBy,
		UpdatedAt: updatedAt,
		UpdatedBy: updatedBy,
		Metadata:  metadata,
	}, nil
}

func toRule(dbr dbRule) (alarms.Rule, error) {
	var metadata alarms.Metadata
	if dbr.Metadata != nil {
		if err := json.Unmarshal([]byte(dbr.Metadata), &metadata); err != nil {
			return alarms.Rule{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
	}

	var updatedBy string
	if dbr.UpdatedBy != nil {
		updatedBy = *dbr.UpdatedBy
	}
	var updatedAt time.Time
	if dbr.UpdatedAt.Valid {
		updatedAt = dbr.UpdatedAt.Time
	}

	return alarms.Rule{
		ID:        dbr.ID,
		Name:      dbr.Name,
		UserID:    dbr.UserID,
		DomainID:  dbr.DomainID,
		Condition: dbr.Condition,
		Channel:   dbr.Channel,
		CreatedAt: dbr.CreatedAt,
		CreatedBy: dbr.CreatedBy,
		UpdatedAt: updatedAt,
		UpdatedBy: updatedBy,
		Metadata:  metadata,
	}, nil
}

type dbAlarm struct {
	ID         string        `db:"id"`
	RuleID     string        `db:"rule_id"`
	Message    string        `db:"message"`
	Status     alarms.Status `db:"status"`
	UserID     string        `db:"user_id"`
	DomainID   string        `db:"domain_id"`
	AssigneeID string        `db:"assignee_id"`
	CreatedAt  time.Time     `db:"created_at"`
	CreatedBy  string        `db:"created_by"`
	UpdatedAt  sql.NullTime  `db:"updated_at,omitempty"`
	UpdatedBy  *string       `db:"updated_by,omitempty"`
	ResolvedAt sql.NullTime  `db:"resolved_at,omitempty"`
	ResolvedBy *string       `db:"resolved_by,omitempty"`
	Metadata   []byte        `db:"metadata,omitempty"`
}

func toDBAlarm(a alarms.Alarm) (dbAlarm, error) {
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now()
	}
	var updatedBy *string
	if a.UpdatedBy != "" {
		updatedBy = &a.UpdatedBy
	}
	var updatedAt sql.NullTime
	if a.UpdatedAt != (time.Time{}) {
		updatedAt = sql.NullTime{Time: a.UpdatedAt, Valid: true}
	}

	var resolvedBy *string
	if a.ResolvedBy != "" {
		resolvedBy = &a.ResolvedBy
	}
	var resolvedAt sql.NullTime
	if a.ResolvedAt != (time.Time{}) {
		resolvedAt = sql.NullTime{Time: a.ResolvedAt, Valid: true}
	}

	metadata := []byte("{}")
	if len(a.Metadata) > 0 {
		b, err := json.Marshal(a.Metadata)
		if err != nil {
			return dbAlarm{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
		metadata = b
	}

	return dbAlarm{
		ID:         a.ID,
		RuleID:     a.RuleID,
		Message:    a.Message,
		Status:     a.Status,
		UserID:     a.UserID,
		DomainID:   a.DomainID,
		AssigneeID: a.AssigneeID,
		CreatedAt:  a.CreatedAt,
		CreatedBy:  a.CreatedBy,
		UpdatedAt:  updatedAt,
		UpdatedBy:  updatedBy,
		ResolvedAt: resolvedAt,
		ResolvedBy: resolvedBy,
		Metadata:   metadata,
	}, nil
}

func toAlarm(dbr dbAlarm) (alarms.Alarm, error) {
	var updatedBy string
	if dbr.UpdatedBy != nil {
		updatedBy = *dbr.UpdatedBy
	}
	var updatedAt time.Time
	if dbr.UpdatedAt.Valid {
		updatedAt = dbr.UpdatedAt.Time
	}

	var resolvedBy string
	if dbr.ResolvedBy != nil {
		resolvedBy = *dbr.ResolvedBy
	}
	var resolvedAt time.Time
	if dbr.ResolvedAt.Valid {
		resolvedAt = dbr.ResolvedAt.Time
	}

	var metadata map[string]interface{}
	if len(dbr.Metadata) > 0 {
		err := json.Unmarshal(dbr.Metadata, &metadata)
		if err != nil {
			return alarms.Alarm{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
	}

	return alarms.Alarm{
		ID:         dbr.ID,
		RuleID:     dbr.RuleID,
		Message:    dbr.Message,
		Status:     dbr.Status,
		UserID:     dbr.UserID,
		DomainID:   dbr.DomainID,
		AssigneeID: dbr.AssigneeID,
		CreatedAt:  dbr.CreatedAt,
		CreatedBy:  dbr.CreatedBy,
		UpdatedAt:  updatedAt,
		UpdatedBy:  updatedBy,
		ResolvedAt: resolvedAt,
		ResolvedBy: resolvedBy,
		Metadata:   metadata,
	}, nil
}

func pageQuery(pm alarms.PageMetadata, table string) (string, error) {
	var query []string
	if pm.UserID != "" {
		query = append(query, "user_id = :user_id")
	}
	if pm.DomainID != "" {
		query = append(query, "domain_id = :domain_id")
	}
	if pm.ChannelID != "" {
		query = append(query, "channel_id = :channel_id")
	}
	if pm.RuleID != "" {
		query = append(query, "rule_id = :rule_id")
	}
	if table == "alarms" {
		if pm.Status != alarms.AllStatus {
			query = append(query, "status = :status")
		}
	}
	if pm.AssigneeID != "" {
		query = append(query, "assignee_id = :assignee_id")
	}

	var emq string
	if len(query) > 0 {
		emq = fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
	}

	return emq, nil
}
