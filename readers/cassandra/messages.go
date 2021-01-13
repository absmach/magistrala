// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cassandra

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gocql/gocql"
	"github.com/mainflux/mainflux/pkg/errors"
	jsont "github.com/mainflux/mainflux/pkg/transformers/json"
	"github.com/mainflux/mainflux/pkg/transformers/senml"
	"github.com/mainflux/mainflux/readers"
)

var errReadMessages = errors.New("failed to read messages from cassandra database")

const (
	format = "format"
	// Table for SenML messages
	defTable = "messages"
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

func (cr cassandraRepository) ReadAll(chanID string, offset, limit uint64, query map[string]string) (readers.MessagesPage, error) {
	table, ok := query[format]
	if !ok {
		table = defTable
	}
	// Remove format filter and format the rest properly.
	delete(query, format)

	q, vals := buildQuery(chanID, offset, limit, query)

	selectCQL := fmt.Sprintf(`SELECT channel, subtopic, publisher, protocol, name, unit,
		value, string_value, bool_value, data_value, sum, time,
		update_time FROM messages WHERE channel = ? %s LIMIT ?
		ALLOW FILTERING`, q)
	countCQL := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE channel = ? %s ALLOW FILTERING`, defTable, q)

	if table != defTable {
		selectCQL = fmt.Sprintf(`SELECT channel, subtopic, publisher, protocol, created, payload FROM %s WHERE channel = ? %s LIMIT ?
			ALLOW FILTERING`, table, q)
		countCQL = fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE channel = ? %s ALLOW FILTERING`, table, q)
	}

	iter := cr.session.Query(selectCQL, vals...).Iter()
	defer iter.Close()
	scanner := iter.Scanner()

	// skip first OFFSET rows
	for i := uint64(0); i < offset; i++ {
		if !scanner.Next() {
			break
		}
	}

	page := readers.MessagesPage{
		Offset:   offset,
		Limit:    limit,
		Messages: []readers.Message{},
	}

	switch table {
	case defTable:
		for scanner.Next() {
			var msg senml.Message
			err := scanner.Scan(&msg.Channel, &msg.Subtopic, &msg.Publisher, &msg.Protocol,
				&msg.Name, &msg.Unit, &msg.Value, &msg.StringValue, &msg.BoolValue,
				&msg.DataValue, &msg.Sum, &msg.Time, &msg.UpdateTime)
			if err != nil {
				return readers.MessagesPage{}, errors.Wrap(errReadMessages, err)
			}
			page.Messages = append(page.Messages, msg)
		}
	default:
		for scanner.Next() {
			var msg jsonMessage
			err := scanner.Scan(&msg.Channel, &msg.Subtopic, &msg.Publisher, &msg.Protocol, &msg.Created, &msg.Payload)
			if err != nil {
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

	if err := cr.session.Query(countCQL, vals[:len(vals)-1]...).Scan(&page.Total); err != nil {
		return readers.MessagesPage{}, errors.Wrap(errReadMessages, err)
	}

	return page, nil
}

func buildQuery(chanID string, offset, limit uint64, query map[string]string) (string, []interface{}) {
	var condCQL string
	vals := []interface{}{chanID}

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
			fVal, err := strconv.ParseFloat(val, 64)
			if err != nil {
				continue
			}
			vals = append(vals, fVal)
			condCQL = fmt.Sprintf(`%s AND value = ?`, condCQL)
		case "vb":
			bVal, err := strconv.ParseBool(val)
			if err != nil {
				continue
			}
			vals = append(vals, bVal)
			condCQL = fmt.Sprintf(`%s AND bool_value = ?`, condCQL)
		case "vs":
			vals = append(vals, val)
			condCQL = fmt.Sprintf(`%s AND string_value = ?`, condCQL)
		case "vd":
			vals = append(vals, val)
			condCQL = fmt.Sprintf(`%s AND data_value = ?`, condCQL)
		case "from":
			fVal, err := strconv.ParseFloat(val, 64)
			if err != nil {
				continue
			}
			vals = append(vals, fVal)
			condCQL = fmt.Sprintf(`%s AND time >= ?`, condCQL)
		case "to":
			fVal, err := strconv.ParseFloat(val, 64)
			if err != nil {
				continue
			}
			vals = append(vals, fVal)
			condCQL = fmt.Sprintf(`%s AND time < ?`, condCQL)
		}
	}
	vals = append(vals, offset+limit)

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
