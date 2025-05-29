// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/absmach/magistrala/pkg/schedule"
	"github.com/absmach/magistrala/re"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/lib/pq"
)

// dbRule represents the database structure for a Rule.
type dbRule struct {
	ID              string             `db:"id"`
	Name            string             `db:"name"`
	DomainID        string             `db:"domain_id"`
	Metadata        []byte             `db:"metadata,omitempty"`
	InputChannel    string             `db:"input_channel"`
	InputTopic      sql.NullString     `db:"input_topic"`
	LogicType       re.ScriptType      `db:"logic_type"`
	LogicOutputs    pq.Int32Array      `db:"logic_output"`
	LogicValue      string             `db:"logic_value"`
	OutputChannel   sql.NullString     `db:"output_channel"`
	OutputTopic     sql.NullString     `db:"output_topic"`
	StartDateTime   sql.NullTime       `db:"start_datetime"`
	Time            sql.NullTime       `db:"time"`
	Recurring       schedule.Recurring `db:"recurring"`
	RecurringPeriod uint               `db:"recurring_period"`
	Status          re.Status          `db:"status"`
	CreatedAt       time.Time          `db:"created_at"`
	CreatedBy       string             `db:"created_by"`
	UpdatedAt       time.Time          `db:"updated_at"`
	UpdatedBy       string             `db:"updated_by"`
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
		Schedule: schedule.Schedule{
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
