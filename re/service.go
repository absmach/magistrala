// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"context"
	"fmt"
	"strings"
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
	UpdateRuleDue(ctx context.Context, id string, due time.Time) (Rule, error)

	AddReportConfig(ctx context.Context, cfg ReportConfig) (ReportConfig, error)
	ViewReportConfig(ctx context.Context, id string) (ReportConfig, error)
	UpdateReportConfig(ctx context.Context, cfg ReportConfig) (ReportConfig, error)
	UpdateReportSchedule(ctx context.Context, cfg ReportConfig) (ReportConfig, error)
	RemoveReportConfig(ctx context.Context, id string) error
	UpdateReportConfigStatus(ctx context.Context, cfg ReportConfig) (ReportConfig, error)
	ListReportsConfig(ctx context.Context, pm PageMeta) (ReportConfigPage, error)
	UpdateReportDue(ctx context.Context, id string, due time.Time) (ReportConfig, error)
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

	GenerateReport(ctx context.Context, session authn.Session, config ReportConfig, action ReportAction) (ReportPage, error)
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
	r.Schedule.Time = r.Schedule.StartDateTime

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
	r.UpdatedAt = time.Now().UTC()
	r.UpdatedBy = session.UserID
	rule, err := re.repo.UpdateRule(ctx, r)
	if err != nil {
		return Rule{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return rule, nil
}

func (re *re) UpdateRuleSchedule(ctx context.Context, session authn.Session, r Rule) (Rule, error) {
	r.UpdatedAt = time.Now().UTC()
	r.UpdatedBy = session.UserID
	r.Schedule.Time = r.Schedule.StartDateTime
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
		UpdatedAt: time.Now().UTC(),
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
		UpdatedAt: time.Now().UTC(),
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
	cfg.Schedule.Time = cfg.Schedule.StartDateTime

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
	cfg.UpdatedAt = time.Now().UTC()
	cfg.UpdatedBy = session.UserID
	reportConfig, err := re.repo.UpdateReportConfig(ctx, cfg)
	if err != nil {
		return ReportConfig{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return reportConfig, nil
}

func (re *re) UpdateReportSchedule(ctx context.Context, session authn.Session, cfg ReportConfig) (ReportConfig, error) {
	cfg.UpdatedAt = time.Now().UTC()
	cfg.UpdatedBy = session.UserID
	cfg.Schedule.Time = cfg.Schedule.StartDateTime
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
		UpdatedAt: time.Now().UTC(),
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
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: session.UserID,
		Status:    status,
	}
	cfg, err = re.repo.UpdateReportConfigStatus(ctx, cfg)
	if err != nil {
		return ReportConfig{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return cfg, nil
}

func (re *re) GenerateReport(ctx context.Context, session authn.Session, config ReportConfig, action ReportAction) (ReportPage, error) {
	config.DomainID = session.DomainID

	if config.Status != EnabledStatus {
		return ReportPage{}, svcerr.ErrInvalidStatus
	}

	reportPage, err := re.generateReport(ctx, config, action)
	if err != nil {
		return ReportPage{}, err
	}

	return reportPage, nil
}

func (re *re) generateReport(ctx context.Context, cfg ReportConfig, action ReportAction) (ReportPage, error) {
	genReportFile, err := generateFileFunc(action, cfg.Config.FileFormat)
	if err != nil {
		return ReportPage{}, err
	}

	agg := grpcReadersV1.Aggregation_AGGREGATION_UNSPECIFIED
	switch cfg.Config.Aggregation.AggType {
	case AggregationMAX:
		agg = grpcReadersV1.Aggregation_MAX
	case AggregationMIN:
		agg = grpcReadersV1.Aggregation_MIN
	case AggregationCOUNT:
		agg = grpcReadersV1.Aggregation_COUNT
	case AggregationAVG:
		agg = grpcReadersV1.Aggregation_AVG
	case AggregationSUM:
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

	var mets []Metric
	var reports []Report
	for _, metric := range cfg.Metrics {
		switch {
		case len(metric.ClientIDs) != 0:
			for _, clientID := range metric.ClientIDs {
				mets = append(mets, Metric{
					ChannelID: metric.ChannelID,
					ClientID:  clientID,
					Name:      metric.Name,
					Subtopic:  metric.Subtopic,
					Protocol:  metric.Protocol,
					Format:    metric.Format,
				})
			}
		default:
			mets = append(mets, Metric{
				ChannelID: metric.ChannelID,
				Name:      metric.Name,
				Subtopic:  metric.Subtopic,
				Protocol:  metric.Protocol,
				Format:    metric.Format,
			})
		}
	}

	for _, metric := range mets {
		sMsgs := []senml.Message{}

		pm.Offset = uint64(0)
		pm.Name = metric.Name
		if metric.ClientID != "" {
			pm.Publisher = metric.ClientID
		}
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

		reports = append(reports, convertToReports(metric, sMsgs)...)
	}

	switch {
	case genReportFile != nil:
		data, err := genReportFile(cfg.Config.Title, reports)
		if err != nil {
			return ReportPage{}, err
		}
		timeStr := strings.ReplaceAll(time.Now().Format(time.RFC3339), ":", "")
		filePrefix := cfg.Name
		if filePrefix == "" {
			filePrefix = "report"
		}
		fileName := fmt.Sprintf("%s_%s.%s", filePrefix, timeStr, cfg.Config.FileFormat.Extension())

		file := ReportFile{
			Name:   fileName,
			Data:   data,
			Format: cfg.Config.FileFormat,
		}

		switch action {
		case EmailReport:
			if err := re.emailReports(*cfg.Email, file); err != nil {
				return ReportPage{}, errors.Wrap(err, svcerr.ErrCreateEntity)
			}

			return ReportPage{}, nil
		default:
			return ReportPage{
				File: file,
			}, nil
		}

	default:
		return ReportPage{
			From:        from,
			To:          to,
			Aggregation: cfg.Config.Aggregation,
			Total:       uint64(len(reports)),
			Reports:     reports,
		}, nil
	}
}

func generateFileFunc(action ReportAction, format Format) (func(string, []Report) ([]byte, error), error) {
	switch action {
	case DownloadReport, EmailReport:
		switch format {
		case PDF:
			return generatePDFReport, nil
		case CSV:
			return generateCSVReport, nil
		default:
			return nil, errors.New("file format not supported")
		}
	default:
		return nil, nil
	}
}

func (re *re) emailReports(es EmailSetting, file ReportFile) error {
	if err := es.Validate(); err != nil {
		return errors.Wrap(svcerr.ErrMalformedEntity, err)
	}

	attachments := map[string][]byte{
		file.Name: file.Data,
	}

	if err := re.email.SendEmailNotification(
		es.To,
		"",
		es.Subject,
		"",
		"",
		es.Content,
		"",
		attachments,
	); err != nil {
		return err
	}
	return nil
}

func convertToSenml(g *grpcReadersV1.SenMLMessage) senml.Message {
	if g == nil {
		return senml.Message{}
	}
	return senml.Message{
		Protocol:    g.Base.GetProtocol(),
		Subtopic:    g.Base.GetSubtopic(),
		Publisher:   g.Base.GetPublisher(),
		Channel:     g.Base.GetChannel(),
		Name:        g.GetName(),
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

func convertToReports(metric Metric, senmlMsgs []senml.Message) []Report {
	if metric.ClientID != "" {
		return []Report{
			{
				Metric:   metric,
				Messages: senmlMsgs,
			},
		}
	}

	return groupReportsByPublisher(metric, senmlMsgs)
}

func groupReportsByPublisher(metric Metric, sMsgs []senml.Message) []Report {
	publishers := map[string][]senml.Message{}

	for _, msg := range sMsgs {
		publishers[msg.Publisher] = append(publishers[msg.Publisher], msg)
	}

	var groupedReports []Report
	for publisher, messages := range publishers {
		gMetric := metric
		gMetric.ClientID = publisher
		groupedReports = append(groupedReports, Report{
			Metric:   gMetric,
			Messages: messages,
		})
	}

	return groupedReports
}
