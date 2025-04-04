// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package alarms

import (
	"context"
	"time"

	"github.com/absmach/supermq/pkg/authn"
)

type Metadata map[string]interface{}

// Alarm represents an alarm instance
type Alarm struct {
	ID         string    `json:"id"`
	RuleID     string    `json:"rule_id"`
	Message    string    `json:"message"`
	Status     Status    `json:"status"`
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
	DomainID   string `json:"domain_id"   db:"domain_id"`
	ChannelID  string `json:"channel_id"  db:"channel_id"`
	RuleID     string `json:"rule_id"     db:"rule_id"`
	Status     Status `json:"status"      db:"status"`
	AssigneeID string `json:"assignee_id" db:"assignee_id"`
}

// Service specifies an API that must be fulfilled by the domain service
type Service interface {
	CreateAlarm(ctx context.Context, session authn.Session, alarm Alarm) (Alarm, error)
	UpdateAlarm(ctx context.Context, session authn.Session, alarm Alarm) (Alarm, error)
	ViewAlarm(ctx context.Context, session authn.Session, id string) (Alarm, error)
	ListAlarms(ctx context.Context, session authn.Session, pm PageMetadata) (AlarmsPage, error)
	DeleteAlarm(ctx context.Context, session authn.Session, id string) error
}

type Repository interface {
	CreateAlarm(ctx context.Context, alarm Alarm) (Alarm, error)
	UpdateAlarm(ctx context.Context, alarm Alarm) (Alarm, error)
	ViewAlarm(ctx context.Context, id string) (Alarm, error)
	ListAlarms(ctx context.Context, pm PageMetadata) (AlarmsPage, error)
	DeleteAlarm(ctx context.Context, id string) error
}
