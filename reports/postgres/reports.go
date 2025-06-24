// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/schedule"
	"github.com/absmach/magistrala/reports"
)

// dbReport represents the database structure for a Report.
type dbReport struct {
	ID              string             `db:"id"`
	Name            string             `db:"name"`
	Description     string             `db:"description"`
	DomainID        string             `db:"domain_id"`
	StartDateTime   sql.NullTime       `db:"start_datetime"`
	Due             sql.NullTime       `db:"due"`
	Recurring       schedule.Recurring `db:"recurring"`
	RecurringPeriod uint               `db:"recurring_period"`
	Status          reports.Status     `db:"status"`
	CreatedAt       time.Time          `db:"created_at"`
	CreatedBy       string             `db:"created_by"`
	UpdatedAt       time.Time          `db:"updated_at"`
	UpdatedBy       string             `db:"updated_by"`
	Config          []byte             `db:"config,omitempty"`
	Metrics         []byte             `db:"metrics"`
	Email           []byte             `db:"email"`
}

func reportToDb(r reports.ReportConfig) (dbReport, error) {
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
	start := sql.NullTime{}
	if r.Schedule.StartDateTime != nil && !r.Schedule.StartDateTime.IsZero() {
		start.Time = *r.Schedule.StartDateTime
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
		Due:             t,
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

func dbToReport(dto dbReport) (reports.ReportConfig, error) {
	var config reports.MetricConfig
	if dto.Config != nil {
		if err := json.Unmarshal(dto.Config, &config); err != nil {
			return reports.ReportConfig{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
	}

	var email reports.EmailSetting
	if dto.Email != nil {
		if err := json.Unmarshal(dto.Email, &email); err != nil {
			return reports.ReportConfig{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
	}

	var metrics []reports.ReqMetric
	if dto.Metrics != nil {
		if err := json.Unmarshal(dto.Metrics, &metrics); err != nil {
			return reports.ReportConfig{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
	}

	rpt := reports.ReportConfig{
		ID:          dto.ID,
		Name:        dto.Name,
		Description: dto.Description,
		DomainID:    dto.DomainID,
		Config:      &config,
		Metrics:     metrics,
		Schedule: schedule.Schedule{
			StartDateTime:   &dto.StartDateTime.Time,
			Time:            dto.Due.Time,
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
