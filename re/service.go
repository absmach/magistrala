// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"context"
	"time"

	"github.com/absmach/supermq"
	"github.com/absmach/supermq/consumers"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/messaging"
	mgjson "github.com/absmach/supermq/pkg/transformers/json"
	lua "github.com/yuin/gopher-lua"
)

type (
	ScriptType uint
	Metadata   map[string]interface{}
	Script     struct {
		Type  ScriptType `json:"type"`
		Value string     `json:"value"`
	}
)

// Type can be daily, weekly or monthly.
type ReccuringType uint

const (
	None ReccuringType = iota
	Daily
	Weekly
	Monthly
)

type Schedule struct {
	Time            []time.Time   `json:"date,omitempty"`
	RecurringType   ReccuringType `json:"recurring_type"`
	RecurringPeriod uint          `json:"recurring_period"` // 1 meaning every Recurring value, 2 meaning every other, and so on.
}

type Rule struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	DomainID      string    `json:"domain"`
	Metadata      Metadata  `json:"metadata,omitempty"`
	InputChannel  string    `json:"input_channel"`
	InputTopic    string    `json:"input_topic"`
	Logic         Script    `json:"logic"`
	OutputChannel string    `json:"output_channel,omitempty"`
	OutputTopic   string    `json:"output_topic,omitempty"`
	Schedule      Schedule  `json:"schedule,omitempty"`
	Status        Status    `json:"status"`
	CreatedAt     time.Time `json:"created_at,omitempty"`
	CreatedBy     string    `json:"created_by,omitempty"`
	UpdatedAt     time.Time `json:"updated_at,omitempty"`
	UpdatedBy     string    `json:"updated_by,omitempty"`
}

type Repository interface {
	AddRule(ctx context.Context, r Rule) (Rule, error)
	ViewRule(ctx context.Context, id string) (Rule, error)
	UpdateRule(ctx context.Context, r Rule) (Rule, error)
	RemoveRule(ctx context.Context, id string) error
	UpdateRuleStatus(ctx context.Context, id string, status Status) (Rule, error)
	ListRules(ctx context.Context, pm PageMeta) (Page, error)
}

// PageMeta contains page metadata that helps navigation.
type PageMeta struct {
	Total         uint64 `json:"total" db:"total"`
	Offset        uint64 `json:"offset" db:"offset"`
	Limit         uint64 `json:"limit" db:"limit"`
	Dir           string `json:"dir" db:"dir"`
	Name          string `json:"name" db:"name"`
	InputChannel  string `json:"input_channel,omitempty" db:"input_channel"`
	OutputChannel string `json:"output_channel,omitempty" db:"output_channel"`
	Status        Status `json:"status,omitempty" db:"status"`
	Domain        string `json:"domain_id,omitempty" db:"domain_id"`
}

type Page struct {
	PageMeta
	Rules []Rule `json:"rules"`
}

type Service interface {
	consumers.AsyncConsumer
	AddRule(ctx context.Context, session authn.Session, r Rule) (Rule, error)
	ViewRule(ctx context.Context, session authn.Session, id string) (Rule, error)
	UpdateRule(ctx context.Context, session authn.Session, r Rule) (Rule, error)
	ListRules(ctx context.Context, session authn.Session, pm PageMeta) (Page, error)
	RemoveRule(ctx context.Context, session authn.Session, id string) error
	EnableRule(ctx context.Context, session authn.Session, id string) (Rule, error)
	DisableRule(ctx context.Context, session authn.Session, id string) (Rule, error)
}

type re struct {
	idp    supermq.IDProvider
	repo   Repository
	pubSub messaging.PubSub
	errors chan error
}

func NewService(repo Repository, idp supermq.IDProvider, pubSub messaging.PubSub) Service {
	return &re{
		repo:   repo,
		idp:    idp,
		pubSub: pubSub,
		errors: make(chan error),
	}
}

func (re *re) AddRule(ctx context.Context, session authn.Session, r Rule) (Rule, error) {
	id, err := re.idp.ID()
	if err != nil {
		return Rule{}, err
	}
	r.CreatedAt = time.Now()
	r.ID = id
	r.CreatedBy = session.UserID
	r.DomainID = session.DomainID
	r.Status = EnabledStatus
	return re.repo.AddRule(ctx, r)
}

func (re *re) ViewRule(ctx context.Context, session authn.Session, id string) (Rule, error) {
	return re.repo.ViewRule(ctx, id)
}

func (re *re) UpdateRule(ctx context.Context, session authn.Session, r Rule) (Rule, error) {
	r.UpdatedAt = time.Now()
	r.UpdatedBy = session.UserID
	return re.repo.UpdateRule(ctx, r)
}

func (re *re) ListRules(ctx context.Context, session authn.Session, pm PageMeta) (Page, error) {
	pm.Domain = session.DomainID
	return re.repo.ListRules(ctx, pm)
}

func (re *re) RemoveRule(ctx context.Context, session authn.Session, id string) error {
	return re.repo.RemoveRule(ctx, id)
}

func (re *re) EnableRule(ctx context.Context, session authn.Session, id string) (Rule, error) {
	status, err := ToStatus(Enabled)
	if err != nil {
		return Rule{}, err
	}
	return re.repo.UpdateRuleStatus(ctx, id, status)
}

func (re *re) DisableRule(ctx context.Context, session authn.Session, id string) (Rule, error) {
	status, err := ToStatus(Disabled)
	if err != nil {
		return Rule{}, err
	}
	return re.repo.UpdateRuleStatus(ctx, id, status)
}

func (re *re) ConsumeAsync(ctx context.Context, msgs interface{}) {
	switch m := msgs.(type) {
	case *messaging.Message:
		pm := PageMeta{
			InputChannel: m.Channel,
			Status:       EnabledStatus,
		}
		page, err := re.repo.ListRules(ctx, pm)
		if err != nil {
			re.errors <- err
			return
		}
		for _, r := range page.Rules {
			go func(ctx context.Context) {
				re.errors <- re.process(ctx, r, m)
			}(ctx)
		}
	case mgjson.Message:
	default:
	}
}

func (re *re) Errors() <-chan error {
	return re.errors
}

func (re *re) process(ctx context.Context, r Rule, msg *messaging.Message) error {
	l := lua.NewState()
	defer l.Close()

	message := l.NewTable()

	l.RawSet(message, lua.LString("channel"), lua.LString(msg.Channel))
	l.RawSet(message, lua.LString("subtopic"), lua.LString(msg.Subtopic))
	l.RawSet(message, lua.LString("publisher"), lua.LString(msg.Publisher))
	l.RawSet(message, lua.LString("protocol"), lua.LString(msg.Protocol))
	l.RawSet(message, lua.LString("created"), lua.LNumber(msg.Created))

	pld := l.NewTable()
	for i, b := range msg.Payload {
		l.RawSet(pld, lua.LNumber(i+1), lua.LNumber(b)) // Lua tables are 1-indexed
	}
	l.RawSet(message, lua.LString("payload"), pld)

	// Set the message object as a Lua global variable.
	l.SetGlobal("message", message)

	if err := l.DoString(string(r.Logic.Value)); err != nil {
		return err
	}

	result := l.Get(-1) // Get the last result
	switch result {
	case lua.LNil:
		return nil
	default:
		if len(r.OutputChannel) == 0 {
			return nil
		}
		m := &messaging.Message{
			Publisher: "magistrala.re",
			Created:   time.Now().Unix(),
			Payload:   []byte(result.String()),
		}
		return re.pubSub.Publish(ctx, m.Channel, m)
	}
}
