// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"context"
	"time"

	grpcReadersV1 "github.com/absmach/magistrala/api/grpc/readers/v1"
	"github.com/absmach/magistrala/pkg/reltime"
	"github.com/absmach/supermq"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/transformers/senml"
)

const limit = 1000

type Repository interface {
	AddRule(ctx context.Context, r Rule) (Rule, error)
	ViewRule(ctx context.Context, id string) (Rule, error)
	UpdateRule(ctx context.Context, r Rule) (Rule, error)
	UpdateRuleSchedule(ctx context.Context, r Rule) (Rule, error)
	RemoveRule(ctx context.Context, id string) error
	UpdateRuleStatus(ctx context.Context, r Rule) (Rule, error)
	ListRules(ctx context.Context, pm PageMeta) (Page, error)

	AddReportConfig(ctx context.Context, cfg ReportConfig) (ReportConfig, error)
	ViewReportConfig(ctx context.Context, id string) (ReportConfig, error)
	UpdateReportConfig(ctx context.Context, cfg ReportConfig) (ReportConfig, error)
	UpdateReportSchedule(ctx context.Context, cfg ReportConfig) (ReportConfig, error)
	RemoveReportConfig(ctx context.Context, id string) error
	UpdateReportConfigStatus(ctx context.Context, cfg ReportConfig) (ReportConfig, error)
	ListReportsConfig(ctx context.Context, pm PageMeta) (ReportConfigPage, error)
}

// PageMeta contains page metadata that helps navigation.
type PageMeta struct {
	Total           uint64     `json:"total" db:"total"`
	Offset          uint64     `json:"offset" db:"offset"`
	Limit           uint64     `json:"limit" db:"limit"`
	Dir             string     `json:"dir" db:"dir"`
	Name            string     `json:"name" db:"name"`
	InputChannel    string     `json:"input_channel,omitempty" db:"input_channel"`
	InputTopic      *string    `json:"input_topic,omitempty" db:"input_topic"`
	OutputChannel   string     `json:"output_channel,omitempty" db:"output_channel"`
	Status          Status     `json:"status,omitempty" db:"status"`
	Domain          string     `json:"domain_id,omitempty" db:"domain_id"`
	ScheduledBefore *time.Time `json:"scheduled_before,omitempty" db:"scheduled_before"` // Filter rules scheduled before this time
	ScheduledAfter  *time.Time `json:"scheduled_after,omitempty" db:"scheduled_after"`   // Filter rules scheduled after this time
	Recurring       *Recurring `json:"recurring,omitempty" db:"recurring"`               // Filter by recurring type
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
	UpdateRuleSchedule(ctx context.Context, session authn.Session, r Rule) (Rule, error)
	ListRules(ctx context.Context, session authn.Session, pm PageMeta) (Page, error)
	RemoveRule(ctx context.Context, session authn.Session, id string) error
	EnableRule(ctx context.Context, session authn.Session, id string) (Rule, error)
	DisableRule(ctx context.Context, session authn.Session, id string) (Rule, error)

	AddReportConfig(ctx context.Context, session authn.Session, cfg ReportConfig) (ReportConfig, error)
	ViewReportConfig(ctx context.Context, session authn.Session, id string) (ReportConfig, error)
	UpdateReportConfig(ctx context.Context, session authn.Session, cfg ReportConfig) (ReportConfig, error)
	UpdateReportSchedule(ctx context.Context, session authn.Session, cfg ReportConfig) (ReportConfig, error)
	RemoveReportConfig(ctx context.Context, session authn.Session, id string) error
	ListReportsConfig(ctx context.Context, session authn.Session, pm PageMeta) (ReportConfigPage, error)
	EnableReportConfig(ctx context.Context, session authn.Session, id string) (ReportConfig, error)
	DisableReportConfig(ctx context.Context, session authn.Session, id string) (ReportConfig, error)

	GenerateReport(ctx context.Context, session authn.Session, config ReportConfig, download bool) (ReportPage, error)
	StartScheduler(ctx context.Context) error
}

type re struct {
	repo       Repository
	errors     chan error
	idp        supermq.IDProvider
	rePubSub   messaging.PubSub
	writersPub messaging.Publisher
	alarmsPub  messaging.Publisher
	ticker     Ticker
	email      Emailer
	readers    grpcReadersV1.ReadersServiceClient
}

func NewService(repo Repository, errors chan (error), idp supermq.IDProvider, rePubSub messaging.PubSub, writersPub, alarmsPub messaging.Publisher, tck Ticker, emailer Emailer, readers grpcReadersV1.ReadersServiceClient) Service {
	return &re{
		repo:       repo,
		idp:        idp,
		errors:     errors,
		rePubSub:   rePubSub,
		writersPub: writersPub,
		alarmsPub:  alarmsPub,
		ticker:     tck,
		email:      emailer,
		readers:    readers,
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

func (re *re) UpdateRuleSchedule(ctx context.Context, session authn.Session, r Rule) (Rule, error) {
	r.UpdatedAt = time.Now()
	r.UpdatedBy = session.UserID
	rule, err := re.repo.UpdateRuleSchedule(ctx, r)
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
	r := Rule{
		ID:        id,
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
		Status:    status,
	}
	rule, err := re.repo.UpdateRuleStatus(ctx, r)
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
	r := Rule{
		ID:        id,
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
		Status:    status,
	}
	rule, err := re.repo.UpdateRuleStatus(ctx, r)
	if err != nil {
		return Rule{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return rule, nil
}

func (re *re) Cancel() error {
	return nil
}

func (re *re) Errors() <-chan error {
	return re.errors
}

func (re *re) AddReportConfig(ctx context.Context, session authn.Session, cfg ReportConfig) (ReportConfig, error) {
	id, err := re.idp.ID()
	if err != nil {
		return ReportConfig{}, err
	}

	now := time.Now()
	cfg.ID = id
	cfg.CreatedAt = now
	cfg.CreatedBy = session.UserID
	cfg.DomainID = session.DomainID
	cfg.Status = EnabledStatus

	if cfg.Schedule.StartDateTime.IsZero() {
		cfg.Schedule.StartDateTime = now
	}

	reportConfig, err := re.repo.AddReportConfig(ctx, cfg)
	if err != nil {
		return ReportConfig{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	return reportConfig, nil
}

func (re *re) ViewReportConfig(ctx context.Context, session authn.Session, id string) (ReportConfig, error) {
	cfg, err := re.repo.ViewReportConfig(ctx, id)
	if err != nil {
		return ReportConfig{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	return cfg, nil
}

func (re *re) UpdateReportConfig(ctx context.Context, session authn.Session, cfg ReportConfig) (ReportConfig, error) {
	cfg.UpdatedAt = time.Now()
	cfg.UpdatedBy = session.UserID
	reportConfig, err := re.repo.UpdateReportConfig(ctx, cfg)
	if err != nil {
		return ReportConfig{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return reportConfig, nil
}

func (re *re) UpdateReportSchedule(ctx context.Context, session authn.Session, cfg ReportConfig) (ReportConfig, error) {
	cfg.UpdatedAt = time.Now()
	cfg.UpdatedBy = session.UserID
	c, err := re.repo.UpdateReportSchedule(ctx, cfg)
	if err != nil {
		return ReportConfig{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return c, nil
}

func (re *re) RemoveReportConfig(ctx context.Context, session authn.Session, id string) error {
	if err := re.repo.RemoveReportConfig(ctx, id); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return nil
}

func (re *re) ListReportsConfig(ctx context.Context, session authn.Session, pm PageMeta) (ReportConfigPage, error) {
	pm.Domain = session.DomainID
	page, err := re.repo.ListReportsConfig(ctx, pm)
	if err != nil {
		return ReportConfigPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return page, nil
}

func (re *re) EnableReportConfig(ctx context.Context, session authn.Session, id string) (ReportConfig, error) {
	status, err := ToStatus(Enabled)
	if err != nil {
		return ReportConfig{}, err
	}
	cfg := ReportConfig{
		ID:        id,
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
		Status:    status,
	}
	cfg, err = re.repo.UpdateReportConfigStatus(ctx, cfg)
	if err != nil {
		return ReportConfig{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return cfg, nil
}

func (re *re) DisableReportConfig(ctx context.Context, session authn.Session, id string) (ReportConfig, error) {
	status, err := ToStatus(Disabled)
	if err != nil {
		return ReportConfig{}, err
	}
	cfg := ReportConfig{
		ID:        id,
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
		Status:    status,
	}
	cfg, err = re.repo.UpdateReportConfigStatus(ctx, cfg)
	if err != nil {
		return ReportConfig{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return cfg, nil
}

func (re *re) GenerateReport(ctx context.Context, session authn.Session, config ReportConfig, download bool) (ReportPage, error) {
	config.DomainID = session.DomainID

	if config.Status != EnabledStatus {
		return ReportPage{}, svcerr.ErrInvalidStatus
	}

	reportPage, err := re.generateReport(ctx, config, download)
	if err != nil {
		return ReportPage{}, err
	}

	return reportPage, nil
}

func (re *re) generateReport(ctx context.Context, cfg ReportConfig, download bool) (ReportPage, error) {
	agg := grpcReadersV1.Aggregation_AGGREGATION_UNSPECIFIED
	switch cfg.Config.Aggregation.AggType {
	case "MAX":
		agg = grpcReadersV1.Aggregation_MAX
	case "MIN":
		agg = grpcReadersV1.Aggregation_MIN
	case "COUNT":
		agg = grpcReadersV1.Aggregation_COUNT
	case "AVG":
		agg = grpcReadersV1.Aggregation_AVG
	case "SUM":
		agg = grpcReadersV1.Aggregation_SUM
	}

	from, err := reltime.Parse(cfg.Config.From)
	if err != nil {
		return ReportPage{}, err
	}
	to, err := reltime.Parse(cfg.Config.To)
	if err != nil {
		return ReportPage{}, err
	}
	pm := &grpcReadersV1.PageMetadata{
		Aggregation: agg,
		Limit:       limit,
		From:        float64(from.UnixMicro()),
		To:          float64(to.UnixNano()),
		Interval:    cfg.Config.Aggregation.Interval,
	}

	reportPage := ReportPage{
		Reports:     make([]Report, 0),
		From:        from,
		To:          to,
		Aggregation: cfg.Config.Aggregation,
	}

	for _, metric := range cfg.Metrics {
		sMsgs := []senml.Message{}

		pm.Offset = uint64(0)
		pm.Publisher = metric.ClientID
		pm.Name = metric.Name
		if metric.Subtopic != "" {
			pm.Subtopic = metric.Subtopic
		}
		if metric.Protocol != "" {
			pm.Protocol = metric.Protocol
		}
		if metric.Format != "" {
			pm.Format = metric.Format
		}

		msgs, err := re.readers.ReadMessages(ctx, &grpcReadersV1.ReadMessagesReq{
			ChannelId:    metric.ChannelID,
			DomainId:     cfg.DomainID,
			PageMetadata: pm,
		})
		if err != nil {
			return ReportPage{}, err
		}
		for _, msg := range msgs.Messages {
			sMsgs = append(sMsgs, convertToSenml(msg.GetSenml()))
		}

		for msgs.GetTotal() > (pm.Offset + pm.Limit) {
			pm.Offset = pm.Offset + pm.Limit
			msgs, err := re.readers.ReadMessages(ctx, &grpcReadersV1.ReadMessagesReq{
				ChannelId:    metric.ChannelID,
				DomainId:     cfg.DomainID,
				PageMetadata: pm,
			})
			if err != nil {
				return ReportPage{}, err
			}
			for _, msg := range msgs.Messages {
				sMsgs = append(sMsgs, convertToSenml(msg.GetSenml()))
			}
		}

		reportPage.Reports = append(reportPage.Reports, Report{
			Metric:   metric,
			Messages: sMsgs,
		})
	}

	reportPage.Total = uint64(len(reportPage.Reports))

	if download {
		switch cfg.Email.Format {
		case PDF:
			reportPage.PDF, err = generatePDFReport(reportPage.Reports)
			if err != nil {
				return reportPage, err
			}
		case CSV:
			reportPage.CSV, err = generateCSVReport(reportPage.Reports)
			if err != nil {
				return reportPage, err
			}
		case AllFormats:
			reportPage.PDF, err = generatePDFReport(reportPage.Reports)
			if err != nil {
				return reportPage, err
			}
			reportPage.CSV, err = generateCSVReport(reportPage.Reports)
			if err != nil {
				return reportPage, err
			}
		}
	}

	return reportPage, nil
}

func convertToSenml(g *grpcReadersV1.SenMLMessage) senml.Message {
	return senml.Message{
		Protocol:    g.Base.GetProtocol(),
		Subtopic:    g.Base.GetSubtopic(),
		Unit:        g.GetUnit(),
		Time:        g.GetTime(),
		UpdateTime:  g.GetUpdateTime(),
		Value:       g.Value,
		StringValue: g.StringValue,
		DataValue:   g.DataValue,
		BoolValue:   g.BoolValue,
		Sum:         g.Sum,
	}
}
