// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"context"
	"fmt"
	"strings"
	"time"

	grpcReadersV1 "github.com/absmach/magistrala/api/grpc/readers/v1"
	"github.com/absmach/magistrala/pkg/emailer"
	pkglog "github.com/absmach/magistrala/pkg/logger"
	"github.com/absmach/magistrala/pkg/reltime"
	"github.com/absmach/magistrala/pkg/ticker"
	"github.com/absmach/supermq"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/transformers/senml"
)

const limit = 1000

type report struct {
	repo    Repository
	runInfo chan pkglog.RunInfo
	idp     supermq.IDProvider
	email   emailer.Emailer
	ticker  ticker.Ticker
	readers grpcReadersV1.ReadersServiceClient
}

func NewService(repo Repository, runInfo chan pkglog.RunInfo, idp supermq.IDProvider, tck ticker.Ticker, emailer emailer.Emailer, readers grpcReadersV1.ReadersServiceClient) Service {
	return &report{
		repo:    repo,
		idp:     idp,
		runInfo: runInfo,
		email:   emailer,
		ticker:  tck,
		readers: readers,
	}
}

func (r *report) AddReportConfig(ctx context.Context, session authn.Session, cfg ReportConfig) (ReportConfig, error) {
	id, err := r.idp.ID()
	if err != nil {
		return ReportConfig{}, err
	}

	now := time.Now().UTC()
	cfg.ID = id
	cfg.CreatedAt = now
	cfg.CreatedBy = session.UserID
	cfg.DomainID = session.DomainID
	cfg.Status = EnabledStatus

	if cfg.Schedule.StartDateTime == nil || cfg.Schedule.StartDateTime.IsZero() {
		cfg.Schedule.StartDateTime = &now
	}
	cfg.Schedule.Time = *cfg.Schedule.StartDateTime

	reportConfig, err := r.repo.AddReportConfig(ctx, cfg)
	if err != nil {
		return ReportConfig{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	return reportConfig, nil
}

func (r *report) ViewReportConfig(ctx context.Context, session authn.Session, id string) (ReportConfig, error) {
	cfg, err := r.repo.ViewReportConfig(ctx, id)
	if err != nil {
		return ReportConfig{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	return cfg, nil
}

func (r *report) UpdateReportConfig(ctx context.Context, session authn.Session, cfg ReportConfig) (ReportConfig, error) {
	cfg.UpdatedAt = time.Now().UTC()
	cfg.UpdatedBy = session.UserID
	reportConfig, err := r.repo.UpdateReportConfig(ctx, cfg)
	if err != nil {
		return ReportConfig{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return reportConfig, nil
}

func (r *report) UpdateReportSchedule(ctx context.Context, session authn.Session, cfg ReportConfig) (ReportConfig, error) {
	cfg.UpdatedAt = time.Now().UTC()
	cfg.UpdatedBy = session.UserID
	cfg.Schedule.Time = *cfg.Schedule.StartDateTime
	c, err := r.repo.UpdateReportSchedule(ctx, cfg)
	if err != nil {
		return ReportConfig{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return c, nil
}

func (r *report) RemoveReportConfig(ctx context.Context, session authn.Session, id string) error {
	if err := r.repo.RemoveReportConfig(ctx, id); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return nil
}

func (r *report) ListReportsConfig(ctx context.Context, session authn.Session, pm PageMeta) (ReportConfigPage, error) {
	pm.Domain = session.DomainID
	page, err := r.repo.ListReportsConfig(ctx, pm)
	if err != nil {
		return ReportConfigPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return page, nil
}

func (r *report) EnableReportConfig(ctx context.Context, session authn.Session, id string) (ReportConfig, error) {
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
	cfg, err = r.repo.UpdateReportConfigStatus(ctx, cfg)
	if err != nil {
		return ReportConfig{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return cfg, nil
}

func (r *report) DisableReportConfig(ctx context.Context, session authn.Session, id string) (ReportConfig, error) {
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
	cfg, err = r.repo.UpdateReportConfigStatus(ctx, cfg)
	if err != nil {
		return ReportConfig{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return cfg, nil
}

func (r *report) GenerateReport(ctx context.Context, session authn.Session, config ReportConfig, action ReportAction) (ReportPage, error) {
	config.DomainID = session.DomainID

	if config.Status != EnabledStatus {
		return ReportPage{}, svcerr.ErrInvalidStatus
	}

	reportPage, err := r.generateReport(ctx, config, action)
	if err != nil {
		return ReportPage{}, err
	}

	return reportPage, nil
}

func (r *report) generateReport(ctx context.Context, cfg ReportConfig, action ReportAction) (ReportPage, error) {
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

		msgs, err := r.readers.ReadMessages(ctx, &grpcReadersV1.ReadMessagesReq{
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
			msgs, err := r.readers.ReadMessages(ctx, &grpcReadersV1.ReadMessagesReq{
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
			if err := r.emailReports(*cfg.Email, file); err != nil {
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

func (r *report) emailReports(es EmailSetting, file ReportFile) error {
	if err := es.Validate(); err != nil {
		return errors.Wrap(svcerr.ErrMalformedEntity, err)
	}

	attachments := map[string][]byte{
		file.Name: file.Data,
	}

	if err := r.email.SendEmailNotification(
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
