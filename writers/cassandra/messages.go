// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cassandra

import (
	"github.com/gocql/gocql"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/transformers/senml"
	"github.com/mainflux/mainflux/writers"
)

var errSaveMessage = errors.New("failed to save message to cassandra database")

var _ writers.MessageRepository = (*cassandraRepository)(nil)

type cassandraRepository struct {
	session *gocql.Session
}

// New instantiates Cassandra message repository.
func New(session *gocql.Session) writers.MessageRepository {
	return &cassandraRepository{session}
}

func (cr *cassandraRepository) Save(messages ...senml.Message) error {
	cql := `INSERT INTO messages (id, channel, subtopic, publisher, protocol,
			name, unit, value, string_value, bool_value, data_value, sum,
			time, update_time)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	id := gocql.TimeUUID()

	for _, msg := range messages {
		err := cr.session.Query(cql, id, msg.Channel, msg.Subtopic, msg.Publisher,
			msg.Protocol, msg.Name, msg.Unit, msg.Value, msg.StringValue,
			msg.BoolValue, msg.DataValue, msg.Sum, msg.Time, msg.UpdateTime).Exec()
		if err != nil {
			return errors.Wrap(errSaveMessage, err)
		}
	}

	return nil
}
