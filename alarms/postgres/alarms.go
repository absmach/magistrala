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
	"github.com/absmach/magistrala/pkg/policies"
	api "github.com/absmach/supermq/api/http"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/postgres"
	"github.com/absmach/supermq/pkg/roles"
	rolesPostgres "github.com/absmach/supermq/pkg/roles/repo/postgres"
	"github.com/jmoiron/sqlx"
)

const (
	rolesTableNamePrefix = "alarms"
	entityTableName      = "alarms"
	entityIDColumnName   = "id"
)

type repository struct {
	db *sqlx.DB
	rolesPostgres.Repository
}

var _ alarms.Repository = (*repository)(nil)

func NewAlarmsRepo(db *sqlx.DB) alarms.Repository {
	rolesRepo := rolesPostgres.NewRepository(db, policies.AlarmsType, rolesTableNamePrefix, entityTableName, entityIDColumnName)
	return &repository{
		db:         db,
		Repository: rolesRepo,
	}
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

func (r *repository) RetrieveByIDWithRoles(ctx context.Context, id, memberID string) (alarms.Alarm, error) {
	query := `
	WITH selected_alarm AS (
		SELECT
			a.id,
			a.domain_id
		FROM
			alarms a
		WHERE
			a.id = :id
		LIMIT 1
	),
	selected_alarm_roles AS (
		SELECT
			ar.entity_id AS alarm_id,
			arm.member_id AS member_id,
			ar.id AS role_id,
			ar."name" AS role_name,
			jsonb_agg(DISTINCT ara."action") AS actions,
			'direct' AS access_type,
			'' AS access_provider_id
		FROM
			alarms_roles ar
		JOIN
			alarms_role_members arm ON ar.id = arm.role_id
		JOIN
			alarms_role_actions ara ON ar.id = ara.role_id
		JOIN
			selected_alarm sa ON sa.id = ar.entity_id
			AND arm.member_id = :member_id
		GROUP BY
			ar.entity_id, ar.id, ar.name, arm.member_id
	),
	selected_domain_roles AS (
		SELECT
			sa.id AS alarm_id,
			drm.member_id AS member_id,
			dr.id AS role_id,
			dr."name" AS role_name,
			jsonb_agg(DISTINCT all_actions."action") AS actions,
			'domain' AS access_type,
			dr.entity_id AS access_provider_id
		FROM
			domains d
		JOIN
			selected_alarm sa ON sa.domain_id = d.id
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
			AND dra."action" LIKE 'alarm%'
		GROUP BY
			sa.id, dr.entity_id, dr.id, dr."name", drm.member_id
	),
	all_roles AS (
		SELECT
			sar.alarm_id,
			sar.member_id,
			sar.role_id,
			sar.role_name,
			sar.actions,
			sar.access_type,
			sar.access_provider_id
		FROM
			selected_alarm_roles sar
		UNION
		SELECT
			sdr.alarm_id,
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
			ar.alarm_id,
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
			ar.alarm_id, ar.member_id
	)
	SELECT
		a2.id,
		a2.rule_id,
		a2.domain_id,
		a2.channel_id,
		a2.subtopic,
		a2.client_id,
		a2.measurement,
		a2.value,
		a2.unit,
		a2.threshold,
		a2.cause,
		a2.status,
		a2.severity,
		a2.assignee_id,
		a2.created_at,
		a2.updated_at,
		a2.updated_by,
		a2.assigned_at,
		a2.assigned_by,
		a2.acknowledged_at,
		a2.acknowledged_by,
		a2.resolved_at,
		a2.resolved_by,
		a2.metadata,
		fr.member_id,
		fr.roles
	FROM alarms a2
	JOIN final_roles fr ON fr.alarm_id = a2.id
	`
	parameters := map[string]any{
		"id":        id,
		"member_id": memberID,
	}
	row, err := r.db.NamedQueryContext(ctx, query, parameters)
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

func (r *repository) ListAlarms(ctx context.Context, pm alarms.PageMetadata) (alarms.AlarmsPage, error) {
	query, err := pageQuery(pm)
	if err != nil {
		return alarms.AlarmsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	dir := api.DescDir
	if pm.Dir == api.AscDir {
		dir = api.AscDir
	}

	orderClause := ""

	switch pm.Order {
	case api.CreatedAtOrder:
		orderClause = fmt.Sprintf("ORDER BY created_at %s, id %s", dir, dir)
	case api.UpdatedAtOrder:
		orderClause = fmt.Sprintf("ORDER BY COALESCE(updated_at, created_at) %s, id %s", dir, dir)
	default:
		orderClause = fmt.Sprintf("ORDER BY COALESCE(updated_at, created_at) %s, id %s", dir, dir)
	}

	q := fmt.Sprintf(`SELECT id, rule_id, domain_id, channel_id, client_id, subtopic, measurement, value, unit,
			threshold, cause, status, severity, assignee_id, created_at, updated_at, updated_by, assigned_at,
			assigned_by, acknowledged_at, acknowledged_by, resolved_at, resolved_by, metadata
			FROM alarms %s %s LIMIT :limit OFFSET :offset;`, query, orderClause)

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
	MemberID       string        `db:"member_id,omitempty"`
	Roles          json.RawMessage `db:"roles,omitempty"`
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

	var roles []roles.MemberRoleActions
	if dbr.Roles != nil {
		if err := json.Unmarshal(dbr.Roles, &roles); err != nil {
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
		Roles:          roles,
	}, nil
}

func pageQuery(pm alarms.PageMetadata) (string, error) {
	var query []string
	if pm.DomainID != "" {
		query = append(query, "domain_id = :domain_id")
	}
	if pm.RuleID != "" {
		query = append(query, "rule_id = :rule_id")
	}
	if pm.ChannelID != "" {
		query = append(query, "channel_id = :channel_id")
	}
	if pm.Subtopic != "" {
		query = append(query, "subtopic = :subtopic")
	}
	if pm.ClientID != "" {
		query = append(query, "client_id = :client_id")
	}
	if pm.Measurement != "" {
		query = append(query, "measurement = :measurement")
	}
	if pm.Status != alarms.AllStatus {
		query = append(query, "status = :status")
	}
	if pm.Severity != math.MaxUint8 {
		query = append(query, "severity = :severity")
	}
	if pm.AssigneeID != "" {
		query = append(query, "assignee_id = :assignee_id")
	}
	if pm.UpdatedBy != "" {
		query = append(query, "updated_by = :updated_by")
	}
	if pm.ResolvedBy != "" {
		query = append(query, "resolved_by = :resolved_by")
	}
	if pm.AcknowledgedBy != "" {
		query = append(query, "acknowledged_by = :acknowledged_by")
	}
	if pm.AssignedBy != "" {
		query = append(query, "assigned_by = :assigned_by")
	}
	if !pm.CreatedFrom.IsZero() {
		query = append(query, "created_at >= :created_from")
	}
	if !pm.CreatedTo.IsZero() {
		query = append(query, "created_at <= :created_to")
	}

	var emq string
	if len(query) > 0 {
		emq = fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
	}

	return emq, nil
}
