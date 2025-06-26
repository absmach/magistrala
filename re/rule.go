// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"context"
	"encoding/json"
	"time"

	"github.com/absmach/magistrala/pkg/schedule"
	"github.com/absmach/magistrala/re/outputs"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/messaging"
)

const (
	LuaType ScriptType = iota
	GoType
)

type (
	// ScriptType indicates Runtime type for the future versions
	// that will support JS or Go runtimes alongside Lua.
	ScriptType uint

	Metadata map[string]interface{}
	Script   struct {
		Type  ScriptType `json:"type"`
		Value string     `json:"value"`
	}
)

var outputRegistry = map[outputs.OutputType]func() Runnable{
	outputs.AlarmsType:       func() Runnable { return &outputs.Alarm{} },
	outputs.EmailType:        func() Runnable { return &outputs.Email{} },
	outputs.SaveRemotePgType: func() Runnable { return &outputs.Postgres{} },
	outputs.ChannelsType:     func() Runnable { return &outputs.ChannelPublisher{} },
	outputs.SaveSenMLType:    func() Runnable { return &outputs.SenML{} },
}

type Rule struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	DomainID     string            `json:"domain"`
	Metadata     Metadata          `json:"metadata,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
	InputChannel string            `json:"input_channel"`
	InputTopic   string            `json:"input_topic"`
	Logic        Script            `json:"logic"`
	Outputs      Outputs           `json:"outputs,omitempty"`
	Schedule     schedule.Schedule `json:"schedule"`
	Status       Status            `json:"status"`
	CreatedAt    time.Time         `json:"created_at"`
	CreatedBy    string            `json:"created_by"`
	UpdatedAt    time.Time         `json:"updated_at"`
	UpdatedBy    string            `json:"updated_by"`
}

type Outputs []Runnable

func (o *Outputs) UnmarshalJSON(data []byte) error {
	var rawList []json.RawMessage
	if err := json.Unmarshal(data, &rawList); err != nil {
		return err
	}

	var runnables []Runnable
	for _, raw := range rawList {
		var meta struct {
			Type outputs.OutputType `json:"type"`
		}
		if err := json.Unmarshal(raw, &meta); err != nil {
			return err
		}

		factory, ok := outputRegistry[meta.Type]
		if !ok {
			return errors.New("unknown output type: " + meta.Type.String())
		}

		instance := factory()
		if err := json.Unmarshal(raw, instance); err != nil {
			return err
		}

		runnables = append(runnables, instance)
	}
	v := Outputs(runnables)
	*o = v
	return nil
}

type Runnable interface {
	Run(ctx context.Context, msg *messaging.Message, val interface{}) error
}

// PageMeta contains page metadata that helps navigation.
type PageMeta struct {
	Total           uint64              `json:"total" db:"total"`
	Offset          uint64              `json:"offset" db:"offset"`
	Limit           uint64              `json:"limit" db:"limit"`
	Dir             string              `json:"dir" db:"dir"`
	Name            string              `json:"name" db:"name"`
	InputChannel    string              `json:"input_channel,omitempty" db:"input_channel"`
	InputTopic      *string             `json:"input_topic,omitempty" db:"input_topic"`
	Scheduled       *bool               `json:"scheduled,omitempty"`
	OutputChannel   string              `json:"output_channel,omitempty" db:"output_channel"`
	Status          Status              `json:"status,omitempty" db:"status"`
	Domain          string              `json:"domain_id,omitempty" db:"domain_id"`
	Tag             string              `json:"tag,omitempty"`
	ScheduledBefore *time.Time          `json:"scheduled_before,omitempty" db:"scheduled_before"` // Filter rules scheduled before this time
	ScheduledAfter  *time.Time          `json:"scheduled_after,omitempty" db:"scheduled_after"`   // Filter rules scheduled after this time
	Recurring       *schedule.Recurring `json:"recurring,omitempty" db:"recurring"`               // Filter by recurring type
}

type Page struct {
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
	Total  uint64 `json:"total"`
	Rules  []Rule `json:"rules"`
}

type Service interface {
	messaging.MessageHandler
	AddRule(ctx context.Context, session authn.Session, r Rule) (Rule, error)
	ViewRule(ctx context.Context, session authn.Session, id string) (Rule, error)
	UpdateRule(ctx context.Context, session authn.Session, r Rule) (Rule, error)
	UpdateRuleTags(ctx context.Context, session authn.Session, r Rule) (Rule, error)
	UpdateRuleSchedule(ctx context.Context, session authn.Session, r Rule) (Rule, error)
	ListRules(ctx context.Context, session authn.Session, pm PageMeta) (Page, error)
	RemoveRule(ctx context.Context, session authn.Session, id string) error
	EnableRule(ctx context.Context, session authn.Session, id string) (Rule, error)
	DisableRule(ctx context.Context, session authn.Session, id string) (Rule, error)

	StartScheduler(ctx context.Context) error
}

type Repository interface {
	AddRule(ctx context.Context, r Rule) (Rule, error)
	ViewRule(ctx context.Context, id string) (Rule, error)
	UpdateRule(ctx context.Context, r Rule) (Rule, error)
	UpdateRuleTags(ctx context.Context, r Rule) (Rule, error)
	UpdateRuleSchedule(ctx context.Context, r Rule) (Rule, error)
	RemoveRule(ctx context.Context, id string) error
	UpdateRuleStatus(ctx context.Context, r Rule) (Rule, error)
	ListRules(ctx context.Context, pm PageMeta) (Page, error)
	UpdateRuleDue(ctx context.Context, id string, due time.Time) (Rule, error)
}
