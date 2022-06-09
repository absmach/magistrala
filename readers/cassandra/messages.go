// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cassandra

import (
	"encoding/json"
	"fmt"

	"github.com/gocql/gocql"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/transformers/senml"
	"github.com/mainflux/mainflux/readers"
)

const (
	// Table for SenML messages
	defTable = "messages"

	// Error code for Undefined table error.
	undefinedTableCode = 8704
)

var _ readers.MessageRepository = (*cassandraRepository)(nil)

type cassandraRepository struct {
	session *gocql.Session
}

// New instantiates Cassandra message repository.
func New(session *gocql.Session) readers.MessageRepository {
	return cassandraRepository{
		session: session,
	}
}

func (cr cassandraRepository) ReadAll(chanID string, rpm readers.PageMetadata) (readers.MessagesPage, error) {
	format := defTable
	if rpm.Format != "" {
		format = rpm.Format
	}

	q, vals := buildQuery(chanID, rpm)

	selectCQL := fmt.Sprintf(`SELECT channel, subtopic, publisher, protocol, name, unit,
		value, string_value, bool_value, data_value, sum, time,
		update_time FROM messages WHERE channel = ? %s LIMIT ?
		ALLOW FILTERING`, q)
	countCQL := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE channel = ? %s ALLOW FILTERING`, format, q)

	if format != defTable {
		selectCQL = fmt.Sprintf(`SELECT channel, subtopic, publisher, protocol, created, payload FROM %s WHERE channel = ? %s LIMIT ?
			ALLOW FILTERING`, format, q)
		countCQL = fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE channel = ? %s ALLOW FILTERING`, format, q)
	}

	iter := cr.session.Query(selectCQL, vals...).Iter()
	defer iter.Close()
	scanner := iter.Scanner()

	// skip first OFFSET rows
	for i := uint64(0); i < rpm.Offset; i++ {
		if !scanner.Next() {
			break
		}
	}

	page := readers.MessagesPage{
		PageMetadata: rpm,
		Messages:     []readers.Message{},
	}

	switch format {
	case defTable:
		for scanner.Next() {
			var msg senml.Message
			err := scanner.Scan(&msg.Channel, &msg.Subtopic, &msg.Publisher, &msg.Protocol,
				&msg.Name, &msg.Unit, &msg.Value, &msg.StringValue, &msg.BoolValue,
				&msg.DataValue, &msg.Sum, &msg.Time, &msg.UpdateTime)
			if err != nil {
				if e, ok := err.(gocql.RequestError); ok {
					if e.Code() == undefinedTableCode {
						return readers.MessagesPage{}, nil
					}
				}
				return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
			}
			page.Messages = append(page.Messages, msg)
		}
	default:
		for scanner.Next() {
			var msg jsonMessage
			err := scanner.Scan(&msg.Channel, &msg.Subtopic, &msg.Publisher, &msg.Protocol, &msg.Created, &msg.Payload)
			if err != nil {
				if e, ok := err.(gocql.RequestError); ok {
					if e.Code() == undefinedTableCode {
						return readers.MessagesPage{}, nil
					}
				}
				return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
			}
			m, err := msg.toMap()
			if err != nil {
				return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
			}
			page.Messages = append(page.Messages, m)
		}
	}

	if err := cr.session.Query(countCQL, vals[:len(vals)-1]...).Scan(&page.Total); err != nil {
		if e, ok := err.(gocql.RequestError); ok {
			if e.Code() == undefinedTableCode {
				return readers.MessagesPage{}, nil
			}
		}
		return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}

	return page, nil
}

func buildQuery(chanID string, rpm readers.PageMetadata) (string, []interface{}) {
	var condCQL string
	vals := []interface{}{chanID}

	var query map[string]interface{}
	meta, err := json.Marshal(rpm)
	if err != nil {
		return condCQL, vals
	}
	json.Unmarshal(meta, &query)

	for name, val := range query {
		switch name {
		case
			"channel",
			"subtopic",
			"publisher",
			"name",
			"protocol":
			vals = append(vals, val)
			condCQL = fmt.Sprintf(`%s AND %s = ?`, condCQL, name)
		case "v":
			vals = append(vals, val)
			comparator := readers.ParseValueComparator(query)
			condCQL = fmt.Sprintf(`%s AND value %s ?`, condCQL, comparator)
		case "vb":
			vals = append(vals, val)
			condCQL = fmt.Sprintf(`%s AND bool_value = ?`, condCQL)
		case "vs":
			vals = append(vals, val)
			condCQL = fmt.Sprintf(`%s AND string_value = ?`, condCQL)
		case "vd":
			vals = append(vals, val)
			condCQL = fmt.Sprintf(`%s AND data_value = ?`, condCQL)
		case "from":
			vals = append(vals, val)
			condCQL = fmt.Sprintf(`%s AND time >= ?`, condCQL)
		case "to":
			vals = append(vals, val)
			condCQL = fmt.Sprintf(`%s AND time < ?`, condCQL)
		}
	}
	vals = append(vals, rpm.Offset+rpm.Limit)

	return condCQL, vals
}

type jsonMessage struct {
	ID        string
	Channel   string
	Created   int64
	Subtopic  string
	Publisher string
	Protocol  string
	Payload   string
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
	if err := json.Unmarshal([]byte(msg.Payload), &pld); err != nil {
		return nil, err
	}
	ret["payload"] = pld
	return ret, nil
}
