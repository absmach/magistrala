// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/absmach/magistrala/pkg/schedule"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/messaging"
)

const (
	LuaType ScriptType = iota
	GoType
)

const protocol = "nats"

// ScriptOutput is the indicator for type of the logic
// so we can move it to the Go instead calling Go from Lua.
type ScriptOutput uint

const (
	Channels ScriptOutput = iota
	Alarms
	SaveSenML
	Email
	SaveRemotePg
)

var (
	scriptKindToString = [...]string{"channels", "alarms", "save_senml", "email", "save_remote_pg"}
	stringToScriptKind = map[string]ScriptOutput{
		"channels":       Channels,
		"alarms":         Alarms,
		"save_senml":     SaveSenML,
		"email":          Email,
		"save_remote_pg": SaveRemotePg,
	}
)

func (s ScriptOutput) String() string {
	if int(s) < 0 || int(s) >= len(scriptKindToString) {
		return "unknown"
	}
	return scriptKindToString[s]
}

// MarshalJSON converts ScriptOutput to JSON.
func (s ScriptOutput) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON parses JSON string into ScriptOutput.
func (s *ScriptOutput) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	lower := strings.ToLower(str)
	if val, ok := stringToScriptKind[lower]; ok {
		*s = val
		return nil
	}
	return errors.New("invalid ScriptOutput: " + str)
}

type (
	// ScriptType indicates Runtime type for the future versions
	// that will support JS or Go runtimes alongside Lua.
	ScriptType uint

	Metadata map[string]interface{}

	Script struct {
		Type    ScriptType     `json:"type"`
		Outputs []ScriptOutput `json:"outputs"`
		Value   string         `json:"value"`
	}
)

type Rule struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	DomainID      string            `json:"domain"`
	Metadata      Metadata          `json:"metadata,omitempty"`
	Tags          []string          `json:"tags,omitempty"`
	InputChannel  string            `json:"input_channel"`
	InputTopic    string            `json:"input_topic"`
	Logic         Script            `json:"logic"`
	OutputChannel string            `json:"output_channel,omitempty"`
	OutputTopic   string            `json:"output_topic,omitempty"`
	Schedule      schedule.Schedule `json:"schedule"`
	Status        Status            `json:"status"`
	CreatedAt     time.Time         `json:"created_at"`
	CreatedBy     string            `json:"created_by"`
	UpdatedAt     time.Time         `json:"updated_at"`
	UpdatedBy     string            `json:"updated_by"`
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
