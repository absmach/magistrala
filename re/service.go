// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"context"
	"time"

	grpcReadersV1 "github.com/absmach/magistrala/api/grpc/readers/v1"
	"github.com/absmach/supermq"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/transformers/senml"
	"github.com/vadv/gopher-lua-libs/strings"
)

type Repository interface {
	AddRule(ctx context.Context, r Rule) (Rule, error)
	ViewRule(ctx context.Context, id string) (Rule, error)
	UpdateRule(ctx context.Context, r Rule) (Rule, error)
	UpdateRuleSchedule(ctx context.Context, r Rule) (Rule, error)
	RemoveRule(ctx context.Context, id string) error
	UpdateRuleStatus(ctx context.Context, id string, status Status) (Rule, error)
	ListRules(ctx context.Context, pm PageMeta) (Page, error)

	AddReportConfig(ctx context.Context, cfg ReportConfig) (ReportConfig, error)
	ViewReportConfig(ctx context.Context, id string) (ReportConfig, error)
	UpdateReportConfig(ctx context.Context, cfg ReportConfig) (ReportConfig, error)
	RemoveReportConfig(ctx context.Context, id string) error
	UpdateReportConfigStatus(ctx context.Context, id string, status Status) (ReportConfig, error)
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
	cfg, err := re.repo.UpdateReportConfigStatus(ctx, id, status)
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
	cfg, err := re.repo.UpdateReportConfigStatus(ctx, id, status)
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
	reportPage := ReportPage{
		Reports: make([]Report, 0),
	}

	report := Report{
		ClientMessages: make(map[string][]senml.Message, 0),
	}

	for _, ch := range cfg.ChannelIDs {
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

		msgs, err := re.readers.ReadMessages(ctx, &grpcReadersV1.ReadMessagesReq{
			ChannelId: ch,
			DomainId:  cfg.DomainID,
			PageMetadata: &grpcReadersV1.PageMetadata{
				Aggregation: agg,
				Limit:       cfg.Limit,
				Offset:      0,
				From:        float64(from.UnixNano()),
				To:          float64(time.Now().UnixNano()),
				Interval:    interval,
			},
		})
		if err != nil {
			return ReportPage{}, err
		}

		for _, msg := range msgs.Messages {
			message := msg.GetSenml()
			publisher := message.Base.Publisher

			if contains(cfg.ClientIDs, publisher) && shouldIncludeMessage(message.Name, cfg.Metrics) {
				report.ClientMessages[publisher] = append(report.ClientMessages[publisher],
					senml.Message{
						Channel:     message.Base.Channel,
						Subtopic:    message.Base.Subtopic,
						Publisher:   message.Base.Publisher,
						Protocol:    message.Base.Protocol,
						Name:        message.Name,
						Unit:        message.Unit,
						Time:        message.Time,
						UpdateTime:  message.UpdateTime,
						Value:       &message.Value,
						StringValue: &message.StringValue,
						DataValue:   &message.DataValue,
						BoolValue:   &message.BoolValue,
						Sum:         &message.Sum,
					},
				)
			}
		}
	}

	if len(reportPage.Reports) > 0 {
		reportPage.Reports = append(reportPage.Reports, report)
		reportPage.Total = uint64(len(reportPage.Reports))
	}
	if download {
		var err error

		reportPage.PDF, err = re.generatePDFReport(report)
		if err != nil {
			return reportPage, err
		}

		reportPage.CSV, err = re.generateCSVReport(report)
		if err != nil {
			return reportPage, err
		}
	}

	return reportPage, nil
}

func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func shouldIncludeMessage(name string, metrics []string) bool {
	if len(metrics) == 0 {
		return true
	}

	for _, metric := range metrics {
		if strings.Contains(name, metric) {
			return true
		}
	}

	return false
}
