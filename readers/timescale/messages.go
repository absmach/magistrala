// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package timescale

import (
	"encoding/json"
	"fmt"
	"strings"

	api "github.com/absmach/supermq/api/http"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/transformers/senml"
	"github.com/absmach/supermq/readers"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx" // required for DB access
)

// Table for SenML messages.
const (
	defTable       = "messages"
	orderByTime    = "time"
	orderByCreated = "created"
)

var _ readers.MessageRepository = (*timescaleRepository)(nil)

type timescaleRepository struct {
	db *sqlx.DB
}

// New returns new TimescaleSQL writer.
func New(db *sqlx.DB) readers.MessageRepository {
	return &timescaleRepository{
		db: db,
	}
}

func (tr timescaleRepository) ReadAll(chanID string, rpm readers.PageMetadata) (readers.MessagesPage, error) {
	format := defTable

	if rpm.Format != "" && rpm.Format != defTable {
		format = rpm.Format
	}

	isSenml := (format == defTable)

	// If aggregation is provided, add time_bucket and aggregation to the query
	const timeDivisor = 1000000000
	isAggregated := isSenml && rpm.Aggregation != "" && rpm.Interval != ""

	if rpm.Order == "" {
		switch {
		case isSenml:
			rpm.Order = orderByTime
		default:
			rpm.Order = orderByCreated
		}
	}

	orderClause := applyOrdering(rpm, isAggregated, isSenml)

	pgData := ""
	if rpm.Limit != 0 {
		pgData = "LIMIT :limit"
	}
	if rpm.Offset != 0 {
		if pgData != "" {
			pgData += " "
		}
		pgData += "OFFSET :offset"
	}

	where := fmtCondition(rpm)

	var q string
	totalQuery := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s;`, format, where)

	if isAggregated {
		q = fmt.Sprintf(`
			SELECT
				EXTRACT(epoch FROM time_bucket('%s', to_timestamp(time/%d))) *%d AS time,
				%s(value) AS value,
				FIRST(publisher, time) AS publisher,
				FIRST(protocol, time) AS protocol,
				FIRST(subtopic, time) AS subtopic,
				FIRST(name,time) AS name,
				FIRST(unit, time) AS unit
			FROM
				%s
			WHERE
				%s
			GROUP BY 1
			%s
			%s;
			`,
			rpm.Interval, timeDivisor, timeDivisor, rpm.Aggregation, format, where, orderClause, pgData)

		totalQuery = fmt.Sprintf(`SELECT COUNT(*) FROM (SELECT EXTRACT(epoch FROM time_bucket('%s', to_timestamp(time/%d))) AS time, %s(value) AS value FROM %s WHERE %s GROUP BY 1) AS subquery;`, rpm.Interval, timeDivisor, rpm.Aggregation, format, where)
	} else {
		q = fmt.Sprintf(`SELECT * FROM %s WHERE %s %s %s;`, format, where, orderClause, pgData)
	}

	params := map[string]any{
		"channel":      chanID,
		"limit":        rpm.Limit,
		"offset":       rpm.Offset,
		"subtopic":     rpm.Subtopic,
		"publisher":    rpm.Publisher,
		"name":         rpm.Name,
		"protocol":     rpm.Protocol,
		"value":        rpm.Value,
		"bool_value":   rpm.BoolValue,
		"string_value": rpm.StringValue,
		"data_value":   rpm.DataValue,
		"from":         rpm.From,
		"to":           rpm.To,
	}

	rows, err := tr.db.NamedQuery(q, params)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == pgerrcode.UndefinedTable {
				return readers.MessagesPage{}, nil
			}
		}
		return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}
	defer rows.Close()

	page := readers.MessagesPage{
		PageMetadata: rpm,
		Messages:     []readers.Message{},
	}

	switch format {
	case defTable:
		for rows.Next() {
			msg := senmlMessage{Message: senml.Message{}}
			if err := rows.StructScan(&msg); err != nil {
				return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
			}

			page.Messages = append(page.Messages, msg.Message)
		}
	default:
		for rows.Next() {
			msg := jsonMessage{}
			if err := rows.StructScan(&msg); err != nil {
				return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
			}
			m, err := msg.toMap()
			if err != nil {
				return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
			}
			page.Messages = append(page.Messages, m)
		}
	}

	rows, err = tr.db.NamedQuery(totalQuery, params)
	if err != nil {
		return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}
	defer rows.Close()

	total := uint64(0)
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return page, err
		}
	}
	page.Total = total

	return page, nil
}

func fmtCondition(rpm readers.PageMetadata) string {
	// Indexed columns conditions based on indices order.
	chCondition := " channel = :channel "

	var query map[string]any
	meta, err := json.Marshal(rpm)
	if err != nil {
		return chCondition
	}
	if err := json.Unmarshal(meta, &query); err != nil {
		return chCondition
	}

	conditions := []string{chCondition}

	if _, ok := query["subtopic"]; ok {
		conditions = append(conditions, " subtopic = :subtopic ")
	}

	if _, ok := query["publisher"]; ok {
		conditions = append(conditions, " publisher = :publisher ")
	}

	if _, ok := query["name"]; ok {
		conditions = append(conditions, " name = :name ")
	}

	if _, ok := query["from"]; ok {
		conditions = append(conditions, " time >= :from ")
	}

	if _, ok := query["to"]; ok {
		conditions = append(conditions, " time < :to ")
	}

	// Non Indexed columns conditions added after indexed columns conditions order.
	if _, ok := query["protocol"]; ok {
		conditions = append(conditions, " protocol = :protocol ")
	}

	for name := range query {
		switch name {
		case "v":
			comparator := readers.ParseValueComparator(query)
			conditions = append(conditions, fmt.Sprintf(" value %s :value ", comparator))
		case "vb":
			conditions = append(conditions, "bool_value = :bool_value")
		case "vs":
			comparator := readers.ParseValueComparator(query)
			switch comparator {
			case "=":
				conditions = append(conditions, " string_value = :string_value ")
			case ">":
				conditions = append(conditions, " string_value LIKE '%%' || :string_value || '%%' AND string_value <> :string_value ")
			case ">=":
				conditions = append(conditions, " string_value LIKE '%%' || :string_value || '%%' ")
			case "<=":
				conditions = append(conditions, " :string_value LIKE '%%' || string_value || '%%' ")
			case "<":
				conditions = append(conditions, " :string_value LIKE '%%' || string_value || '%%' AND string_value <> :string_value ")
			}
		case "vd":
			comparator := readers.ParseValueComparator(query)
			conditions = append(conditions, fmt.Sprintf(" data_value %s :data_value ", comparator))
		}
	}

	return strings.Join(conditions, " AND ")
}

type senmlMessage struct {
	ID string `db:"id"`
	senml.Message
}

type jsonMessage struct {
	Channel   string `db:"channel"`
	Created   int64  `db:"created"`
	Subtopic  string `db:"subtopic"`
	Publisher string `db:"publisher"`
	Protocol  string `db:"protocol"`
	Payload   []byte `db:"payload"`
}

func (msg jsonMessage) toMap() (map[string]any, error) {
	ret := map[string]any{
		"channel":   msg.Channel,
		"created":   msg.Created,
		"subtopic":  msg.Subtopic,
		"publisher": msg.Publisher,
		"protocol":  msg.Protocol,
		"payload":   map[string]any{},
	}
	pld := make(map[string]any)
	if err := json.Unmarshal(msg.Payload, &pld); err != nil {
		return nil, err
	}
	ret["payload"] = pld
	return ret, nil
}

func applyOrdering(pm readers.PageMetadata, isAggregated bool, isSenml bool) string {
	timeCol := orderByTime
	if !isSenml {
		timeCol = orderByCreated
	}

	dir := pm.Dir
	if dir != api.AscDir && dir != api.DescDir {
		dir = api.DescDir
	}

	aggCols := map[string]bool{
		orderByTime: true,
		"value":     true,
		"sum":       true,
		"publisher": true,
		"protocol":  true,
		"subtopic":  true,
		"name":      true,
		"unit":      true,
	}

	senmlCols := map[string]bool{
		orderByTime:    true,
		"value":        true,
		"bool_value":   true,
		"string_value": true,
		"data_value":   true,
		"publisher":    true,
		"name":         true,
		"protocol":     true,
		"channel":      true,
		"subtopic":     true,
		"unit":         true,
	}

	jsonCols := map[string]bool{
		orderByCreated: true, "publisher": true, "protocol": true,
		"channel": true, "subtopic": true,
	}

	if isAggregated {
		col := pm.Order
		if !aggCols[col] {
			col = orderByTime
		}
		if col == orderByTime {
			return fmt.Sprintf("ORDER BY time %s", dir)
		}
		return fmt.Sprintf("ORDER BY %s %s, time %s", col, dir, dir)
	}

	col := pm.Order
	switch {
	case isSenml:
		if !senmlCols[col] {
			col = orderByTime
		}
	case !isSenml:
		if !jsonCols[col] {
			col = orderByCreated
		}
	}

	secondary := fmt.Sprintf("%s DESC", timeCol)

	if col == timeCol {
		return fmt.Sprintf("ORDER BY %s %s", col, dir)
	}
	return fmt.Sprintf("ORDER BY %s %s, %s", col, dir, secondary)
}
