// Package nats contains NATS-specific message repository implementation.
package nats

import (
	"encoding/json"

	"github.com/mainflux/mainflux/writer"
	broker "github.com/nats-io/go-nats"
)

const topic string = "msg.http"

var _ writer.MessageRepository = (*natsRepository)(nil)

type natsRepository struct {
	nc *broker.Conn
}

// NewMessageRepository instantiates NATS message repository. Note that the
// repository will not truly persist messages, but instead they will be
// published to the topic and made available for persisting by all interested
// parties, i.e. the message-writer service.
func NewMessageRepository(nc *broker.Conn) writer.MessageRepository {
	return &natsRepository{nc}
}

func (repo *natsRepository) Save(msg writer.RawMessage) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return repo.nc.Publish(topic, b)
}
