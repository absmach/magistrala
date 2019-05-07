//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package cassandra

import (
	"fmt"

	"github.com/gocql/gocql"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/readers"
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
	names := []string{}
	vals := []interface{}{chanID}
	for name, val := range query {
		names = append(names, name)
		vals = append(vals, val)
	}
	vals = append(vals, offset+limit)

	selectCQL := buildSelectQuery(chanID, offset, limit, names)
	countCQL := buildCountQuery(chanID, names)

	iter := cr.session.Query(selectCQL, vals...).Iter()
	defer iter.Close()
	scanner := iter.Scanner()

	// skip first OFFSET rows
	for i := uint64(0); i < offset; i++ {
		if !scanner.Next() {
			break
		}
	}

	var floatVal, valueSum *float64
	var strVal, dataVal *string
	var boolVal *bool

	page := readers.MessagesPage{
		Offset:   offset,
		Limit:    limit,
		Messages: []mainflux.Message{},
	}
	for scanner.Next() {
		var msg mainflux.Message
		err := scanner.Scan(&msg.Channel, &msg.Subtopic, &msg.Publisher, &msg.Protocol,
			&msg.Name, &msg.Unit, &floatVal, &strVal, &boolVal,
			&dataVal, &valueSum, &msg.Time, &msg.UpdateTime, &msg.Link)
		if err != nil {
			return readers.MessagesPage{}, err
		}

		switch {
		case floatVal != nil:
			msg.Value = &mainflux.Message_FloatValue{FloatValue: *floatVal}
		case strVal != nil:
			msg.Value = &mainflux.Message_StringValue{StringValue: *strVal}
		case boolVal != nil:
			msg.Value = &mainflux.Message_BoolValue{BoolValue: *boolVal}
		case dataVal != nil:
			msg.Value = &mainflux.Message_DataValue{DataValue: *dataVal}
		}

		if valueSum != nil {
			msg.ValueSum = &mainflux.SumValue{Value: *valueSum}
		}

		page.Messages = append(page.Messages, msg)
	}

	if err := cr.session.Query(countCQL, vals[:len(vals)-1]...).Scan(&page.Total); err != nil {
		return readers.MessagesPage{}, err
	}

	return page, nil
}

func buildSelectQuery(chanID string, offset, limit uint64, names []string) string {
	var condCQL string
	cql := `SELECT channel, subtopic, publisher, protocol, name, unit,
	        value, string_value, bool_value, data_value, value_sum, time,
			update_time, link FROM messages WHERE channel = ? %s LIMIT ?
			ALLOW FILTERING`

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

func buildCountQuery(chanID string, names []string) string {
	var condCQL string
	cql := `SELECT COUNT(*) FROM messages WHERE channel = ? %s ALLOW FILTERING`

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
