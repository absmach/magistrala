// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cassandra

import (
	"encoding/json"
	"fmt"

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

	names := []string{}
	vals := []interface{}{chanID}
	for name, val := range query {
		names = append(names, name)
		vals = append(vals, val)
	}
	vals = append(vals, offset+limit)

	selectCQL := buildSelectQuery(table, chanID, offset, limit, names)
	countCQL := buildCountQuery(table, chanID, names)

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

func buildSelectQuery(table, chanID string, offset, limit uint64, names []string) string {
	var condCQL string
	cql := `SELECT channel, subtopic, publisher, protocol, name, unit,
	        value, string_value, bool_value, data_value, sum, time,
			update_time FROM messages WHERE channel = ? %s LIMIT ?
			ALLOW FILTERING`
	if table != defTable {
		cql = fmt.Sprintf(`SELECT channel, subtopic, publisher, protocol, created, payload FROM %s WHERE channel = ? %s LIMIT ?
			ALLOW FILTERING`, table, "%s")
	}
	for _, name := range names {
		switch name {
		case
			"channel",
			"subtopic",
			"publisher",
			"name",
			"protocol":
			condCQL = fmt.Sprintf(`%s AND %s = ?`, condCQL, name)
		}
	}

	return fmt.Sprintf(cql, condCQL)
}

func buildCountQuery(table, chanID string, names []string) string {
	var condCQL string
	cql := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE channel = ? %s ALLOW FILTERING`, table, "%s")

	for _, name := range names {
		switch name {
		case
			"channel",
			"subtopic",
			"publisher",
			"name",
			"protocol":
			condCQL = fmt.Sprintf(`%s AND %s = ?`, condCQL, name)
		}
	}

	return fmt.Sprintf(cql, condCQL)
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
