// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"context"
	"encoding/json"
	"time"

	"github.com/absmach/supermq"
	"github.com/absmach/supermq/consumers"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/transformers"
	mgjson "github.com/absmach/supermq/pkg/transformers/json"
	"github.com/absmach/supermq/pkg/transformers/senml"
	mgsenml "github.com/absmach/supermq/pkg/transformers/senml"
	"github.com/vadv/gopher-lua-libs/argparse"
	"github.com/vadv/gopher-lua-libs/base64"
	"github.com/vadv/gopher-lua-libs/crypto"
	"github.com/vadv/gopher-lua-libs/db"
	"github.com/vadv/gopher-lua-libs/filepath"
	"github.com/vadv/gopher-lua-libs/ioutil"
	luajson "github.com/vadv/gopher-lua-libs/json"
	"github.com/vadv/gopher-lua-libs/regexp"
	"github.com/vadv/gopher-lua-libs/storage"
	"github.com/vadv/gopher-lua-libs/strings"
	luatime "github.com/vadv/gopher-lua-libs/time"
	"github.com/vadv/gopher-lua-libs/yaml"
	lua "github.com/yuin/gopher-lua"
)

const (
	hoursInDay   = 24
	daysInWeek   = 7
	monthsInYear = 12

	publisher = "magistrala.re"
)

var ErrInvalidRecurringType = errors.New("invalid recurring type")

type (
	ScriptType uint
	Metadata   map[string]interface{}
	Script     struct {
		Type  ScriptType `json:"type"`
		Value string     `json:"value"`
	}
)

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
	Total           uint64     `json:"total" db:"total"`
	Offset          uint64     `json:"offset" db:"offset"`
	Limit           uint64     `json:"limit" db:"limit"`
	Dir             string     `json:"dir" db:"dir"`
	Name            string     `json:"name" db:"name"`
	InputChannel    string     `json:"input_channel,omitempty" db:"input_channel"`
	OutputChannel   string     `json:"output_channel,omitempty" db:"output_channel"`
	Status          Status     `json:"status,omitempty" db:"status"`
	Domain          string     `json:"domain_id,omitempty" db:"domain_id"`
	ScheduledBefore *time.Time `json:"scheduled_before,omitempty" db:"scheduled_before"` // Filter rules scheduled before this time
	ScheduledAfter  *time.Time `json:"scheduled_after,omitempty" db:"scheduled_after"`   // Filter rules scheduled after this time
	Recurring       *Recurring `json:"recurring,omitempty" db:"recurring"`               // Filter by recurring type
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
	StartScheduler(ctx context.Context) error
}

type re struct {
	writersPubSub messaging.PubSub
	alarmsPubSub  messaging.PubSub
	rePubSub      messaging.PubSub
	idp           supermq.IDProvider
	repo          Repository
	errors        chan error
	ticker        Ticker
	email         Emailer
	ts            []transformers.Transformer
}

func NewService(repo Repository, idp supermq.IDProvider, rePubSub messaging.PubSub, writersPubSub messaging.PubSub, alarmsPubSub messaging.PubSub, tck Ticker, emailer Emailer) Service {
	return &re{
		writersPubSub: writersPubSub,
		alarmsPubSub:  alarmsPubSub,
		rePubSub:      rePubSub,
		repo:          repo,
		idp:           idp,
		errors:        make(chan error),
		ticker:        tck,
		email:         emailer,
		// Transformers order is important since SenML is also JSON content type.
		ts: []transformers.Transformer{
			mgsenml.New(mgsenml.JSON),
			mgjson.New(nil),
		},
	}
}

func (re *re) AddRule(ctx context.Context, session authn.Session, r Rule) (Rule, error) {
	id, err := re.idp.ID()
	if err != nil {
		return Rule{}, err
	}
	now := time.Now()
	r.CreatedAt = now
	r.ID = id
	r.CreatedBy = session.UserID
	r.DomainID = session.DomainID
	r.Status = EnabledStatus

	if r.Schedule.StartDateTime.IsZero() {
		r.Schedule.StartDateTime = now
	}

	rule, err := re.repo.AddRule(ctx, r)
	if err != nil {
		return Rule{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	return rule, nil
}

func (re *re) ViewRule(ctx context.Context, session authn.Session, id string) (Rule, error) {
	rule, err := re.repo.ViewRule(ctx, id)
	if err != nil {
		return Rule{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	return rule, nil
}

func (re *re) UpdateRule(ctx context.Context, session authn.Session, r Rule) (Rule, error) {
	r.UpdatedAt = time.Now()
	r.UpdatedBy = session.UserID
	rule, err := re.repo.UpdateRule(ctx, r)
	if err != nil {
		return Rule{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return rule, nil
}

func (re *re) ListRules(ctx context.Context, session authn.Session, pm PageMeta) (Page, error) {
	pm.Domain = session.DomainID
	page, err := re.repo.ListRules(ctx, pm)
	if err != nil {
		return Page{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return page, nil
}

func (re *re) RemoveRule(ctx context.Context, session authn.Session, id string) error {
	if err := re.repo.RemoveRule(ctx, id); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return nil
}

func (re *re) EnableRule(ctx context.Context, session authn.Session, id string) (Rule, error) {
	status, err := ToStatus(Enabled)
	if err != nil {
		return Rule{}, err
	}
	rule, err := re.repo.UpdateRuleStatus(ctx, id, status)
	if err != nil {
		return Rule{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return rule, nil
}

func (re *re) DisableRule(ctx context.Context, session authn.Session, id string) (Rule, error) {
	status, err := ToStatus(Disabled)
	if err != nil {
		return Rule{}, err
	}
	rule, err := re.repo.UpdateRuleStatus(ctx, id, status)
	if err != nil {
		return Rule{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return rule, nil
}

func (re *re) ConsumeAsync(ctx context.Context, msgs interface{}) {
	m, ok := msgs.(*messaging.Message)
	if !ok {
		return
	}
	inputChannel := m.Channel
	for _, t := range re.ts {
		if v, err := t.Transform(m); err == nil {
			msgs = v
			break
		}
	}

	pm := PageMeta{
		InputChannel: inputChannel,
		Status:       EnabledStatus,
	}
	page, err := re.repo.ListRules(ctx, pm)
	if err != nil {
		re.errors <- err
		return
	}

	for _, r := range page.Rules {
		go func(ctx context.Context) {
			re.errors <- re.process(ctx, r, msgs)
		}(ctx)
	}
}

func (re *re) Errors() <-chan error {
	return re.errors
}

func (re *re) process(ctx context.Context, r Rule, msg interface{}) error {
	l := lua.NewState()
	defer l.Close()
	preload(l)

	message := l.NewTable()
	messages := l.NewTable()

	switch m := msg.(type) {
	case *messaging.Message:
		{
			l.RawSet(message, lua.LString("channel"), lua.LString(m.Channel))
			l.RawSet(message, lua.LString("subtopic"), lua.LString(m.Subtopic))
			l.RawSet(message, lua.LString("publisher"), lua.LString(m.Publisher))
			l.RawSet(message, lua.LString("protocol"), lua.LString(m.Protocol))
			l.RawSet(message, lua.LString("created"), lua.LNumber(m.Created))

			pld := l.NewTable()
			for i, b := range m.Payload {
				l.RawSet(pld, lua.LNumber(i+1), lua.LNumber(b)) // Lua tables are 1-indexed
			}
			l.RawSet(message, lua.LString("payload"), pld)
		}

	case []mgsenml.Message:
		for i, msg := range m {
			insert := l.NewTable()
			insert.RawSetString("channel", lua.LString(msg.Channel))
			insert.RawSetString("subtopic", lua.LString(msg.Subtopic))
			insert.RawSetString("publisher", lua.LString(msg.Publisher))
			insert.RawSetString("protocol", lua.LString(msg.Protocol))
			insert.RawSetString("name", lua.LString(msg.Name))
			insert.RawSetString("unit", lua.LString(msg.Unit))
			insert.RawSetString("time", lua.LNumber(msg.Time))
			insert.RawSetString("update_time", lua.LNumber(msg.UpdateTime))

			if msg.Value != nil {
				insert.RawSetString("value", lua.LNumber(*msg.Value))
			}
			if msg.StringValue != nil {
				insert.RawSetString("string_value", lua.LString(*msg.StringValue))
			}
			if msg.DataValue != nil {
				insert.RawSetString("data_value", lua.LString(*msg.DataValue))
			}
			if msg.BoolValue != nil {
				insert.RawSetString("bool_value", lua.LBool(*msg.BoolValue))
			}
			if msg.Sum != nil {
				insert.RawSetString("sum", lua.LNumber(*msg.Sum))
			}
			messages.RawSetInt(i+1, insert) // Lua index starts at 1.
		}
		if len(m) == 1 {
			message = messages.RawGetInt(1).(*lua.LTable)
		}

	case mgjson.Messages:
		for i, msg := range m.Data {
			insert := l.NewTable()
			insert.RawSetString("channel", lua.LString(msg.Channel))
			insert.RawSetString("subtopic", lua.LString(msg.Subtopic))
			insert.RawSetString("publisher", lua.LString(msg.Publisher))
			insert.RawSetString("protocol", lua.LString(msg.Protocol))
			insert.RawSetString("format", lua.LString(m.Format))
			traverseJson(l, insert, lua.LString("payload"), map[string]interface{}(msg.Payload))
			messages.RawSetInt(i+1, insert) // Lua index starts at 1.
		}
		if len(m.Data) == 1 {
			message = messages.RawGetInt(1).(*lua.LTable)
		}
	}

	// Set the message object as a Lua global variable.
	l.SetGlobal("message", message)
	l.SetGlobal("messages", messages)
	l.SetGlobal("domain_id", lua.LString(r.DomainID))

	// set the email function as a Lua global function
	l.SetGlobal("send_email", l.NewFunction(re.sendEmail))
	l.SetGlobal("save_senml", l.NewFunction(re.saveSenml))

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
			Publisher: publisher,
			Created:   time.Now().Unix(),
			Payload:   []byte(result.String()),
			Channel:   r.OutputChannel,
			Domain:    r.DomainID,
			Subtopic:  r.OutputTopic,
		}
		return re.rePubSub.Publish(ctx, m.Channel, m)
	}
}

func (re *re) StartScheduler(ctx context.Context) error {
	defer re.ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-re.ticker.Tick():
			startTime := time.Now()

			pm := PageMeta{
				Status:          EnabledStatus,
				ScheduledBefore: &startTime,
			}

			page, err := re.repo.ListRules(ctx, pm)
			if err != nil {
				return err
			}

			for _, rule := range page.Rules {
				if rule.shouldRun(startTime) {
					go func(r Rule) {
						msg := &messaging.Message{
							Channel: r.InputChannel,
							Created: startTime.Unix(),
						}
						re.errors <- re.process(ctx, r, msg)
					}(rule)
				}
			}
		}
	}
}

func (r Rule) shouldRun(startTime time.Time) bool {
	// Don't run if the rule's start time is in the future
	// This allows scheduling rules to start at a specific future time
	if r.Schedule.StartDateTime.After(startTime) {
		return false
	}

	t := r.Schedule.Time.Truncate(time.Minute).UTC()
	startTimeOnly := time.Date(0, 1, 1, startTime.Hour(), startTime.Minute(), 0, 0, time.UTC)
	if t.Equal(startTimeOnly) {
		return true
	}

	if r.Schedule.RecurringPeriod == 0 {
		return false
	}

	period := int(r.Schedule.RecurringPeriod)

	switch r.Schedule.Recurring {
	case Daily:
		if r.Schedule.RecurringPeriod > 0 {
			daysSinceStart := startTime.Sub(r.Schedule.StartDateTime).Hours() / hoursInDay
			if int(daysSinceStart)%period == 0 {
				return true
			}
		}
	case Weekly:
		if r.Schedule.RecurringPeriod > 0 {
			weeksSinceStart := startTime.Sub(r.Schedule.StartDateTime).Hours() / (hoursInDay * daysInWeek)
			if int(weeksSinceStart)%period == 0 {
				return true
			}
		}
	case Monthly:
		if r.Schedule.RecurringPeriod > 0 {
			monthsSinceStart := (startTime.Year()-r.Schedule.StartDateTime.Year())*monthsInYear +
				int(startTime.Month()-r.Schedule.StartDateTime.Month())
			if monthsSinceStart%period == 0 {
				return true
			}
		}
	}

	return false
}

func (re *re) sendEmail(L *lua.LState) int {
	recipientsTable := L.ToTable(1)
	subject := L.ToString(2)
	content := L.ToString(3)

	var recipients []string
	recipientsTable.ForEach(func(_, value lua.LValue) {
		if str, ok := value.(lua.LString); ok {
			recipients = append(recipients, string(str))
		}
	})

	if err := re.email.SendEmailNotification(recipients, "", subject, "", "", content, ""); err != nil {
		return 0
	}
	return 1
}

func (re *re) saveSenml(L *lua.LState) int {
	luaString := L.ToString(1)
	var message senml.Message

	if err := json.Unmarshal([]byte(luaString), &message); err != nil {
		return 0
	}

	payload, err := json.Marshal(message)
	if err != nil {
		return 0
	}

	domainId := L.GetGlobal("domain_id").String()

	ctx := context.Background()
	m := &messaging.Message{
		Publisher: message.Publisher,
		Created:   time.Now().Unix(),
		Payload:   payload,
		Channel:   message.Channel,
		Subtopic:  message.Subtopic,
		Domain:    domainId,
	}

	if err := re.writersPubSub.Publish(ctx, message.Channel, m); err != nil {
		return 0
	}
	return 1
}

func preload(l *lua.LState) {
	db.Preload(l)
	ioutil.Preload(l)
	luajson.Preload(l)
	yaml.Preload(l)
	crypto.Preload(l)
	regexp.Preload(l)
	luatime.Preload(l)
	storage.Preload(l)
	base64.Preload(l)
	argparse.Preload(l)
	strings.Preload(l)
	filepath.Preload(l)
}

func traverseJson(l *lua.LState, parent *lua.LTable, key lua.LValue, value interface{}) {
	var lval lua.LValue
	switch val := value.(type) {
	case string:
		lval = lua.LString(val)
	case float64:
		lval = lua.LNumber(val)
	case int:
		lval = lua.LNumber(float64(val))
	case json.Number:
		if num, err := val.Float64(); err != nil {
			lval = lua.LNumber(num)
		}
	case bool:
		lval = lua.LBool(val)
	case []interface{}:
		t := l.NewTable()
		for i, j := range val {
			traverseJson(l, t, lua.LNumber(i+1), j)
		}
		lval = t
	case map[string]interface{}:
		t := l.NewTable()
		for k, v := range val {
			traverseJson(l, t, lua.LString(k), v)
		}
		lval = t
	case []map[string]interface{}:
		t := l.NewTable()
		for i, j := range val {
			traverseJson(l, t, lua.LNumber(i+1), j)
		}
		lval = t
	}
	parent.RawSet(key, lval)
	return
}
