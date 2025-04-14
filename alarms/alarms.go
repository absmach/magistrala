// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package alarms

import (
	"context"
	"errors"
	"time"

	"github.com/absmach/supermq/pkg/authn"
)

const SeverityMax uint8 = 100

var ErrInvalidSeverity = errors.New("invalid severity. Must be between 0 and 100")

type Metadata map[string]interface{}

// Alarm represents an alarm instance.
type Alarm struct {
	ID             string    `json:"id"`
	RuleID         string    `json:"rule_id"`
	DomainID       string    `json:"domain_id"`
	ChannelID      string    `json:"channel_id"`
	ClientID       string    `json:"client_id"`
	Subtopic       string    `json:"subtopic"`
	Status         Status    `json:"status"`
	Measurement    string    `json:"measurement"`
	Value          string    `json:"value"`
	Unit           string    `json:"unit"`
	Threshold      string    `json:"threshold"`
	Cause          string    `json:"cause"`
	Severity       uint8     `json:"severity"`
	AssigneeID     string    `json:"assignee_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	UpdatedBy      string    `json:"updated_by"`
	AssignedAt     time.Time `json:"assigned_at,omitempty"`
	AssignedBy     string    `json:"assigned_by,omitempty"`
	AcknowledgedAt time.Time `json:"acknowledged_at,omitempty"`
	AcknowledgedBy string    `json:"acknowledged_by,omitempty"`
	ResolvedAt     time.Time `json:"resolved_at,omitempty"`
	ResolvedBy     string    `json:"resolved_by,omitempty"`
	Metadata       Metadata  `json:"metadata,omitempty"`
}

type AlarmsPage struct {
	Offset uint64  `json:"offset"`
	Limit  uint64  `json:"limit"`
	Total  uint64  `json:"total"`
	Alarms []Alarm `json:"alarms"`
}

type PageMetadata struct {
	Offset         uint64 `json:"offset"          db:"offset"`
	Limit          uint64 `json:"limit"           db:"limit"`
	DomainID       string `json:"domain_id"       db:"domain_id"`
	ChannelID      string `json:"channel_id"      db:"channel_id"`
	ClientID       string `json:"client_id"       db:"client_id"`
	Subtopic       string `json:"subtopic"        db:"subtopic"`
	RuleID         string `json:"rule_id"         db:"rule_id"`
	Status         Status `json:"status"          db:"status"`
	AssigneeID     string `json:"assignee_id"     db:"assignee_id"`
	Severity       uint8  `json:"severity"        db:"severity"`
	UpdatedBy      string `json:"updated_by"      db:"updated_by"`
	AssignedBy     string `json:"assigned_by"     db:"assigned_by"`
	AcknowledgedBy string `json:"acknowledged_by" db:"acknowledged_by"`
	ResolvedBy     string `json:"resolved_by"     db:"resolved_by"`
}

func (a Alarm) Validate() error {
	if a.RuleID == "" {
		return errors.New("rule_id is required")
	}
	if a.DomainID == "" {
		return errors.New("domain_id is required")
	}
	if a.ChannelID == "" {
		return errors.New("channel_id is required")
	}
	if a.ClientID == "" {
		return errors.New("client_id is required")
	}
	if a.Subtopic == "" {
		return errors.New("subtopic is required")
	}
	if a.Measurement == "" {
		return errors.New("measurement is required")
	}
	if a.Value == "" {
		return errors.New("value is required")
	}
	if a.Unit == "" {
		return errors.New("unit is required")
	}
	if a.Cause == "" {
		return errors.New("cause is required")
	}
	if a.Severity > SeverityMax {
		return ErrInvalidSeverity
	}

	return nil
}

// Service specifies an API that must be fulfilled by the domain service.
type Service interface {
	CreateAlarm(ctx context.Context, alarm Alarm) error
	UpdateAlarm(ctx context.Context, session authn.Session, alarm Alarm) (Alarm, error)
	ViewAlarm(ctx context.Context, session authn.Session, id string) (Alarm, error)
	ListAlarms(ctx context.Context, session authn.Session, pm PageMetadata) (AlarmsPage, error)
	DeleteAlarm(ctx context.Context, session authn.Session, id string) error
}

type Repository interface {
	CreateAlarm(ctx context.Context, alarm Alarm) (Alarm, error)
	UpdateAlarm(ctx context.Context, alarm Alarm) (Alarm, error)
	ViewAlarm(ctx context.Context, alarmID, domainID string) (Alarm, error)
	ListAlarms(ctx context.Context, pm PageMetadata) (AlarmsPage, error)
	DeleteAlarm(ctx context.Context, id string) error
}
