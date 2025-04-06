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
	ID              string           `db:"id"`
	Name            string           `db:"name"`
	DomainID        string           `db:"domain_id"`
	Metadata        []byte           `db:"metadata,omitempty"`
	InputChannel    string           `db:"input_channel"`
	InputTopic      sql.NullString   `db:"input_topic"`
	LogicType       re.ScriptType    `db:"logic_type"`
	LogicOutputs    pq.Int32Array  `db:"logic_output"`
	LogicValue      string           `db:"logic_value"`
	OutputChannel   sql.NullString   `db:"output_channel"`
	OutputTopic     sql.NullString   `db:"output_topic"`
	StartDateTime   time.Time        `db:"start_datetime"`
	Time            time.Time        `db:"time"`
	Recurring       re.Recurring     `db:"recurring"`
	RecurringPeriod uint             `db:"recurring_period"`
	Status          re.Status        `db:"status"`
	CreatedAt       time.Time        `db:"created_at"`
	CreatedBy       string           `db:"created_by"`
	UpdatedAt       time.Time        `db:"updated_at"`
	UpdatedBy       string           `db:"updated_by"`
	ChannelIDs      []sql.NullString `db:"channel_ids"`
	ClientIDs       []sql.NullString `db:"client_ids"`
	Metrics         []sql.NullString `db:"metrics"`
	To              []sql.NullString `db:"to"`
	From            sql.NullString   `db:"from"`
	Subject         sql.NullString   `db:"subject"`
	Aggregation     sql.NullString   `db:"aggregation"`
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
	if r.ReportConfig != nil {
		return dbRule{
			ID:              r.ID,
			Name:            r.ReportConfig.Name,
			DomainID:        r.ReportConfig.DomainID,
			StartDateTime:   r.ReportConfig.StartDateTime,
			Time:            r.ReportConfig.Time,
			Recurring:       r.ReportConfig.Recurring,
			RecurringPeriod: r.ReportConfig.RecurringPeriod,
			Status:          r.Status,
			CreatedAt:       r.CreatedAt,
			CreatedBy:       r.CreatedBy,
			UpdatedAt:       r.UpdatedAt,
			UpdatedBy:       r.UpdatedBy,
			ChannelIDs:      toNullStringSlice(r.ReportConfig.ChannelIDs),
			ClientIDs:       toNullStringSlice(r.ReportConfig.ClientIDs),
			Metrics:         toNullStringSlice(r.ReportConfig.Metrics),
			Aggregation:     toNullString(r.ReportConfig.Aggregation),
			To:              toNullStringSlice(r.ReportConfig.Email.To),
			From:            toNullString(r.ReportConfig.Email.From),
			Subject:         toNullString(r.ReportConfig.Email.Subject),
		}, nil
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
		StartDateTime:   r.Schedule.StartDateTime,
		Time:            r.Schedule.Time,
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

	rule := re.Rule{
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
			StartDateTime:   dto.StartDateTime,
			Time:            dto.Time,
			Recurring:       dto.Recurring,
			RecurringPeriod: dto.RecurringPeriod,
		},
		Status:    re.Status(dto.Status),
		CreatedAt: dto.CreatedAt,
		CreatedBy: dto.CreatedBy,
		UpdatedAt: dto.UpdatedAt,
		UpdatedBy: dto.UpdatedBy,
	}

	if dto.ChannelIDs != nil && len(dto.ChannelIDs) > 0 {
		rule.ReportConfig = &re.ReportConfig{
			Name:            dto.Name,
			DomainID:        dto.DomainID,
			ChannelIDs:      fromNullStringSlice(dto.ChannelIDs),
			ClientIDs:       fromNullStringSlice(dto.ClientIDs),
			StartDateTime:   dto.StartDateTime,
			Time:            dto.Time,
			Recurring:       dto.Recurring,
			RecurringPeriod: dto.RecurringPeriod,
			Aggregation:     fromNullString(dto.Aggregation),
			Metrics:         fromNullStringSlice(dto.Metrics),
		}

		if (dto.To != nil && len(dto.To) > 0) || dto.From.Valid || dto.Subject.Valid {
			rule.ReportConfig.Email = &re.Email{
				To:      fromNullStringSlice(dto.To),
				From:    fromNullString(dto.From),
				Subject: fromNullString(dto.Subject),
			}
		}
	}

	return rule, nil
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

func toNullStringSlice(value []string) []sql.NullString {
	var slice []sql.NullString
	for _, v := range value {
		if v != "" {
			slice = append(slice, sql.NullString{String: v, Valid: true})
		}
	}
	return slice
}

func fromNullStringSlice(nullString []sql.NullString) []string {
	var slice []string
	for _, v := range nullString {
		if v.Valid {
			slice = append(slice, v.String)
		}
	}
	return slice
}
