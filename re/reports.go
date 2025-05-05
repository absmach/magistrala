// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/absmach/magistrala/pkg/reltime"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/transformers/senml"
)

var (
	errFromTimeNotProvided        = errors.New("\"from time\" not provided")
	errInvalidFromTime            = errors.New("invalid \"from time\" ")
	errToTimeNotProvided          = errors.New("\"to time\" not provided")
	errInvalidToTime              = errors.New("invalid \"to time\"")
	errAggIntervalTimeNotProvided = errors.New("aggregation interval time not provided")
	errInvalidAggInterval         = errors.New("invalid aggregation interval time")
	errNoToEmail                  = errors.New("no \"To\" email address found")
	errChannelIDNotProvided       = errors.New("channel id not provided")
	errNameNotProvided            = errors.New("name not provided")
)

const (
	errInvalidFormatFmt       = "invalid format %s"
	errInvalidReportActionFmt = "invalid action %s"
	errInvalidToEmail         = "invalid \"To\" email %s"

	errUnknownAggregationFmt       = "unknown aggregation type %d"
	errUnknownAggregationStringFmt = "unknown aggregation type %s"
)

type Report struct {
	Metric   Metric          `json:"metric,omitempty"`
	Messages []senml.Message `json:"messages,omitempty"`
}

type ReportPage struct {
	Total       uint64     `json:"total"`
	From        time.Time  `json:"from,omitempty"`
	To          time.Time  `json:"to,omitempty"`
	Aggregation AggConfig  `json:"aggregation,omitempty"`
	Reports     []Report   `json:"reports,omitempty"`
	File        ReportFile `json:"file,omitempty"`
}

type ReportFile struct {
	Name   string `json:"name,omitempty"`
	Data   []byte `json:"data,omitempty"`
	Format Format `json:"format,omitempty"`
}

type AggConfig struct {
	AggType  Aggregation `json:"agg_type,omitempty"` // Optional field
	Interval string      `json:"interval,omitempty"` // Mandatory field if "AggType" field is set MAX, MIN, COUNT, SUM, AVG
}

func (ac AggConfig) Validate() error {
	if ac.AggType != AggregationNONE {
		if ac.Interval == "" {
			return errAggIntervalTimeNotProvided
		}

		if _, err := time.ParseDuration(ac.Interval); err != nil {
			return errInvalidAggInterval
		}
	}
	return nil
}

type MetricConfig struct {
	From string `json:"from,omitempty"` // Mandatory field
	To   string `json:"to,omitempty"`   // Mandatory field

	FileFormat Format `json:"file_format,omitempty"` // Optional field

	Aggregation AggConfig `json:"aggregation,omitempty"` // Optional field
}

func (mc MetricConfig) Validate() error {
	if mc.From == "" {
		return errFromTimeNotProvided
	}

	if _, err := reltime.Parse(mc.From); err != nil {
		return errInvalidFromTime
	}

	if mc.To == "" {
		return errToTimeNotProvided
	}

	if _, err := reltime.Parse(mc.To); err != nil {
		return errInvalidToTime
	}
	if err := mc.Aggregation.Validate(); err != nil {
		return err
	}

	return nil
}

type Metric struct {
	ChannelID string `json:"channel_id,omitempty"` // Mandatory field
	ClientID  string `json:"client_id,omitempty"`  // Optional field
	Name      string `json:"name,omitempty"`       // Mandatory field
	Subtopic  string `json:"subtopic,omitempty"`   // Optional field
	Protocol  string `json:"protocol,omitempty"`   // Optional field
	Format    string `json:"format,omitiempty"`    // Optional field
}

type ReqMetric struct {
	ChannelID string   `json:"channel_id,omitempty"` // Mandatory field
	ClientIDs []string `json:"client_ids,omitempty"` // Optional field
	Name      string   `json:"name,omitempty"`       // Mandatory field
	Subtopic  string   `json:"subtopic,omitempty"`   // Optional field
	Protocol  string   `json:"protocol,omitempty"`   // Optional field
	Format    string   `json:"format,omitiempty"`    // Optional field
}

func (rm ReqMetric) Validate() error {
	if rm.ChannelID == "" {
		return errChannelIDNotProvided
	}
	if rm.Name == "" {
		return errNameNotProvided
	}
	return nil
}

type ReportConfig struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	DomainID    string        `json:"domain_id"`
	Schedule    Schedule      `json:"schedule,omitempty"`
	Config      *MetricConfig `json:"config,omitempty"`
	Email       *EmailSetting `json:"email,omitempty"`
	Metrics     []ReqMetric   `json:"metrics,omitempty"`
	Status      Status        `json:"status"`
	CreatedAt   time.Time     `json:"created_at,omitempty"`
	CreatedBy   string        `json:"created_by,omitempty"`
	UpdatedAt   time.Time     `json:"updated_at,omitempty"`
	UpdatedBy   string        `json:"updated_by,omitempty"`
}

type ReportConfigPage struct {
	PageMeta
	ReportConfigs []ReportConfig `json:"report_configs"`
}

type EmailSetting struct {
	To      []string `json:"to,omitempty"`
	Subject string   `json:"subject,omitempty"`
	Content string   `json:"content,omitempty"`
}

func (es *EmailSetting) Validate() error {
	if len(es.To) == 0 {
		return errNoToEmail
	}
	for _, to := range es.To {
		if _, err := mail.ParseAddress(to); err != nil {
			return errors.Wrap(fmt.Errorf(errInvalidToEmail, to), err)
		}
	}
	return nil
}

type Format uint8

const (
	PDF = iota
	CSV
	AllFormats
)

const (
	PdfFormat   = "pdf"
	CsvFormat   = "csv"
	All_Formats = "AllFormats"
)

func (f Format) String() string {
	switch f {
	case PDF:
		return PdfFormat
	case CSV:
		return CsvFormat
	case AllFormats:
		return All_Formats
	default:
		return Unknown
	}
}

func (f Format) Extension() string {
	switch f {
	case PDF:
		return PdfFormat
	case CSV:
		return CsvFormat
	default:
		return Unknown
	}
}

func (f Format) ContentType() string {
	switch f {
	case PDF:
		return "application/pdf"
	case CSV:
		return "text/csv"
	default:
		return Unknown
	}
}

func ToFormat(format string) (Format, error) {
	switch format {
	case "", PdfFormat:
		return PDF, nil
	case CsvFormat:
		return CSV, nil
	case All_Formats:
		return AllFormats, nil
	}
	return Format(0), fmt.Errorf(errInvalidFormatFmt, format)
}

func (f Format) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.String())
}

func (f *Format) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), "\"")
	val, err := ToFormat(str)
	*f = val
	return err
}

type ReportAction uint8

const (
	ViewReport = iota
	DownloadReport
	EmailReport
)

const (
	ViewReportAction     = "view"
	DownloadReportAction = "download"
	EmailReportAction    = "email"
)

func (ra ReportAction) String() string {
	switch ra {
	case ViewReport:
		return ViewReportAction
	case DownloadReport:
		return DownloadReportAction
	case EmailReport:
		return EmailReportAction
	default:
		return Unknown
	}
}

func ToReportAction(action string) (ReportAction, error) {
	switch action {
	case "", ViewReportAction:
		return ViewReport, nil
	case DownloadReportAction:
		return DownloadReport, nil
	case EmailReportAction:
		return EmailReport, nil
	}
	return ReportAction(0), fmt.Errorf(errInvalidReportActionFmt, action)
}

func (ra ReportAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(ra.String())
}

func (ra *ReportAction) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), "\"")
	val, err := ToReportAction(str)
	*ra = val
	return err
}

type Aggregation uint8

const (
	AggregationNONE = iota
	AggregationMAX
	AggregationMIN
	AggregationSUM
	AggregationCOUNT
	AggregationAVG
)

const (
	aggregationNONE  = "none"
	aggregationMAX   = "max"
	aggregationMIN   = "min"
	aggregationSUM   = "sum"
	aggregationCOUNT = "count"
	aggregationAVG   = "avg"
)

func (a Aggregation) String() string {
	switch a {
	case AggregationNONE:
		return aggregationNONE
	case AggregationMAX:
		return aggregationMAX
	case AggregationMIN:
		return aggregationMIN
	case AggregationSUM:
		return aggregationSUM
	case AggregationCOUNT:
		return aggregationCOUNT
	case AggregationAVG:
		return aggregationAVG
	default:
		return fmt.Sprintf(errUnknownAggregationFmt, a)
	}
}

func ToAggregation(agg string) (Aggregation, error) {
	switch strings.ToLower(agg) {
	case "", aggregationNONE:
		return AggregationNONE, nil
	case aggregationMAX:
		return AggregationMAX, nil
	case aggregationMIN:
		return AggregationMIN, nil
	case aggregationSUM:
		return AggregationSUM, nil
	case aggregationCOUNT:
		return AggregationCOUNT, nil
	case aggregationAVG:
		return AggregationAVG, nil
	default:
		return Aggregation(0), fmt.Errorf(errUnknownAggregationStringFmt, agg)
	}
}

func (a Aggregation) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

func (a *Aggregation) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), "\"")
	val, err := ToAggregation(str)
	*a = val
	return err
}
