package writers

import (
	"fmt"

	"github.com/go-kit/kit/metrics"
	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/mainflux"
	log "github.com/mainflux/mainflux/logger"
	nats "github.com/nats-io/go-nats"
)

const senML = "out.senml"

type consumer struct {
	nc     *nats.Conn
	logger log.Logger
	name   string
	repo   MessageRepository
}

// Start method starts to consume normalized messages received from NATS.
func Start(name string, nc *nats.Conn, logger log.Logger, repo MessageRepository, counter metrics.Counter, latency metrics.Histogram) error {
	repo = newMetricsMiddleware(repo, counter, latency)
	consumer := consumer{
		nc:     nc,
		logger: logger,
		name:   name,
		repo:   repo,
	}

	_, err := nc.Subscribe(senML, consumer.consume)
	return err
}

func (c *consumer) consume(m *nats.Msg) {
	msg := &mainflux.Message{}

	if err := proto.Unmarshal(m.Data, msg); err != nil {
		c.logger.Warn(fmt.Sprintf("%s failed to unmarshal received message: %s", c.name, err))
		return
	}

	if err := c.repo.Save(*msg); err != nil {
		c.logger.Warn(fmt.Sprintf("%s failed to save message: %s", c.name, err))
		return
	}
}
