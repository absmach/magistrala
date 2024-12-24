package re

import (
	"context"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/messaging"
	lua "github.com/yuin/gopher-lua"
)

type ScriptType uint

type Script struct {
	Type  ScriptType `json:"type"`
	Value string     `json:"value"`
}

// daily, weekly or monthly
type ReccuringType uint

const (
	Daily ReccuringType = iota
	Weekly
	Monthly
)

type Schedule struct {
	Time            []time.Time `json:"date,omitempty"`
	RecurringType   ReccuringType
	RecurringPeriod uint // 1 meaning every Recurring value, 2 meaning every other, and so on.
}

// Status represents Rule status.

type Rule struct {
	ID           string    `json:"id"`
	DomainID     string    `json:"domain"`
	InputTopic   string    `json:"input_topics"`
	Logic        Script    `json:"logic"`
	OutputTopics []string  `json:"output_topics,omitempty"`
	Schedule     Schedule  `json:"schedule,omitempty"`
	Status       Status    `json:"status"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
	CreatedBy    string    `json:"created_by,omitempty"`
	UpdatedAt    time.Time `json:"updated_at,omitempty"`
	UpdatedBy    string    `json:"updated_by,omitempty"`
}

type Repository interface {
	AddRule(ctx context.Context, r Rule) (Rule, error)
	ViewRule(ctx context.Context, id string) (Rule, error)
	UpdateRule(ctx context.Context, r Rule) (Rule, error)
	RemoveRule(ctx context.Context, id string) error
	ListRules(ctx context.Context, pm PageMeta) ([]Rule, error)
}

// PageMeta contains page metadata that helps navigation.
type PageMeta struct {
	Total      uint64 `json:"total"`
	Offset     uint64 `json:"offset"`
	Limit      uint64 `json:"limit"`
	InputTopic string `json:"input_topic,omitempty"`
	Status     Status `json:"status,omitempty"`
}

type Service interface {
	AddRule(ctx context.Context, session authn.Session, r Rule) (Rule, error)
	ViewRule(ctx context.Context, session authn.Session, id string) (Rule, error)
	UpdateRule(ctx context.Context, session authn.Session, r Rule) (Rule, error)
	ListRules(ctx context.Context, session authn.Session, pm PageMeta) ([]Rule, error)
	RemoveRule(ctx context.Context, session authn.Session, id string) error
}

type re struct {
	idp    magistrala.IDProvider
	repo   Repository
	cache  Repository
	pubSub messaging.PubSub
}

func NewService(repo, cache Repository, idp magistrala.IDProvider, pubSub messaging.PubSub) Service {
	return &re{
		repo:   repo,
		cache:  cache,
		idp:    idp,
		pubSub: pubSub,
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
	return re.repo.UpdateRule(ctx, r)
}

func (re *re) ListRules(ctx context.Context, session authn.Session, pm PageMeta) ([]Rule, error) {
	return re.repo.ListRules(ctx, pm)
}

func (re *re) RemoveRule(ctx context.Context, session authn.Session, id string) error {
	return re.repo.RemoveRule(ctx, id)
}

func (re *re) Process(ctx context.Context, session authn.Session) error {
	rls, _ := re.ListRules(ctx, session, PageMeta{})
	for _, r := range rls {
		l := lua.NewState()
		defer l.Close()
		if err := l.DoString(string(r.Logic.Value)); err != nil {
			panic(err)
		}
	}
	return nil
}
