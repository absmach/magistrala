// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// --- source row structs (Magistrala) ---

type srcDomain struct {
	ID        string         `db:"id"`
	Name      sql.NullString `db:"name"`
	Route     sql.NullString `db:"route"`
	Tags      pq.StringArray `db:"tags"`
	Metadata  []byte         `db:"metadata"`
	CreatedAt sql.NullTime   `db:"created_at"`
	UpdatedAt sql.NullTime   `db:"updated_at"`
	CreatedBy sql.NullString `db:"created_by"`
	UpdatedBy sql.NullString `db:"updated_by"`
	Status    int16          `db:"status"`
}

type srcUser struct {
	ID             string         `db:"id"`
	FirstName      sql.NullString `db:"first_name"`
	LastName       sql.NullString `db:"last_name"`
	Username       sql.NullString `db:"username"`
	Email          sql.NullString `db:"email"`
	Metadata       []byte         `db:"metadata"`
	ProfilePicture sql.NullString `db:"profile_picture"`
	AuthProvider   sql.NullString `db:"auth_provider"`
	Status         int16          `db:"status"`
	Role           sql.NullInt16  `db:"role"`
	VerifiedAt     sql.NullTime   `db:"verified_at"`
	CreatedAt      sql.NullTime   `db:"created_at"`
	UpdatedAt      sql.NullTime   `db:"updated_at"`
}

type srcClient struct {
	ID            string         `db:"id"`
	Name          sql.NullString `db:"name"`
	DomainID      string         `db:"domain_id"`
	ParentGroupID sql.NullString `db:"parent_group_id"`
	Identity      sql.NullString `db:"identity"`
	Secret        sql.NullString `db:"secret"`
	Tags          pq.StringArray `db:"tags"`
	Metadata      []byte         `db:"metadata"`
	PrivateMeta   []byte         `db:"private_metadata"`
	Status        int16          `db:"status"`
	CreatedAt     sql.NullTime   `db:"created_at"`
	UpdatedAt     sql.NullTime   `db:"updated_at"`
}

type srcChannel struct {
	ID            string         `db:"id"`
	Name          sql.NullString `db:"name"`
	DomainID      string         `db:"domain_id"`
	ParentGroupID sql.NullString `db:"parent_group_id"`
	Route         sql.NullString `db:"route"`
	Tags          pq.StringArray `db:"tags"`
	Metadata      []byte         `db:"metadata"`
	CreatedBy     sql.NullString `db:"created_by"`
	Status        int16          `db:"status"`
	CreatedAt     sql.NullTime   `db:"created_at"`
	UpdatedAt     sql.NullTime   `db:"updated_at"`
}

type srcConnection struct {
	ChannelID string `db:"channel_id"`
	DomainID  string `db:"domain_id"`
	ClientID  string `db:"client_id"`
	Type      int16  `db:"type"`
}

type srcGroup struct {
	ID          string         `db:"id"`
	ParentID    sql.NullString `db:"parent_id"`
	DomainID    string         `db:"domain_id"`
	Name        string         `db:"name"`
	Description sql.NullString `db:"description"`
	Metadata    []byte         `db:"metadata"`
	Tags        pq.StringArray `db:"tags"`
	Status      int16          `db:"status"`
	CreatedAt   sql.NullTime   `db:"created_at"`
	UpdatedAt   sql.NullTime   `db:"updated_at"`
}

// srcRole / action / member are generic across the *_roles families.
type srcRole struct {
	ID        string       `db:"id"`
	Name      string       `db:"name"`
	EntityID  string       `db:"entity_id"`
	CreatedAt sql.NullTime `db:"created_at"`
	UpdatedAt sql.NullTime `db:"updated_at"`
}

type srcRoleAction struct {
	RoleID string `db:"role_id"`
	Action string `db:"action"`
}

type srcRoleMember struct {
	RoleID   string `db:"role_id"`
	MemberID string `db:"member_id"`
}

type srcPAT struct {
	ID        string         `db:"id"`
	Name      string         `db:"name"`
	UserID    sql.NullString `db:"user_id"`
	Desc      sql.NullString `db:"description"`
	ExpiresAt sql.NullTime   `db:"expires_at"`
	Revoked   sql.NullBool   `db:"revoked"`
	IssuedAt  sql.NullTime   `db:"issued_at"`
}

// srcRule is a rules-engine rule (rules_engine.rules) -> Atom resource kind=rule.
type srcRule struct {
	ID              string          `db:"id"`
	Name            sql.NullString  `db:"name"`
	DomainID        string          `db:"domain_id"`
	Metadata        []byte          `db:"metadata"`
	CreatedBy       sql.NullString  `db:"created_by"`
	CreatedAt       sql.NullTime    `db:"created_at"`
	UpdatedAt       sql.NullTime    `db:"updated_at"`
	UpdatedBy       sql.NullString  `db:"updated_by"`
	InputChannel    sql.NullString  `db:"input_channel"`
	InputTopic      sql.NullString  `db:"input_topic"`
	Outputs         json.RawMessage `db:"outputs"`
	Status          int16           `db:"status"`
	LogicType       int16           `db:"logic_type"`
	LogicValue      []byte          `db:"logic_value"`
	Time            sql.NullTime    `db:"time"`
	Recurring       sql.NullInt16   `db:"recurring"`
	RecurringPeriod sql.NullInt16   `db:"recurring_period"`
	StartDatetime   sql.NullTime    `db:"start_datetime"`
	Tags            pq.StringArray  `db:"tags"`
}

// srcReport is a report config (reports.report_config) -> Atom resource kind=report.
type srcReport struct {
	ID              string          `db:"id"`
	Name            sql.NullString  `db:"name"`
	Description     sql.NullString  `db:"description"`
	DomainID        string          `db:"domain_id"`
	Status          int16           `db:"status"`
	CreatedAt       sql.NullTime    `db:"created_at"`
	CreatedBy       sql.NullString  `db:"created_by"`
	UpdatedAt       sql.NullTime    `db:"updated_at"`
	UpdatedBy       sql.NullString  `db:"updated_by"`
	Due             sql.NullTime    `db:"due"`
	Recurring       sql.NullInt16   `db:"recurring"`
	RecurringPeriod sql.NullInt16   `db:"recurring_period"`
	StartDatetime   sql.NullTime    `db:"start_datetime"`
	Config          json.RawMessage `db:"config"`
	Email           json.RawMessage `db:"email"`
	Metrics         json.RawMessage `db:"metrics"`
	ReportTemplate  sql.NullString  `db:"report_template"`
}

// srcAlarm is an alarm (alarms.alarms) -> Atom resource kind=alarm.
type srcAlarm struct {
	ID             string         `db:"id"`
	RuleID         string         `db:"rule_id"`
	DomainID       string         `db:"domain_id"`
	ChannelID      string         `db:"channel_id"`
	Subtopic       string         `db:"subtopic"`
	ClientID       string         `db:"client_id"`
	Measurement    string         `db:"measurement"`
	Value          string         `db:"value"`
	Unit           string         `db:"unit"`
	Threshold      string         `db:"threshold"`
	Cause          string         `db:"cause"`
	Status         int16          `db:"status"`
	Severity       int16          `db:"severity"`
	AssigneeID     sql.NullString `db:"assignee_id"`
	CreatedAt      sql.NullTime   `db:"created_at"`
	UpdatedAt      sql.NullTime   `db:"updated_at"`
	UpdatedBy      sql.NullString `db:"updated_by"`
	AssignedAt     sql.NullTime   `db:"assigned_at"`
	AssignedBy     sql.NullString `db:"assigned_by"`
	AcknowledgedAt sql.NullTime   `db:"acknowledged_at"`
	AcknowledgedBy sql.NullString `db:"acknowledged_by"`
	ResolvedAt     sql.NullTime   `db:"resolved_at"`
	ResolvedBy     sql.NullString `db:"resolved_by"`
	Metadata       []byte         `db:"metadata"`
}

// --- readers ---

func readDomains(ctx context.Context, db *sqlx.DB) ([]srcDomain, error) {
	var out []srcDomain
	q := `SELECT id, name, route, tags, metadata, created_at, updated_at, created_by, updated_by, status FROM domains`
	return out, db.SelectContext(ctx, &out, q)
}

func readUsers(ctx context.Context, db *sqlx.DB) ([]srcUser, error) {
	var out []srcUser
	q := `SELECT id, first_name, last_name, username, email, metadata, profile_picture,
	             auth_provider, status, role, verified_at, created_at, updated_at
	      FROM users`
	return out, db.SelectContext(ctx, &out, q)
}

func readClients(ctx context.Context, db *sqlx.DB) ([]srcClient, error) {
	var out []srcClient
	q := `SELECT id, name, domain_id, parent_group_id, identity, secret, tags, metadata, private_metadata,
	             status, created_at, updated_at
	      FROM clients`
	return out, db.SelectContext(ctx, &out, q)
}

func readConnections(ctx context.Context, db *sqlx.DB) ([]srcConnection, error) {
	var out []srcConnection
	return out, db.SelectContext(ctx, &out, `SELECT channel_id, domain_id, client_id, type FROM connections`)
}

func readChannels(ctx context.Context, db *sqlx.DB) ([]srcChannel, error) {
	var out []srcChannel
	q := `SELECT id, name, domain_id, parent_group_id, route, tags, metadata, created_by,
	             status, created_at, updated_at
	      FROM channels`
	return out, db.SelectContext(ctx, &out, q)
}

func readGroups(ctx context.Context, db *sqlx.DB) ([]srcGroup, error) {
	var out []srcGroup
	q := `SELECT id, parent_id, domain_id, name, description, metadata, tags, status,
	             created_at, updated_at
	      FROM groups`
	return out, db.SelectContext(ctx, &out, q)
}

func readRules(ctx context.Context, db *sqlx.DB) ([]srcRule, error) {
	var out []srcRule
	q := `SELECT id, name, domain_id, metadata, created_by, created_at, updated_at, updated_by,
	             input_channel, input_topic, outputs, status, logic_type, logic_value,
	             "time", recurring, recurring_period, start_datetime, tags
	      FROM rules`
	return out, db.SelectContext(ctx, &out, q)
}

func readReports(ctx context.Context, db *sqlx.DB) ([]srcReport, error) {
	var out []srcReport
	q := `SELECT id, name, description, domain_id, status, created_at, created_by, updated_at,
	             updated_by, due, recurring, recurring_period, start_datetime,
	             config, email, metrics, report_template
	      FROM report_config`
	return out, db.SelectContext(ctx, &out, q)
}

func readAlarms(ctx context.Context, db *sqlx.DB) ([]srcAlarm, error) {
	var out []srcAlarm
	q := `SELECT id, rule_id, domain_id, channel_id, subtopic, client_id, measurement, value,
	             unit, threshold, cause, status, severity, assignee_id, created_at, updated_at,
	             updated_by, assigned_at, assigned_by, acknowledged_at, acknowledged_by,
	             resolved_at, resolved_by, metadata
	      FROM alarms`
	return out, db.SelectContext(ctx, &out, q)
}

// readRoleFamily reads <prefix>_roles, _role_actions, _role_members for one service.
func readRoleFamily(ctx context.Context, db *sqlx.DB, prefix string) ([]srcRole, []srcRoleAction, []srcRoleMember, error) {
	var roles []srcRole
	if err := db.SelectContext(ctx, &roles,
		`SELECT id, name, entity_id, created_at, updated_at FROM `+prefix+`_roles`); err != nil {
		return nil, nil, nil, err
	}
	var acts []srcRoleAction
	if err := db.SelectContext(ctx, &acts,
		`SELECT role_id, action FROM `+prefix+`_role_actions`); err != nil {
		return nil, nil, nil, err
	}
	var mems []srcRoleMember
	if err := db.SelectContext(ctx, &mems,
		`SELECT role_id, member_id FROM `+prefix+`_role_members`); err != nil {
		return nil, nil, nil, err
	}
	return roles, acts, mems, nil
}

func readPATs(ctx context.Context, db *sqlx.DB) ([]srcPAT, error) {
	var out []srcPAT
	q := `SELECT id, name, user_id, description, expires_at, revoked, issued_at FROM pats`
	return out, db.SelectContext(ctx, &out, q)
}

type srcPATScope struct {
	PatID      string         `db:"pat_id"`
	DomainID   sql.NullString `db:"domain_id"`
	EntityType string         `db:"entity_type"`
	Operation  string         `db:"operation"`
	EntityID   string         `db:"entity_id"`
}

func readPATScopes(ctx context.Context, db *sqlx.DB) ([]srcPATScope, error) {
	var out []srcPATScope
	q := `SELECT pat_id, domain_id, entity_type, operation, entity_id FROM pat_scopes`
	return out, db.SelectContext(ctx, &out, q)
}

type srcInvitation struct {
	InvitedBy   string       `db:"invited_by"`
	InviteeID   string       `db:"invitee_user_id"`
	DomainID    string       `db:"domain_id"`
	RoleID      string       `db:"role_id"`
	CreatedAt   sql.NullTime `db:"created_at"`
	ConfirmedAt sql.NullTime `db:"confirmed_at"`
	RejectedAt  sql.NullTime `db:"rejected_at"`
}

func readInvitations(ctx context.Context, db *sqlx.DB) ([]srcInvitation, error) {
	var out []srcInvitation
	q := `SELECT invited_by, invitee_user_id, domain_id, role_id, created_at, confirmed_at, rejected_at FROM invitations`
	return out, db.SelectContext(ctx, &out, q)
}
