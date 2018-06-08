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

	return cr.session.Query(cql, id, msg.GetChannel(), msg.GetPublisher(),
		msg.GetProtocol(), msg.GetName(), msg.GetUnit(), msg.GetValue(),
		msg.GetStringValue(), msg.GetBoolValue(), msg.GetDataValue(),
		msg.GetValueSum(), msg.GetTime(), msg.GetUpdateTime(), msg.GetLink()).Exec()
}
