// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"context"
	"time"

	"github.com/absmach/supermq"
	"github.com/absmach/supermq/consumers"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/messaging"
	mgjson "github.com/absmach/supermq/pkg/transformers/json"
	"github.com/absmach/supermq/pkg/transformers/senml"
	"github.com/vadv/gopher-lua-libs/argparse"
	"github.com/vadv/gopher-lua-libs/base64"
	"github.com/vadv/gopher-lua-libs/crypto"
	"github.com/vadv/gopher-lua-libs/db"
	"github.com/vadv/gopher-lua-libs/filepath"
	"github.com/vadv/gopher-lua-libs/ioutil"
	"github.com/vadv/gopher-lua-libs/json"
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
	idp    supermq.IDProvider
	repo   Repository
	pubSub messaging.PubSub
	errors chan error
	ticker Ticker
	email  Emailer
}

func NewService(repo Repository, idp supermq.IDProvider, pubSub messaging.PubSub, tck Ticker, emailer Emailer) Service {
	return &re{
		repo:   repo,
		idp:    idp,
		pubSub: pubSub,
		errors: make(chan error),
		ticker: tck,
		email:  emailer,
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
	var inputChannel string

	switch m := msgs.(type) {
	case *messaging.Message:
		inputChannel = m.Channel

	case []senml.Message:
		if len(m) == 0 {
			return
		}
		message := m[0]
		inputChannel = message.Channel
	case mgjson.Message:
		return
	default:
		return
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

	switch m := msg.(type) {
	case messaging.Message:
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

	case []senml.Message:
		msg := m[0]
		l.RawSet(message, lua.LString("channel"), lua.LString(msg.Channel))
		l.RawSet(message, lua.LString("subtopic"), lua.LString(msg.Subtopic))
		l.RawSet(message, lua.LString("publisher"), lua.LString(msg.Publisher))
		l.RawSet(message, lua.LString("protocol"), lua.LString(msg.Protocol))
		l.RawSet(message, lua.LString("name"), lua.LString(msg.Name))
		l.RawSet(message, lua.LString("unit"), lua.LString(msg.Unit))
		l.RawSet(message, lua.LString("time"), lua.LNumber(msg.Time))
		l.RawSet(message, lua.LString("update_time"), lua.LNumber(msg.UpdateTime))

		if msg.Value != nil {
			l.RawSet(message, lua.LString("value"), lua.LNumber(*msg.Value))
		}
		if msg.StringValue != nil {
			l.RawSet(message, lua.LString("string_value"), lua.LString(*msg.StringValue))
		}
		if msg.DataValue != nil {
			l.RawSet(message, lua.LString("data_value"), lua.LString(*msg.DataValue))
		}
		if msg.BoolValue != nil {
			l.RawSet(message, lua.LString("bool_value"), lua.LBool(*msg.BoolValue))
		}
		if msg.Sum != nil {
			l.RawSet(message, lua.LString("sum"), lua.LNumber(*msg.Sum))
		}
	}

	// Set the message object as a Lua global variable.
	l.SetGlobal("message", message)

	// set the email function as a Lua global function
	l.SetGlobal("send_email", l.NewFunction(re.sendEmail))

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
			Channel:   r.OutputChannel,
		}
		return re.pubSub.Publish(ctx, m.Channel, m)
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

func preload(l *lua.LState) {
	db.Preload(l)
	ioutil.Preload(l)
	json.Preload(l)
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
