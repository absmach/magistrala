// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/transformers/senml"
	"github.com/absmach/magistrala/readers"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

var _ readers.MessageRepository = (*postgresRepository)(nil)

const (
	messageFieldChannel    = "channel"
	messageFieldDeviceID   = "device_id"
	messageFieldDeviceIDs  = "device_ids"
	messageFieldName       = "name"
	messageFieldProtocol   = "protocol"
	messageFieldPublisher  = "publisher"
	messageFieldPublishers = "publishers"
	messageFieldSubtopic   = "subtopic"
	messageFieldValue      = "value"
)

type postgresRepository struct {
	db *sqlx.DB
}

// New returns new PostgreSQL writer.
func New(db *sqlx.DB) readers.MessageRepository {
	return &postgresRepository{
		db: db,
	}
}

func (tr postgresRepository) ReadAll(chanID string, rpm readers.PageMetadata) (readers.MessagesPage, error) {
	order := "time"
	format := defTable

	if rpm.Format != "" && rpm.Format != defTable {
		order = "created"
		format = rpm.Format
	}
	cond := fmtCondition(chanID, rpm)

	q := fmt.Sprintf(`SELECT * FROM %s
    WHERE %s ORDER BY %s DESC
	LIMIT :limit OFFSET :offset;`, format, cond, order)

	params := map[string]any{
		messageFieldChannel:    chanID,
		"limit":                rpm.Limit,
		"offset":               rpm.Offset,
		messageFieldSubtopic:   rpm.Subtopic,
		messageFieldPublisher:  rpm.Publisher,
		messageFieldPublishers: rpm.Publishers,
		messageFieldDeviceIDs:  rpm.DeviceIDs,
		messageFieldName:       rpm.Name,
		messageFieldProtocol:   rpm.Protocol,
		messageFieldValue:      rpm.Value,
		"bool_value":           rpm.BoolValue,
		"string_value":         rpm.StringValue,
		"data_value":           rpm.DataValue,
		"from":                 rpm.From,
		"to":                   rpm.To,
	}
	rows, err := tr.db.NamedQuery(q, params)
	if err != nil {
		if pgErr, ok := pgError(err); ok {
			switch pgErr.Code {
			case pgerrcode.UndefinedTable:
				return readers.MessagesPage{}, nil
			case pgerrcode.UndefinedColumn:
				if isLegacyJSONDeviceFilter(format, rpm, pgErr) {
					return emptyPage(rpm), nil
				}
			}
		}
		return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}
	defer rows.Close()

	page := emptyPage(rpm)
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

	q = fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s;`, format, cond)
	rows, err = tr.db.NamedQuery(q, params)
	if err != nil {
		if pgErr, ok := pgError(err); ok {
			if pgErr.Code == pgerrcode.UndefinedColumn && isLegacyJSONDeviceFilter(format, rpm, pgErr) {
				return emptyPage(rpm), nil
			}
		}
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

func emptyPage(rpm readers.PageMetadata) readers.MessagesPage {
	return readers.MessagesPage{
		PageMetadata: rpm,
		Messages:     []readers.Message{},
	}
}

func pgError(err error) (*pgconn.PgError, bool) {
	if preErr, ok := err.(*pgconn.PrepareError); ok {
		err = preErr.Unwrap()
	}
	pgErr, ok := err.(*pgconn.PgError)
	return pgErr, ok
}

func isLegacyJSONDeviceFilter(format string, rpm readers.PageMetadata, pgErr *pgconn.PgError) bool {
	return format != defTable &&
		len(rpm.DeviceIDs) > 0 &&
		strings.Contains(pgErr.Message, messageFieldDeviceID)
}

func fmtCondition(chanID string, rpm readers.PageMetadata) string {
	condition := `channel = :channel`

	var query map[string]any
	meta, err := json.Marshal(rpm)
	if err != nil {
		return condition
	}
	if err := json.Unmarshal(meta, &query); err != nil {
		return condition
	}

	_, hasPublishers := query[messageFieldPublishers]

	for name := range query {
		switch name {
		case messageFieldPublisher:
			if hasPublishers {
				continue
			}
			condition = fmt.Sprintf(`%s AND %s = :%s`, condition, name, name)
		case messageFieldPublishers:
			condition = fmt.Sprintf(`%s AND %s = ANY(:%s)`, condition, messageFieldPublisher, messageFieldPublishers)
		case messageFieldDeviceIDs:
			condition = fmt.Sprintf(`%s AND %s = ANY(:%s)`, condition, messageFieldDeviceID, messageFieldDeviceIDs)
		case
			messageFieldSubtopic,
			messageFieldName,
			messageFieldProtocol:
			condition = fmt.Sprintf(`%s AND %s = :%s`, condition, name, name)
		case "v":
			comparator := readers.ParseValueComparator(query)
			condition = fmt.Sprintf(`%s AND value %s :value`, condition, comparator)
		case "vb":
			condition = fmt.Sprintf(`%s AND bool_value = :bool_value`, condition)
		case "vs":
			comparator := readers.ParseValueComparator(query)
			switch comparator {
			case "=":
				condition = fmt.Sprintf("%s AND string_value = :string_value ", condition)
			case ">":
				condition = fmt.Sprintf("%s AND string_value LIKE '%%' || :string_value || '%%' AND string_value <> :string_value", condition)
			case ">=":
				condition = fmt.Sprintf("%s AND string_value LIKE '%%' || :string_value || '%%'", condition)
			case "<=":
				condition = fmt.Sprintf("%s AND :string_value LIKE '%%' || string_value || '%%'", condition)
			case "<":
				condition = fmt.Sprintf("%s AND :string_value LIKE '%%' || string_value || '%%' AND string_value <> :string_value", condition)
			}
		case "vd":
			comparator := readers.ParseValueComparator(query)
			condition = fmt.Sprintf(`%s AND data_value %s :data_value`, condition, comparator)
		case "from":
			condition = fmt.Sprintf(`%s AND time >= :from`, condition)
		case "to":
			condition = fmt.Sprintf(`%s AND time < :to`, condition)
		}
	}
	return condition
}

type senmlMessage struct {
	ID string `db:"id"`
	senml.Message
}

type jsonMessage struct {
	ID        string `db:"id"`
	Channel   string `db:"channel"`
	Created   int64  `db:"created"`
	Subtopic  string `db:"subtopic"`
	Publisher string `db:"publisher"`
	Protocol  string `db:"protocol"`
	Payload   []byte `db:"payload"`
	DeviceID  string `db:"device_id"`
}

func (msg jsonMessage) toMap() (map[string]any, error) {
	ret := map[string]any{
		"id":                  msg.ID,
		messageFieldChannel:   msg.Channel,
		"created":             msg.Created,
		messageFieldSubtopic:  msg.Subtopic,
		messageFieldPublisher: msg.Publisher,
		messageFieldProtocol:  msg.Protocol,
		"payload":             map[string]any{},
	}
	// Mirrors the `omitempty` on senml.Message.DeviceId: rows with no device —
	// direct publishers, or anything written before this column existed — are
	// reported exactly as they were before.
	if msg.DeviceID != "" {
		ret[messageFieldDeviceID] = msg.DeviceID
	}
	pld := make(map[string]any)
	if err := json.Unmarshal(msg.Payload, &pld); err != nil {
		return nil, err
	}
	ret["payload"] = pld
	return ret, nil
}
