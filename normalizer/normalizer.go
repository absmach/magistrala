package normalizer

import (
	"fmt"
	"strings"

	"github.com/cisco/senml"
	"github.com/go-kit/kit/metrics"
	"github.com/golang/protobuf/proto"
	"github.com/mainflux/mainflux"
	log "github.com/mainflux/mainflux/logger"
	nats "github.com/nats-io/go-nats"
)

const (
	queue         = "normalizers"
	input         = "channel.*"
	outputUnknown = "out.unknown"
	senML         = "application/senml+json"
)

type eventFlow struct {
	nc     *nats.Conn
	logger log.Logger
}

// Subscribe instantiates and starts a new NATS message flow.
func Subscribe(nc *nats.Conn, logger log.Logger, counter metrics.Counter, latency metrics.Histogram) {
	flow := eventFlow{nc, logger}
	mm := newMetricsMiddleware(flow, counter, latency)
	flow.nc.QueueSubscribe(input, queue, mm.handleMessage)
}

func (ef eventFlow) handleMsg(m *nats.Msg) {
	msg := mainflux.RawMessage{}

	if err := proto.Unmarshal(m.Data, &msg); err != nil {
		ef.logger.Warn(fmt.Sprintf("Unmarshalling failed: %s", err))
		return
	}

	if err := ef.publish(msg); err != nil {
		ef.logger.Warn(fmt.Sprintf("Publishing failed: %s", err))
		return
	}
}

func (ef eventFlow) publish(msg mainflux.RawMessage) error {
	output := mainflux.OutputSenML
	normalized, err := ef.normalize(msg)
	if err != nil {
		ef.logger.Warn(fmt.Sprintf("Normalization failed: %s", err))
		switch ct := strings.ToLower(msg.ContentType); ct {
		case senML:
			return err
		case "":
			output = outputUnknown
		default:
			output = fmt.Sprintf("out.%s", ct)
		}
	}

	for _, v := range normalized {
		data, err := proto.Marshal(&v)
		if err != nil {
			ef.logger.Warn(fmt.Sprintf("Marshalling failed: %s", err))
			return err
		}

		if err = ef.nc.Publish(output, data); err != nil {
			ef.logger.Warn(fmt.Sprintf("Publishing failed: %s", err))
			return err
		}
	}

	return nil
}

func (ef eventFlow) normalize(msg mainflux.RawMessage) ([]mainflux.Message, error) {
	var (
		raw, normalized senml.SenML
		err             error
	)

	if raw, err = senml.Decode(msg.Payload, senml.JSON); err != nil {
		return nil, err
	}

	normalized = senml.Normalize(raw)

	msgs := make([]mainflux.Message, len(normalized.Records))
	for k, v := range normalized.Records {
		m := mainflux.Message{
			Channel:     msg.Channel,
			Publisher:   msg.Publisher,
			Protocol:    msg.Protocol,
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
