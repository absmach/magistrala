package cassandra

import (
	"github.com/gocql/gocql"
	"github.com/mainflux/mainflux/writer"
)

var _ writer.MessageRepository = (*msgRepository)(nil)

type msgRepository struct {
	session *gocql.Session
}

// NewMessageRepository instantiates Cassandra message repository.
func NewMessageRepository(session *gocql.Session) writer.MessageRepository {
	return &msgRepository{session}
}

func (repo *msgRepository) Save(msg writer.Message) error {
	cql := `INSERT INTO messages_by_channel
			(channel, id, publisher, protocol, bn, bt, bu, bv, bs, bver, n, u, v, vs, vb, vd, s, t, ut, l)
			VALUES (?, now(), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	return repo.session.Query(cql, msg.Channel, msg.Publisher, msg.Protocol,
		msg.BaseName, msg.BaseTime, msg.BaseUnit, msg.BaseValue, msg.BaseSum,
		msg.Version, msg.Name, msg.Unit, msg.Value, msg.StringValue, msg.BoolValue,
		msg.DataValue, msg.ValueSum, msg.Time, msg.UpdateTime, msg.Link).Exec()
}
