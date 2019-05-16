//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package postgres

import (
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx" // required for DB access
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/readers"
)

const errInvalid = "invalid_text_representation"

var errInvalidMessage = errors.New("invalid message representation")

var _ readers.MessageRepository = (*postgresRepository)(nil)

type postgresRepository struct {
	db *sqlx.DB
}

// New returns new PostgreSQL writer.
func New(db *sqlx.DB) readers.MessageRepository {
	return &postgresRepository{
		db: db,
	}
}

func (tr postgresRepository) ReadAll(chanID string, offset, limit uint64, query map[string]string) (readers.MessagesPage, error) {
	subtopicQuery := ""
	if query["subtopic"] != "" {
		subtopicQuery = `AND subtopic = :subtopic`
	}
	q := fmt.Sprintf(`SELECT * FROM messages
    WHERE channel = :channel %s ORDER BY time DESC
    LIMIT :limit OFFSET :offset;`, subtopicQuery)

	params := map[string]interface{}{
		"channel":  chanID,
		"limit":    limit,
		"offset":   offset,
		"subtopic": query["subtopic"],
	}

	rows, err := tr.db.NamedQuery(q, params)
	if err != nil {
		return readers.MessagesPage{}, err
	}
	defer rows.Close()

	page := readers.MessagesPage{
		Offset:   offset,
		Limit:    limit,
		Messages: []mainflux.Message{},
	}
	for rows.Next() {
		dbm := dbMessage{Channel: chanID}
		if err := rows.StructScan(&dbm); err != nil {
			return readers.MessagesPage{}, err
		}

		msg, err := toMessage(dbm)
		if err != nil {
			return readers.MessagesPage{}, err
		}

		page.Messages = append(page.Messages, msg)
	}

	q = `SELECT COUNT(*) FROM messages WHERE channel = $1;`
	qParams := []interface{}{chanID}

	if query["subtopic"] != "" {
		q = `SELECT COUNT(*) FROM messages WHERE channel = $1 AND subtopic = $2;`
		qParams = append(qParams, query["subtopic"])
	}

	if err := tr.db.QueryRow(q, qParams...).Scan(&page.Total); err != nil {
		return readers.MessagesPage{}, err
	}

	return page, nil
}

type dbMessage struct {
	ID          string   `db:"id"`
	Channel     string   `db:"channel"`
	Subtopic    string   `db:"subtopic"`
	Publisher   string   `db:"publisher"`
	Protocol    string   `db:"protocol"`
	Name        string   `db:"name"`
	Unit        string   `db:"unit"`
	FloatValue  *float64 `db:"value"`
	StringValue *string  `db:"string_value"`
	BoolValue   *bool    `db:"bool_value"`
	DataValue   *string  `db:"data_value"`
	ValueSum    *float64 `db:"value_sum"`
	Time        float64  `db:"time"`
	UpdateTime  float64  `db:"update_time"`
	Link        string   `db:"link"`
}

func toMessage(dbm dbMessage) (mainflux.Message, error) {
	msg := mainflux.Message{
		Channel:    dbm.Channel,
		Subtopic:   dbm.Subtopic,
		Publisher:  dbm.Publisher,
		Protocol:   dbm.Protocol,
		Name:       dbm.Name,
		Unit:       dbm.Unit,
		Time:       dbm.Time,
		UpdateTime: dbm.UpdateTime,
		Link:       dbm.Link,
	}

	switch {
	case dbm.FloatValue != nil:
		msg.Value = &mainflux.Message_FloatValue{FloatValue: *dbm.FloatValue}
	case dbm.StringValue != nil:
		msg.Value = &mainflux.Message_StringValue{StringValue: *dbm.StringValue}
	case dbm.BoolValue != nil:
		msg.Value = &mainflux.Message_BoolValue{BoolValue: *dbm.BoolValue}
	case dbm.DataValue != nil:
		msg.Value = &mainflux.Message_DataValue{DataValue: *dbm.DataValue}
	case dbm.ValueSum != nil:
		msg.ValueSum = &mainflux.SumValue{Value: *dbm.ValueSum}
	}

	return msg, nil
}
