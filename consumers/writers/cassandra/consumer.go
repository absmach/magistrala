// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cassandra

import (
	"encoding/json"
	"fmt"

	"github.com/gocql/gocql"
	"github.com/mainflux/mainflux/consumers"
	"github.com/mainflux/mainflux/pkg/errors"
	mfjson "github.com/mainflux/mainflux/pkg/transformers/json"
	"github.com/mainflux/mainflux/pkg/transformers/senml"
)

var (
	errSaveMessage = errors.New("failed to save message to cassandra database")
	errNoTable     = errors.New("table does not exist")
)
var _ consumers.Consumer = (*cassandraRepository)(nil)

type cassandraRepository struct {
	session *gocql.Session
}

// New instantiates Cassandra message repository.
func New(session *gocql.Session) consumers.Consumer {
	return &cassandraRepository{session}
}

func (cr *cassandraRepository) Consume(message interface{}) error {
	switch m := message.(type) {
	case mfjson.Messages:
		return cr.saveJSON(m)
	default:
		return cr.saveSenml(m)
	}
}

func (cr *cassandraRepository) saveSenml(messages interface{}) error {
	msgs, ok := messages.([]senml.Message)
	if !ok {
		return errSaveMessage
	}
	cql := `INSERT INTO messages (id, channel, subtopic, publisher, protocol,
            name, unit, value, string_value, bool_value, data_value, sum,
            time, update_time)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	id := gocql.TimeUUID()

	for _, msg := range msgs {
		err := cr.session.Query(cql, id, msg.Channel, msg.Subtopic, msg.Publisher,
			msg.Protocol, msg.Name, msg.Unit, msg.Value, msg.StringValue,
			msg.BoolValue, msg.DataValue, msg.Sum, msg.Time, msg.UpdateTime).Exec()
		if err != nil {
			return errors.Wrap(errSaveMessage, err)
		}
	}

	return nil
}

func (cr *cassandraRepository) saveJSON(msgs mfjson.Messages) error {
	if err := cr.insertJSON(msgs); err != nil {
		if err == errNoTable {
			if err := cr.createTable(msgs.Format); err != nil {
				return err
			}
			return cr.insertJSON(msgs)
		}
		return err
	}
	return nil
}

func (cr *cassandraRepository) insertJSON(msgs mfjson.Messages) error {
	cql := `INSERT INTO %s (id, channel, created, subtopic, publisher, protocol, payload) VALUES (?, ?, ?, ?, ?, ?, ?)`
	cql = fmt.Sprintf(cql, msgs.Format)
	for _, msg := range msgs.Data {
		pld, err := json.Marshal(msg.Payload)
		if err != nil {
			return err
		}
		id := gocql.TimeUUID()

		err = cr.session.Query(cql, id, msg.Channel, msg.Created, msg.Subtopic, msg.Publisher, msg.Protocol, string(pld)).Exec()
		if err != nil {
			if err.Error() == fmt.Sprintf("unconfigured table %s", msgs.Format) {
				return errNoTable
			}
			return errors.Wrap(errSaveMessage, err)
		}
	}
	return nil
}

func (cr *cassandraRepository) createTable(name string) error {
	q := fmt.Sprintf(jsonTable, name)
	return cr.session.Query(q).Exec()
}
