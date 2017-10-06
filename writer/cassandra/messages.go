package cassandra

import (
	"github.com/cisco/senml"
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

func normalize(msg writer.RawMessage) ([]writer.Message, error) {
	var (
		rm, nm senml.SenML // raw and normalized message
		err    error
	)

	if rm, err = senml.Decode(msg.Payload, senml.JSON); err != nil {
		return nil, err
	}

	nm = senml.Normalize(rm)

	msgs := make([]writer.Message, len(nm.Records))
	for k, v := range nm.Records {
		m := writer.Message{
			Channel:     msg.Channel,
			Publisher:   msg.Publisher,
			Protocol:    msg.Protocol,
			Version:     v.BaseVersion,
			Name:        v.Name,
			Unit:        v.Unit,
			StringValue: v.StringValue,
			DataValue:   v.DataValue,
			Time:        v.Time,
			UpdateTime:  v.UpdateTime,
			Link:        v.Link,
		}

		if v.Value != nil {
			m.Value = *v.Value
		}

		if v.BoolValue != nil {
			m.BoolValue = *v.BoolValue
		}

		if v.Sum != nil {
			m.ValueSum = *v.Sum
		}

		msgs[k] = m
	}

	return msgs, nil
}

func (repo *msgRepository) Save(raw writer.RawMessage) error {
	var (
		msgs []writer.Message
		err  error
	)

	if msgs, err = normalize(raw); err != nil {
		return err
	}

	for _, msg := range msgs {
		cql := `INSERT INTO messages_by_channel
				(channel, id, publisher, protocol, bver, n, u, v, vs, vb, vd, s, t, ut, l)
				VALUES (?, now(), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

		err = repo.session.Query(cql, msg.Channel, msg.Publisher, msg.Protocol,
			msg.Version, msg.Name, msg.Unit, msg.Value, msg.StringValue, msg.BoolValue,
			msg.DataValue, msg.ValueSum, msg.Time, msg.UpdateTime, msg.Link).Exec()

		if err != nil {
			return err
		}
	}

	return nil
}
