// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	grpcReadersV1 "github.com/absmach/magistrala/api/grpc/readers/v1"
	"github.com/absmach/supermq"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/transformers/senml"
	"github.com/jung-kurt/gofpdf"
	lua "github.com/yuin/gopher-lua"
)

const (
	hoursInDay   = 24
	daysInWeek   = 7
	monthsInYear = 12

	publisher = "magistrala.re"
	from      = 173568960000000000
	interval  = "1s"
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
	InputTopic      string     `json:"input_topic,omitempty" db:"input_topic"`
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

	GenerateReport(ctx context.Context, session authn.Session, config ReportConfig) (ReportPage, error)
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
	readers    grpcReadersV1.ReadersServiceClient
}

func NewService(repo Repository, idp supermq.IDProvider, rePubSub messaging.PubSub, writersPub, alarmsPub messaging.Publisher, tck Ticker, emailer Emailer, readers grpcReadersV1.ReadersServiceClient) Service {
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

func (re *re) Handle(msg *messaging.Message) error {
	inputChannel := msg.Channel

	pm := PageMeta{
		InputChannel: inputChannel,
		Status:       EnabledStatus,
		InputTopic:   msg.Subtopic,
	}
	ctx := context.Background()
	page, err := re.repo.ListRules(ctx, pm)
	if err != nil {
		return err
	}
	reportConfigs, err := re.repo.ListReportsConfig(ctx, pm)
	if err != nil {
		return err
	}

	for _, r := range page.Rules {
		go func(ctx context.Context, rule Rule) {
			if err := re.process(ctx, r, msg); err != nil {
				re.errors <- err
			}
		}(ctx, r)
	}

	for _, cfg := range reportConfigs.ReportConfigs {
		go func(ctx context.Context, config ReportConfig) {
			re.errors <- re.processReportConfig(ctx, config)
		}(ctx, cfg)
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
	l.SetGlobal("send_alarm", l.NewFunction(re.sendAlarm(ctx, r.ID, msg)))

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
		if re.pubSub == nil {
			return errors.New("message broker not initialized")
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

func (re *re) processReportConfig(ctx context.Context, cfg ReportConfig) error {
	reportPage, err := re.generateReport(ctx, cfg)
	if err != nil {
		return err
	}

	if len(cfg.Email.To) > 0 {
		reportContent, err := json.Marshal(reportPage)
		if err != nil {
			return err
		}

		err = re.email.SendEmailNotification(
			cfg.Email.To,
			cfg.Email.From,
			cfg.Email.Subject,
			"",
			"",
			string(reportContent),
			"",
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (re *re) StartScheduler(ctx context.Context) error {
	defer re.ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-re.ticker.Tick():
			startTime := time.Now()

			rulePM := PageMeta{
				Status:          EnabledStatus,
				ScheduledBefore: &startTime,
			}

			page, err := re.repo.ListRules(ctx, rulePM)
			if err != nil {
				return err
			}

			for _, rule := range page.Rules {
				if rule.Schedule.ShouldRun(startTime) {
					go func(r Rule) {
						msg := &messaging.Message{
							Channel: r.InputChannel,
							Created: startTime.Unix(),
						}
						re.errors <- re.process(ctx, r, msg)
					}(rule)
				}
			}

			reportPM := PageMeta{
				Status:          EnabledStatus,
				ScheduledBefore: &startTime,
			}

			reportConfigs, err := re.repo.ListReportsConfig(ctx, reportPM)
			if err != nil {
				return err
			}

			for _, cfg := range reportConfigs.ReportConfigs {
				if cfg.Schedule.ShouldRun(startTime) {
					go func(config ReportConfig) {
						re.errors <- re.processReportConfig(ctx, config)
					}(cfg)
				}
			}
		}
	}
}

func (s Schedule) ShouldRun(startTime time.Time) bool {
	// Don't run if the rule's start time is in the future
	// This allows scheduling rules to start at a specific future time
	if s.StartDateTime.After(startTime) {
		return false
	}

	t := s.Time.Truncate(time.Minute).UTC()
	startTimeOnly := time.Date(0, 1, 1, startTime.Hour(), startTime.Minute(), 0, 0, time.UTC)
	if t.Equal(startTimeOnly) {
		return true
	}

	if s.RecurringPeriod == 0 {
		return false
	}

	period := int(s.RecurringPeriod)

	switch s.Recurring {
	case Daily:
		if s.RecurringPeriod > 0 {
			daysSinceStart := startTime.Sub(s.StartDateTime).Hours() / hoursInDay
			if int(daysSinceStart)%period == 0 {
				return true
			}
		}
	case Weekly:
		if s.RecurringPeriod > 0 {
			weeksSinceStart := startTime.Sub(s.StartDateTime).Hours() / (hoursInDay * daysInWeek)
			if int(weeksSinceStart)%period == 0 {
				return true
			}
		}
	case Monthly:
		if s.RecurringPeriod > 0 {
			monthsSinceStart := (startTime.Year()-s.StartDateTime.Year())*monthsInYear +
				int(startTime.Month()-s.StartDateTime.Month())
			if monthsSinceStart%period == 0 {
				return true
			}
		}
	}

	return false
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

func (re *re) GenerateReport(ctx context.Context, session authn.Session, config ReportConfig) (ReportPage, error) {
	config.DomainID = session.DomainID

	reportPage, err := re.generateReport(ctx, config)
	if err != nil {
		return ReportPage{}, err
	}

	return reportPage, nil
}

func (re *re) generateReport(ctx context.Context, cfg ReportConfig) (ReportPage, error) {
	reportPage := ReportPage{
		Reports: make([]Report, 0),
	}

	report := Report{
		ClientMessages: make(map[string][]senml.Message),
	}

	for _, ch := range cfg.ChannelIDs {
		agg := grpcReadersV1.Aggregation_AGGREGATION_UNSPECIFIED
		switch cfg.Aggregation {
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
				From:        from,
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

	reportPage.Reports = append(reportPage.Reports, report)
	reportPage.Total = uint64(len(reportPage.Reports))

	var err error
	reportPage.PDF, err = re.generatePDFReport(report)
	if err != nil {
		return reportPage, err
	}

	reportPage.CSV, err = re.generateCSVReport(report)
	if err != nil {
		return reportPage, err
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

func (re *re) generatePDFReport(report Report) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "Device Metrics Report")
	pdf.Ln(15)

	for publisher, messages := range report.ClientMessages {
		if len(messages) == 0 {
			continue
		}

		pdf.SetFont("Arial", "B", 12)
		pdf.Cell(40, 10, fmt.Sprintf("Device: %s", publisher))
		pdf.Ln(10)

		pdf.SetFont("Arial", "B", 10)
		pdf.SetFillColor(200, 200, 200)

		headers := []string{"Metric Name", "Value", "Unit", "Time"}
		widths := []float64{60, 40, 30, 40}

		for i, header := range headers {
			pdf.Cell(widths[i], 8, header)
		}
		pdf.Ln(-1)

		pdf.SetFont("Arial", "", 10)
		pdf.SetFillColor(255, 255, 255)

		fill := false

		sort.Slice(messages, func(i, j int) bool {
			return messages[i].Time < messages[j].Time
		})

		for _, msg := range messages {
			timeStr := time.Unix(int64(msg.Time), 0).Format("2006-01-02 15:04:05")

			var valueStr string
			if msg.Value != nil {
				valueStr = fmt.Sprintf("%.2f", *msg.Value)
			} else if msg.StringValue != nil {
				valueStr = *msg.StringValue
			} else if msg.BoolValue != nil {
				valueStr = fmt.Sprintf("%v", *msg.BoolValue)
			} else if msg.DataValue != nil {
				valueStr = *msg.DataValue
			} else {
				valueStr = "N/A"
			}

			pdf.Cell(widths[0], 8, msg.Name)
			pdf.Cell(widths[1], 8, valueStr)
			pdf.Cell(widths[2], 8, msg.Unit)
			pdf.Cell(widths[3], 8, timeStr)
			pdf.Ln(-1)

			fill = !fill
		}

		pdf.Ln(10)
	}

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (re *re) generateCSVReport(report Report) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	for publisher, messages := range report.ClientMessages {
		if len(messages) == 0 {
			continue
		}

		if err := writer.Write([]string{fmt.Sprintf("Device: %s", publisher)}); err != nil {
			return nil, err
		}

		if err := writer.Write([]string{"Metric Name", "Value", "Unit", "Time", "Channel", "Subtopic"}); err != nil {
			return nil, err
		}

		sort.Slice(messages, func(i, j int) bool {
			return messages[i].Time < messages[j].Time
		})

		for _, msg := range messages {
			timeStr := time.Unix(int64(msg.Time), 0).Format("2006-01-02 15:04:05")

			var valueStr string
			if msg.Value != nil {
				valueStr = fmt.Sprintf("%.2f", *msg.Value)
			} else if msg.StringValue != nil {
				valueStr = *msg.StringValue
			} else if msg.BoolValue != nil {
				valueStr = fmt.Sprintf("%v", *msg.BoolValue)
			} else if msg.DataValue != nil {
				valueStr = *msg.DataValue
			} else {
				valueStr = "N/A"
			}

			row := []string{
				msg.Name,
				valueStr,
				msg.Unit,
				timeStr,
				msg.Channel,
				msg.Subtopic,
			}

			if err := writer.Write(row); err != nil {
				return nil, err
			}
		}

		if err := writer.Write([]string{}); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
