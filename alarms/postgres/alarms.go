// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/absmach/magistrala/alarms"
	api "github.com/absmach/magistrala/api/http"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/pkg/postgres"
	"github.com/jmoiron/sqlx"
)

const alarmColumns = `alarms.id, alarms.rule_id, alarms.domain_id, alarms.channel_id, alarms.client_id, alarms.subtopic, alarms.measurement, alarms.value, alarms.unit,
alarms.threshold, alarms.cause, alarms.status, alarms.severity, alarms.assignee_id, alarms.created_at, alarms.updated_at, alarms.updated_by, alarms.assigned_at,
alarms.assigned_by, alarms.acknowledged_at, alarms.acknowledged_by, alarms.resolved_at, alarms.resolved_by, alarms.metadata`

type repository struct {
	db *sqlx.DB
}

var _ alarms.Repository = (*repository)(nil)

func NewAlarmsRepo(db *sqlx.DB) alarms.Repository {
	return &repository{db: db}
}

func (r *repository) CreateAlarm(ctx context.Context, alarm alarms.Alarm) (alarms.Alarm, error) {
	query := `
	WITH existing AS (
		SELECT status, severity
		FROM alarms
		WHERE domain_id = :domain_id
			AND rule_id = :rule_id
			AND channel_id = :channel_id
			AND client_id = :client_id
			AND subtopic = :subtopic
			AND measurement = :measurement
			AND created_at <= :created_at
		ORDER BY created_at DESC
		LIMIT 1
	)
	INSERT INTO alarms (
		id, rule_id, domain_id, channel_id, client_id, subtopic, measurement,
		value, unit, threshold, cause, status, severity, assignee_id,
		created_at, updated_at, updated_by, assigned_at, assigned_by,
		acknowledged_at, acknowledged_by, resolved_at, resolved_by, metadata
	)
	SELECT
		:id, :rule_id, :domain_id, :channel_id, :client_id, :subtopic, :measurement,
		:value, :unit, :threshold, :cause, :status, :severity, :assignee_id,
		:created_at, :updated_at, :updated_by, :assigned_at, :assigned_by,
		:acknowledged_at, :acknowledged_by, :resolved_at, :resolved_by, :metadata
	WHERE (
		EXISTS (
			SELECT 1 FROM existing
			WHERE existing.status IS DISTINCT FROM :status
			OR (:status = 0 AND existing.status = 0 AND existing.severity IS DISTINCT FROM :severity)
		)
		OR (
			NOT EXISTS (SELECT 1 FROM existing) AND :status = 0
		)
	)
	RETURNING
		id, rule_id, domain_id, channel_id, client_id, subtopic, measurement,
		value, unit, threshold, cause, status, severity, created_at,
		assignee_id, updated_at, updated_by, assigned_at, assigned_by,
		acknowledged_at, acknowledged_by, resolved_at, resolved_by, metadata
	;
	`
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
		return alarms.Alarm{}, repoerr.ErrNotFound
	}

	dba = dbAlarm{}
	if err := row.StructScan(&dba); err != nil {
		return alarms.Alarm{}, errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	return toAlarm(dba)
}

func (r *repository) UpdateAlarm(ctx context.Context, alarm alarms.Alarm) (alarms.Alarm, error) {
	var query []string
	var upq string
	if alarm.Status != 0 {
		query = append(query, "status = :status,")
	}
	if alarm.AssigneeID != "" {
		query = append(query, "assignee_id = :assignee_id,")
	}
	if !alarm.AssignedAt.IsZero() {
		query = append(query, "assigned_at = :assigned_at,")
	}
	if alarm.AssignedBy != "" {
		query = append(query, "assigned_by = :assigned_by,")
	}
	if alarm.AcknowledgedBy != "" {
		query = append(query, "acknowledged_by = :acknowledged_by,")
	}
	if !alarm.AcknowledgedAt.IsZero() {
		query = append(query, "acknowledged_at = :acknowledged_at,")
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
		RETURNING id, rule_id, domain_id, channel_id, client_id, subtopic, measurement, value, unit, threshold,
		cause, status, severity, assignee_id, assigned_at, assigned_by, acknowledged_at, acknowledged_by,
		resolved_by, resolved_at, metadata, created_at, updated_by, updated_at;`, upq)

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
		return alarms.Alarm{}, repoerr.ErrNotFound
	}

	dba = dbAlarm{}
	if err := row.StructScan(&dba); err != nil {
		return alarms.Alarm{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	return toAlarm(dba)
}

func (r *repository) ViewAlarm(ctx context.Context, alarmID, domainID string) (alarms.Alarm, error) {
	query := `SELECT * FROM alarms WHERE id = :id AND domain_id = :domain_id;`
	row, err := r.db.NamedQueryContext(ctx, query, map[string]any{
		"id": alarmID, "domain_id": domainID,
	})
	if err != nil {
		return alarms.Alarm{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer row.Close()

	if !row.Next() {
		return alarms.Alarm{}, repoerr.ErrNotFound
	}

	dba := dbAlarm{}
	if err := row.StructScan(&dba); err != nil {
		return alarms.Alarm{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	alarm, err := toAlarm(dba)
	if err != nil {
		return alarms.Alarm{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	return alarm, nil
}

func (r *repository) ListAllAlarms(ctx context.Context, pm alarms.PageMetadata) (alarms.AlarmsPage, error) {
	query, err := pageQuery(pm)
	if err != nil {
		return alarms.AlarmsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	comQuery := fmt.Sprintf(`SELECT %s FROM alarms %s`, alarmColumns, query)

	return r.alarmsPage(ctx, comQuery, pm)
}

func (r *repository) ListUserAlarms(ctx context.Context, userID string, pm alarms.PageMetadata) (alarms.AlarmsPage, error) {
	clauses := []string{
		`(
			EXISTS (
				SELECT 1
				FROM rules_roles rr
				JOIN rules_role_members rrm ON rrm.role_id = rr.id
				WHERE rr.entity_id = alarms.rule_id AND rrm.member_id = :user_id
			)
			OR EXISTS (
				SELECT 1
				FROM domains_roles dr
				JOIN domains_role_members drm ON drm.role_id = dr.id
				JOIN domains_role_actions dra ON dra.role_id = dr.id
				WHERE dr.entity_id = alarms.domain_id
					AND drm.member_id = :user_id
					AND dra.action LIKE 'alarm%'
			)
		)`,
	}

	clauses = append(clauses, pageQueryConditions(pm)...)
	query := fmt.Sprintf("WHERE %s", strings.Join(clauses, " AND "))
	pm.UserID = userID
	comQuery := fmt.Sprintf(`SELECT DISTINCT %s FROM alarms %s`, alarmColumns, query)

	return r.alarmsPage(ctx, comQuery, pm)
}

func (r *repository) alarmsPage(ctx context.Context, comQuery string, pm alarms.PageMetadata) (alarms.AlarmsPage, error) {
	dir := api.DescDir
	if pm.Dir == api.AscDir {
		dir = api.AscDir
	}

	var orderClause string
	switch pm.Order {
	case api.CreatedAtOrder:
		orderClause = fmt.Sprintf("ORDER BY created_at %s, id %s", dir, dir)
	default:
		orderClause = fmt.Sprintf("ORDER BY COALESCE(updated_at, created_at) %s, id %s", dir, dir)
	}

	q := fmt.Sprintf(`SELECT * FROM (%s) AS sub_query %s LIMIT :limit OFFSET :offset;`, comQuery, orderClause)
	cq := fmt.Sprintf(`SELECT COUNT(*) AS total_count FROM (%s) AS sub_query;`, comQuery)

	rows, err := r.db.NamedQueryContext(ctx, q, pm)
	if err != nil {
		return alarms.AlarmsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
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

	total, err := postgres.Total(ctx, r.db, cq, pm)
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
	result, err := r.db.NamedExecContext(ctx, query, map[string]any{"id": id})
	if err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
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

type dbAlarm struct {
	ID             string        `db:"id"`
	RuleID         string        `db:"rule_id"`
	DomainID       string        `db:"domain_id"`
	ChannelID      string        `db:"channel_id"`
	ClientID       string        `db:"client_id"`
	Subtopic       string        `db:"subtopic"`
	Measurement    string        `db:"measurement"`
	Value          string        `db:"value"`
	Unit           string        `db:"unit"`
	Cause          string        `db:"cause"`
	Threshold      string        `db:"threshold"`
	Status         alarms.Status `db:"status"`
	Severity       uint8         `db:"severity"`
	AssigneeID     string        `db:"assignee_id"`
	CreatedAt      time.Time     `db:"created_at"`
	UpdatedAt      sql.NullTime  `db:"updated_at,omitempty"`
	UpdatedBy      *string       `db:"updated_by,omitempty"`
	AssignedAt     sql.NullTime  `db:"assigned_at,omitempty"`
	AssignedBy     *string       `db:"assigned_by,omitempty"`
	AcknowledgedAt sql.NullTime  `db:"acknowledged_at,omitempty"`
	AcknowledgedBy *string       `db:"acknowledged_by,omitempty"`
	ResolvedAt     sql.NullTime  `db:"resolved_at,omitempty"`
	ResolvedBy     *string       `db:"resolved_by,omitempty"`
	Metadata       []byte        `db:"metadata,omitempty"`
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

	var acknowledgedBy *string
	if a.AcknowledgedBy != "" {
		acknowledgedBy = &a.AcknowledgedBy
	}
	var acknowledgedAt sql.NullTime
	if a.AcknowledgedAt != (time.Time{}) {
		acknowledgedAt = sql.NullTime{Time: a.AcknowledgedAt, Valid: true}
	}

	var resolvedBy *string
	if a.ResolvedBy != "" {
		resolvedBy = &a.ResolvedBy
	}
	var resolvedAt sql.NullTime
	if a.ResolvedAt != (time.Time{}) {
		resolvedAt = sql.NullTime{Time: a.ResolvedAt, Valid: true}
	}

	var assignedBy *string
	if a.AssignedBy != "" {
		assignedBy = &a.AssignedBy
	}
	var assignedAt sql.NullTime
	if a.AssignedAt != (time.Time{}) {
		assignedAt = sql.NullTime{Time: a.AssignedAt, Valid: true}
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
		ID:             a.ID,
		RuleID:         a.RuleID,
		DomainID:       a.DomainID,
		ChannelID:      a.ChannelID,
		ClientID:       a.ClientID,
		Subtopic:       a.Subtopic,
		Measurement:    a.Measurement,
		Value:          a.Value,
		Unit:           a.Unit,
		Cause:          a.Cause,
		Threshold:      a.Threshold,
		Status:         a.Status,
		Severity:       a.Severity,
		AssigneeID:     a.AssigneeID,
		CreatedAt:      a.CreatedAt,
		UpdatedAt:      updatedAt,
		UpdatedBy:      updatedBy,
		AssignedAt:     assignedAt,
		AssignedBy:     assignedBy,
		AcknowledgedAt: acknowledgedAt,
		AcknowledgedBy: acknowledgedBy,
		ResolvedAt:     resolvedAt,
		ResolvedBy:     resolvedBy,
		Metadata:       metadata,
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

	var assignedBy string
	if dbr.AssignedBy != nil {
		assignedBy = *dbr.AssignedBy
	}
	var assignedAt time.Time
	if dbr.AssignedAt.Valid {
		assignedAt = dbr.AssignedAt.Time
	}

	var acknowledgedBy string
	if dbr.AcknowledgedBy != nil {
		acknowledgedBy = *dbr.AcknowledgedBy
	}
	var acknowledgedAt time.Time
	if dbr.AcknowledgedAt.Valid {
		acknowledgedAt = dbr.AcknowledgedAt.Time
	}

	var resolvedBy string
	if dbr.ResolvedBy != nil {
		resolvedBy = *dbr.ResolvedBy
	}
	var resolvedAt time.Time
	if dbr.ResolvedAt.Valid {
		resolvedAt = dbr.ResolvedAt.Time
	}

	var metadata map[string]any
	if len(dbr.Metadata) > 0 {
		err := json.Unmarshal(dbr.Metadata, &metadata)
		if err != nil {
			return alarms.Alarm{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
	}

	return alarms.Alarm{
		ID:             dbr.ID,
		RuleID:         dbr.RuleID,
		DomainID:       dbr.DomainID,
		ChannelID:      dbr.ChannelID,
		ClientID:       dbr.ClientID,
		Subtopic:       dbr.Subtopic,
		Measurement:    dbr.Measurement,
		Value:          dbr.Value,
		Unit:           dbr.Unit,
		Threshold:      dbr.Threshold,
		Cause:          dbr.Cause,
		Status:         dbr.Status,
		Severity:       dbr.Severity,
		AssigneeID:     dbr.AssigneeID,
		CreatedAt:      dbr.CreatedAt,
		UpdatedAt:      updatedAt,
		UpdatedBy:      updatedBy,
		AssignedAt:     assignedAt,
		AssignedBy:     assignedBy,
		AcknowledgedAt: acknowledgedAt,
		AcknowledgedBy: acknowledgedBy,
		ResolvedAt:     resolvedAt,
		ResolvedBy:     resolvedBy,
		Metadata:       metadata,
	}, nil
}

func pageQuery(pm alarms.PageMetadata) (string, error) {
	query := pageQueryConditions(pm)

	var emq string
	if len(query) > 0 {
		emq = fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
	}

	return emq, nil
}

func pageQueryConditions(pm alarms.PageMetadata) []string {
	var query []string
	if pm.DomainID != "" {
		query = append(query, "alarms.domain_id = :domain_id")
	}
	if pm.RuleID != "" {
		query = append(query, "alarms.rule_id = :rule_id")
	}
	if pm.ChannelID != "" {
		query = append(query, "alarms.channel_id = :channel_id")
	}
	if pm.Subtopic != "" {
		query = append(query, "alarms.subtopic = :subtopic")
	}
	if pm.ClientID != "" {
		query = append(query, "alarms.client_id = :client_id")
	}
	if pm.Measurement != "" {
		query = append(query, "alarms.measurement = :measurement")
	}
	if pm.Status != alarms.AllStatus {
		query = append(query, "alarms.status = :status")
	}
	if pm.Severity != math.MaxUint8 {
		query = append(query, "alarms.severity = :severity")
	}
	if pm.AssigneeID != "" {
		query = append(query, "alarms.assignee_id = :assignee_id")
	}
	if pm.UpdatedBy != "" {
		query = append(query, "alarms.updated_by = :updated_by")
	}
	if pm.ResolvedBy != "" {
		query = append(query, "alarms.resolved_by = :resolved_by")
	}
	if pm.AcknowledgedBy != "" {
		query = append(query, "alarms.acknowledged_by = :acknowledged_by")
	}
	if pm.AssignedBy != "" {
		query = append(query, "alarms.assigned_by = :assigned_by")
	}
	if !pm.CreatedFrom.IsZero() {
		query = append(query, "alarms.created_at >= :created_from")
	}
	if !pm.CreatedTo.IsZero() {
		query = append(query, "alarms.created_at <= :created_to")
	}
	return query
}
