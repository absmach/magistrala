//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package cassandra

import (
	"github.com/gocql/gocql"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/writers"
)

var _ writers.MessageRepository = (*cassandraRepository)(nil)

type cassandraRepository struct {
	session *gocql.Session
}

// New instantiates Cassandra message repository.
func New(session *gocql.Session) writers.MessageRepository {
	return &cassandraRepository{session}
}

func (cr *cassandraRepository) Save(msg mainflux.Message) error {
	cql := `INSERT INTO messages (id, channel, publisher, protocol, name, unit,
			value, string_value, bool_value, data_value, value_sum, time,
			update_time, link)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	id := gocql.TimeUUID()

	var floatVal, valSum *float64
	var strVal, dataVal *string
	var boolVal *bool
	switch msg.Value.(type) {
	case *mainflux.Message_FloatValue:
		v := msg.GetFloatValue()
		floatVal = &v
	case *mainflux.Message_StringValue:
		v := msg.GetStringValue()
		strVal = &v
	case *mainflux.Message_DataValue:
		v := msg.GetDataValue()
		dataVal = &v
	case *mainflux.Message_BoolValue:
		v := msg.GetBoolValue()
		boolVal = &v
	}

	if msg.GetValueSum() != nil {
		v := msg.GetValueSum().GetValue()
		valSum = &v
	}

	return cr.session.Query(cql, id, msg.GetChannel(), msg.GetPublisher(),
		msg.GetProtocol(), msg.GetName(), msg.GetUnit(), floatVal,
		strVal, boolVal, dataVal, valSum, msg.GetTime(), msg.GetUpdateTime(), msg.GetLink()).Exec()
}
