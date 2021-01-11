// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx" // required for DB access
	"github.com/mainflux/mainflux/pkg/errors"
	jsont "github.com/mainflux/mainflux/pkg/transformers/json"
	"github.com/mainflux/mainflux/pkg/transformers/senml"
	"github.com/mainflux/mainflux/readers"
)

const errInvalid = "invalid_text_representation"

const (
	format = "format"
	// Table for SenML messages
	defTable = "messages"
)

var errReadMessages = errors.New("failed to read messages from postgres database")

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
	table, ok := query[format]
	order := "created"
	if !ok {
		table = defTable
		order = "time"
	}
	// Remove format filter and format the rest properly.
	delete(query, format)
	q := fmt.Sprintf(`SELECT * FROM %s
    WHERE %s ORDER BY %s DESC
	LIMIT :limit OFFSET :offset;`, table, fmtCondition(chanID, query), order)

	params := map[string]interface{}{
		"channel":      chanID,
		"limit":        limit,
		"offset":       offset,
		"subtopic":     query["subtopic"],
		"publisher":    query["publisher"],
		"name":         query["name"],
		"protocol":     query["protocol"],
		"value":        query["v"],
		"bool_value":   query["vb"],
		"string_value": query["vs"],
		"data_value":   query["vd"],
		"from":         query["from"],
		"to":           query["to"],
	}

	rows, err := tr.db.NamedQuery(q, params)
	if err != nil {
		return readers.MessagesPage{}, errors.Wrap(errReadMessages, err)
	}
	defer rows.Close()

	page := readers.MessagesPage{
		Offset:   offset,
		Limit:    limit,
		Messages: []readers.Message{},
	}
	switch table {
	case defTable:
		for rows.Next() {
			msg := dbMessage{Message: senml.Message{}}
			if err := rows.StructScan(&msg); err != nil {
				return readers.MessagesPage{}, errors.Wrap(errReadMessages, err)
			}

			page.Messages = append(page.Messages, msg.Message)
		}
	default:
		for rows.Next() {
			msg := jsonMessage{}
			if err := rows.StructScan(&msg); err != nil {
				return readers.MessagesPage{}, errors.Wrap(errReadMessages, err)
			}
			m, err := msg.toMap()
			if err != nil {
				return readers.MessagesPage{}, errors.Wrap(errReadMessages, err)
			}
			m["payload"] = jsont.ParseFlat(m["payload"])
			page.Messages = append(page.Messages, m)
		}

	}

	q = fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s;`, table, fmtCondition(chanID, query))
	rows, err = tr.db.NamedQuery(q, params)
	if err != nil {
		return readers.MessagesPage{}, errors.Wrap(errReadMessages, err)
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

func fmtCondition(chanID string, query map[string]string) string {
	condition := `channel = :channel`
	for name := range query {
		switch name {
		case
			"subtopic",
			"publisher",
			"name",
			"protocol":
			condition = fmt.Sprintf(`%s AND %s = :%s`, condition, name, name)
		case "v":
			condition = fmt.Sprintf(`%s AND value = :value`, condition)
		case "vb":
			condition = fmt.Sprintf(`%s AND bool_value = :bool_value`, condition)
		case "vs":
			condition = fmt.Sprintf(`%s AND string_value = :string_value`, condition)
		case "vd":
			condition = fmt.Sprintf(`%s AND data_value = :data_value`, condition)
		case "from":
			condition = fmt.Sprintf(`%s AND time >= :from`, condition)
		case "to":
			condition = fmt.Sprintf(`%s AND time < :to`, condition)
		}
	}
	return condition
}

type dbMessage struct {
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
}

func (msg jsonMessage) toMap() (map[string]interface{}, error) {
	ret := map[string]interface{}{
		"id":        msg.ID,
		"channel":   msg.Channel,
		"created":   msg.Created,
		"subtopic":  msg.Subtopic,
		"publisher": msg.Publisher,
		"protocol":  msg.Protocol,
		"payload":   map[string]interface{}{},
	}
	pld := make(map[string]interface{})
	if err := json.Unmarshal(msg.Payload, &pld); err != nil {
		return nil, err
	}
	ret["payload"] = pld
	return ret, nil
}
