// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"context"
	"time"

	"github.com/absmach/supermq"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/messaging"
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
	messaging.MessageHandler
	AddRule(ctx context.Context, session authn.Session, r Rule) (Rule, error)
	ViewRule(ctx context.Context, session authn.Session, id string) (Rule, error)
	UpdateRule(ctx context.Context, session authn.Session, r Rule) (Rule, error)
	ListRules(ctx context.Context, session authn.Session, pm PageMeta) (Page, error)
	RemoveRule(ctx context.Context, session authn.Session, id string) error
	EnableRule(ctx context.Context, session authn.Session, id string) (Rule, error)
	DisableRule(ctx context.Context, session authn.Session, id string) (Rule, error)
	StartScheduler(ctx context.Context) error
	Errors() <-chan error
}

type re struct {
	writersPub messaging.Publisher
	alarmsPub  messaging.Publisher
	rePubSub   messaging.PubSub
	idp        supermq.IDProvider
	repo       Repository
	errors     chan error
	ticker     Ticker
	email      Emailer
}

func NewService(repo Repository, idp supermq.IDProvider, rePubSub messaging.PubSub, writersPub, alarmsPub messaging.Publisher, tck Ticker, emailer Emailer) Service {
	return &re{
		writersPub: writersPub,
		alarmsPub:  alarmsPub,
		rePubSub:   rePubSub,
		repo:       repo,
		idp:        idp,
		errors:     make(chan error),
		ticker:     tck,
		email:      emailer,
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

func (re *re) Handle(msg *messaging.Message) error {
	inputChannel := msg.Channel

	pm := PageMeta{
		InputChannel: inputChannel,
		Status:       EnabledStatus,
	}
	ctx := context.Background()
	page, err := re.repo.ListRules(ctx, pm)
	if err != nil {
		return err
	}

	for _, r := range page.Rules {
		go func(ctx context.Context) {
			if err := re.process(ctx, r, msg); err != nil {
				re.errors <- err
			}
		}(ctx)
	}
	return nil
}

func (re *re) Cancel() error {
	return nil
}

func (re *re) Errors() <-chan error {
	return re.errors
}

func (re *re) process(ctx context.Context, r Rule, msg *messaging.Message) error {
	l := lua.NewState()
	defer l.Close()
	preload(l)

	message := prepareMsg(l, msg)

	// Set the message object as a Lua global variable.
	l.SetGlobal("message", message)

	// set the email function as a Lua global function.
	l.SetGlobal("send_email", l.NewFunction(re.sendEmail))
	l.SetGlobal("save_senml", l.NewFunction(re.save(ctx, msg)))

	if err := l.DoString(string(r.Logic.Value)); err != nil {
		return err
	}

	result := l.Get(-1) // Get the last result.
	switch result {
	case lua.LNil:
		return nil
	default:
		if r.OutputChannel == "" {
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
