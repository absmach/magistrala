// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package alarms

import (
	"context"
	"time"

	"github.com/absmach/supermq/pkg/authn"
)

type Metadata map[string]interface{}

// Rule defines conditions that trigger an alarm
type Rule struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	UserID    string    `json:"user_id"`
	DomainID  string    `json:"domain_id"`
	Condition string    `json:"condition"` // E.g. "temperature > 30"
	Channel   string    `json:"channel"`   // Channel to monitor
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	UpdatedBy string    `json:"updated_by,omitempty"`
	Metadata  Metadata  `json:"metadata,omitempty"`
}

type RulesPage struct {
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
	Total  uint64 `json:"total"`
	Rules  []Rule `json:"rules"`
}

// Alarm represents an alarm instance
type Alarm struct {
	ID         string    `json:"id"`
	RuleID     string    `json:"rule_id"`
	Message    string    `json:"message"`
	Status     Status    `json:"status"`
	UserID     string    `json:"user_id"`
	DomainID   string    `json:"domain_id"`
	AssigneeID string    `json:"assignee_id"`
	CreatedAt  time.Time `json:"created_at"`
	CreatedBy  string    `json:"created_by"`
	UpdatedAt  time.Time `json:"updated_at"`
	UpdatedBy  string    `json:"updated_by"`
	ResolvedAt time.Time `json:"resolved_at,omitempty"`
	ResolvedBy string    `json:"resolved_by,omitempty"`
	Metadata   Metadata  `json:"metadata,omitempty"`
}

type AlarmsPage struct {
	Offset uint64  `json:"offset"`
	Limit  uint64  `json:"limit"`
	Total  uint64  `json:"total"`
	Alarms []Alarm `json:"alarms"`
}

type PageMetadata struct {
	Offset     uint64 `json:"offset"      db:"offset"`
	Limit      uint64 `json:"limit"       db:"limit"`
	UserID     string `json:"user_id"     db:"user_id"`
	DomainID   string `json:"domain_id"   db:"domain_id"`
	ChannelID  string `json:"channel_id"  db:"channel_id"`
	RuleID     string `json:"rule_id"     db:"rule_id"`
	Status     Status `json:"status"      db:"status"`
	AssigneeID string `json:"assignee_id" db:"assignee_id"`
}

// Service specifies an API that must be fulfilled by the domain service
type Service interface {
	CreateRule(ctx context.Context, session authn.Session, rule Rule) (Rule, error)
	UpdateRule(ctx context.Context, session authn.Session, rule Rule) (Rule, error)
	ViewRule(ctx context.Context, session authn.Session, id string) (Rule, error)
	ListRules(ctx context.Context, session authn.Session, pm PageMetadata) (RulesPage, error)
	DeleteRule(ctx context.Context, session authn.Session, id string) error

	CreateAlarm(ctx context.Context, session authn.Session, alarm Alarm) (Alarm, error)
	UpdateAlarm(ctx context.Context, session authn.Session, alarm Alarm) (Alarm, error)
	ViewAlarm(ctx context.Context, session authn.Session, id string) (Alarm, error)
	ListAlarms(ctx context.Context, session authn.Session, pm PageMetadata) (AlarmsPage, error)
	DeleteAlarm(ctx context.Context, session authn.Session, id string) error

	AssignAlarm(ctx context.Context, session authn.Session, alarm Alarm) error
}

type Repository interface {
	CreateRule(ctx context.Context, rule Rule) (Rule, error)
	UpdateRule(ctx context.Context, rule Rule) (Rule, error)
	ViewRule(ctx context.Context, id string) (Rule, error)
	ListRules(ctx context.Context, pm PageMetadata) (RulesPage, error)
	DeleteRule(ctx context.Context, id string) error

	CreateAlarm(ctx context.Context, alarm Alarm) (Alarm, error)
	UpdateAlarm(ctx context.Context, alarm Alarm) (Alarm, error)
	ViewAlarm(ctx context.Context, id string) (Alarm, error)
	ListAlarms(ctx context.Context, pm PageMetadata) (AlarmsPage, error)
	DeleteAlarm(ctx context.Context, id string) error

	AssignAlarm(ctx context.Context, alarm Alarm) error
}
