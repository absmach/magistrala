// Package writer provides message writer concept definitions.
package normalizer

import (
	"github.com/cisco/senml"
	"github.com/mainflux/mainflux/writer"
	"go.uber.org/zap"
)

// Message represents a message emitted by the mainflux adapters layer.
type Message struct {
	Channel     string `json:"channel"`
	Publisher   string `json:"publisher"`
	Protocol    string `json:"protocol"`
	ContentType string `json:"content_type"`
	Payload     []byte `json:"payload"`
}

func Normalize(logger *zap.Logger, msg Message) (msgs []writer.Message, err error) {
	var s, n senml.SenML

	if s, err = senml.Decode(msg.Payload, senml.JSON); err != nil {
		logger.Error("Failed to decode SenML", zap.Error(err))
		return nil, err
	}

	n = senml.Normalize(s)

	msgs = make([]writer.Message, len(n.Records))
	for k, v := range n.Records {
		m := writer.Message{}
		m.Channel = msg.Channel
		m.Publisher = msg.Publisher
		m.Protocol = msg.Protocol

		m.Version = v.BaseVersion
		m.Name = v.Name
		m.Unit = v.Unit
		if v.Value != nil {
			m.Value = *v.Value
		}
		m.StringValue = v.StringValue
		if v.BoolValue != nil {
			m.BoolValue = *v.BoolValue
		}
		m.DataValue = v.DataValue
		if v.Sum != nil {
			m.ValueSum = *v.Sum
		}
		m.Time = v.Time
		m.UpdateTime = v.UpdateTime
		m.Link = v.Link

		msgs[k] = m
	}

	return msgs, nil
}
