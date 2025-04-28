// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/absmach/magistrala/re"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/lib/pq"
)

// dbRule represents the database structure for a Rule.
type dbRule struct {
	ID              string         `db:"id"`
	Name            string         `db:"name"`
	DomainID        string         `db:"domain_id"`
	Metadata        []byte         `db:"metadata,omitempty"`
	InputChannel    string         `db:"input_channel"`
	InputTopic      sql.NullString `db:"input_topic"`
	LogicType       re.ScriptType  `db:"logic_type"`
	LogicOutputs    pq.Int32Array  `db:"logic_output"`
	LogicValue      string         `db:"logic_value"`
	OutputChannel   sql.NullString `db:"output_channel"`
	OutputTopic     sql.NullString `db:"output_topic"`
	StartDateTime   sql.NullTime   `db:"start_datetime"`
	Time            sql.NullTime   `db:"time"`
	Recurring       re.Recurring   `db:"recurring"`
	RecurringPeriod uint           `db:"recurring_period"`
	Status          re.Status      `db:"status"`
	CreatedAt       time.Time      `db:"created_at"`
	CreatedBy       string         `db:"created_by"`
	UpdatedAt       time.Time      `db:"updated_at"`
	UpdatedBy       string         `db:"updated_by"`
}

// dbReport represents the database structure for a Report.
type dbReport struct {
	ID              string       `db:"id"`
	Name            string       `db:"name"`
	Description     string       `db:"description"`
	DomainID        string       `db:"domain_id"`
	StartDateTime   sql.NullTime `db:"start_datetime"`
	Time            sql.NullTime `db:"time"`
	Recurring       re.Recurring `db:"recurring"`
	RecurringPeriod uint         `db:"recurring_period"`
	Status          re.Status    `db:"status"`
	CreatedAt       time.Time    `db:"created_at"`
	CreatedBy       string       `db:"created_by"`
	UpdatedAt       time.Time    `db:"updated_at"`
	UpdatedBy       string       `db:"updated_by"`
	Config          []byte       `db:"config,omitempty"`
	Metrics         []byte       `db:"metrics"`
	Email           []byte       `db:"email"`
}

func ruleToDb(r re.Rule) (dbRule, error) {
	metadata := []byte("{}")
	if len(r.Metadata) > 0 {
		b, err := json.Marshal(r.Metadata)
		if err != nil {
			return dbRule{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
		metadata = b
	}
	lo := pq.Int32Array{}
	for _, v := range r.Logic.Outputs {
		lo = append(lo, int32(v))
	}
	start := sql.NullTime{Time: r.Schedule.StartDateTime}
	if !r.Schedule.StartDateTime.IsZero() {
		start.Valid = true
	}
	t := sql.NullTime{Time: r.Schedule.Time}
	if !r.Schedule.Time.IsZero() {
		t.Valid = true
	}
	return dbRule{
		ID:              r.ID,
		Name:            r.Name,
		DomainID:        r.DomainID,
		Metadata:        metadata,
		InputChannel:    r.InputChannel,
		InputTopic:      toNullString(r.InputTopic),
		LogicType:       r.Logic.Type,
		LogicOutputs:    lo,
		LogicValue:      r.Logic.Value,
		OutputChannel:   toNullString(r.OutputChannel),
		OutputTopic:     toNullString(r.OutputTopic),
		StartDateTime:   start,
		Time:            t,
		Recurring:       r.Schedule.Recurring,
		RecurringPeriod: r.Schedule.RecurringPeriod,
		Status:          r.Status,
		CreatedAt:       r.CreatedAt,
		CreatedBy:       r.CreatedBy,
		UpdatedAt:       r.UpdatedAt,
		UpdatedBy:       r.UpdatedBy,
	}, nil
}

func dbToRule(dto dbRule) (re.Rule, error) {
	var metadata re.Metadata
	if dto.Metadata != nil {
		if err := json.Unmarshal(dto.Metadata, &metadata); err != nil {
			return re.Rule{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
	}
	lo := []re.ScriptOutput{}
	for _, v := range dto.LogicOutputs {
		lo = append(lo, re.ScriptOutput(v))
	}
	return re.Rule{
		ID:           dto.ID,
		Name:         dto.Name,
		DomainID:     dto.DomainID,
		Metadata:     metadata,
		InputChannel: dto.InputChannel,
		InputTopic:   fromNullString(dto.InputTopic),
		Logic: re.Script{
			Outputs: lo,
			Type:    dto.LogicType,
			Value:   dto.LogicValue,
		},
		OutputChannel: fromNullString(dto.OutputChannel),
		OutputTopic:   fromNullString(dto.OutputTopic),
		Schedule: re.Schedule{
			StartDateTime:   dto.StartDateTime.Time,
			Time:            dto.Time.Time,
			Recurring:       dto.Recurring,
			RecurringPeriod: dto.RecurringPeriod,
		},
		Status:    dto.Status,
		CreatedAt: dto.CreatedAt,
		CreatedBy: dto.CreatedBy,
		UpdatedAt: dto.UpdatedAt,
		UpdatedBy: dto.UpdatedBy,
	}, nil
}

func reportToDb(r re.ReportConfig) (dbReport, error) {
	config := []byte("{}")
	if r.Config != nil {
		b, err := json.Marshal(r.Config)
		if err != nil {
			return dbReport{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
		config = b
	}

	metrics := []byte("{}")
	if r.Metrics != nil {
		m, err := json.Marshal(r.Metrics)
		if err != nil {
			return dbReport{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
		metrics = m
	}

	email := []byte("{}")
	if r.Email != nil {
		e, err := json.Marshal(r.Email)
		if err != nil {
			return dbReport{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
		email = e
	}

	start := sql.NullTime{Time: r.Schedule.StartDateTime}
	if !r.Schedule.StartDateTime.IsZero() {
		start.Valid = true
	}
	t := sql.NullTime{Time: r.Schedule.Time}
	if !r.Schedule.Time.IsZero() {
		t.Valid = true
	}

	return dbReport{
		ID:              r.ID,
		Name:            r.Name,
		Description:     r.Description,
		DomainID:        r.DomainID,
		StartDateTime:   start,
		Time:            t,
		Recurring:       r.Schedule.Recurring,
		RecurringPeriod: r.Schedule.RecurringPeriod,
		Status:          r.Status,
		CreatedAt:       r.CreatedAt,
		CreatedBy:       r.CreatedBy,
		UpdatedAt:       r.UpdatedAt,
		UpdatedBy:       r.UpdatedBy,
		Config:          config,
		Metrics:         metrics,
		Email:           email,
	}, nil
}

func dbToReport(dto dbReport) (re.ReportConfig, error) {
	var config re.MetricConfig
	if dto.Config != nil {
		if err := json.Unmarshal(dto.Config, &config); err != nil {
			return re.ReportConfig{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
	}

	var email re.EmailSetting
	if dto.Email != nil {
		if err := json.Unmarshal(dto.Email, &email); err != nil {
			return re.ReportConfig{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
	}

	var metrics []re.Metric
	if dto.Metrics != nil {
		if err := json.Unmarshal(dto.Metrics, &metrics); err != nil {
			return re.ReportConfig{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
	}

	rpt := re.ReportConfig{
		ID:          dto.ID,
		Name:        dto.Name,
		Description: dto.Description,
		DomainID:    dto.DomainID,
		Config:      &config,
		Metrics:     metrics,
		Schedule: re.Schedule{
			StartDateTime:   dto.StartDateTime.Time,
			Time:            dto.Time.Time,
			Recurring:       dto.Recurring,
			RecurringPeriod: dto.RecurringPeriod,
		},
		Email:     &email,
		Status:    dto.Status,
		CreatedAt: dto.CreatedAt,
		CreatedBy: dto.CreatedBy,
		UpdatedAt: dto.UpdatedAt,
		UpdatedBy: dto.UpdatedBy,
	}

	return rpt, nil
}

func toNullString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: value, Valid: true}
}

func fromNullString(nullString sql.NullString) string {
	if !nullString.Valid {
		return ""
	}
	return nullString.String
}
