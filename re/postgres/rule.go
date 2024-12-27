// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"database/sql"
	"time"

	"github.com/absmach/magistrala/re"
	"github.com/jackc/pgx/v5/pgtype"
)

// dbRule represents the database structure for a Rule.
type dbRule struct {
	ID              string                `db:"id"`
	DomainID        string                `db:"domain_id"`
	InputChannel    string                `db:"input_channel"`
	InputTopic      sql.NullString        `db:"input_topic"`
	LogicType       re.ScriptType         `db:"logic_type"`
	LogicValue      string                `db:"logic_value"`
	OutputChannel   sql.NullString        `db:"output_channel"`
	OutputTopic     sql.NullString        `db:"output_topic"`
	RecurringTime   *pgtype.Array[string] `db:"recurring_time"`
	RecurringType   re.ReccuringType      `db:"recurring_type"`
	RecurringPeriod uint                  `db:"recurring_period"`
	Status          re.Status             `db:"status"`
	CreatedAt       time.Time             `db:"created_at"`
	CreatedBy       string                `db:"created_by"`
	UpdatedAt       time.Time             `db:"updated_at"`
	UpdatedBy       string                `db:"updated_by"`
}

func ruleToDb(r re.Rule) dbRule {
	return dbRule{
		ID:              r.ID,
		DomainID:        r.DomainID,
		InputChannel:    r.InputChannel,
		InputTopic:      toNullString(r.InputTopic),
		LogicType:       r.Logic.Type,
		LogicValue:      r.Logic.Value,
		OutputChannel:   toNullString(r.OutputChannel),
		OutputTopic:     toNullString(r.OutputTopic),
		RecurringTime:   toStringArray(r.Schedule.Time),
		RecurringType:   r.Schedule.RecurringType,
		RecurringPeriod: r.Schedule.RecurringPeriod,
		Status:          r.Status,
		CreatedAt:       r.CreatedAt,
		CreatedBy:       r.CreatedBy,
		UpdatedAt:       r.UpdatedAt,
		UpdatedBy:       r.UpdatedBy,
	}
}

func dbToRule(dto dbRule) re.Rule {
	return re.Rule{
		ID:           dto.ID,
		DomainID:     dto.DomainID,
		InputChannel: dto.InputChannel,
		InputTopic:   fromNullString(dto.InputTopic),
		Logic: re.Script{
			Type:  dto.LogicType,
			Value: dto.LogicValue,
		},
		OutputChannel: fromNullString(dto.OutputChannel),
		OutputTopic:   fromNullString(dto.OutputTopic),
		Schedule: re.Schedule{
			Time:            toTimeSlice(dto.RecurringTime),
			RecurringType:   dto.RecurringType,
			RecurringPeriod: dto.RecurringPeriod,
		},
		Status:    re.Status(dto.Status),
		CreatedAt: dto.CreatedAt,
		CreatedBy: dto.CreatedBy,
		UpdatedAt: dto.UpdatedAt,
		UpdatedBy: dto.UpdatedBy,
	}
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

func toStringArray(times []time.Time) *pgtype.Array[string] {
	var strArray []string
	for _, t := range times {
		strArray = append(strArray, t.Format(time.RFC3339))
	}
	ret := pgtype.Array[string]{
		Elements: strArray,
		Valid:    true,
	}
	return &ret
}

func toTimeSlice(strArray *pgtype.Array[string]) []time.Time {
	if strArray == nil || !strArray.Valid {
		return []time.Time{}
	}
	var times []time.Time
	for _, s := range strArray.Elements {
		t, err := time.Parse(time.RFC3339, s)
		if err == nil {
			times = append(times, t)
		}
	}
	return times
}
