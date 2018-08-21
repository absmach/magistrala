package normalizer

import (
	"strings"

	"github.com/cisco/senml"
	"github.com/mainflux/mainflux"
)

type normalizer struct{}

// New returns normalizer service implementation.
func New() Service {
	return normalizer{}
}

func (n normalizer) Normalize(msg mainflux.RawMessage) (NormalizedData, error) {
	raw, err := senml.Decode(msg.Payload, senml.JSON)
	if err != nil {
		return NormalizedData{}, err
	}

	normalized := senml.Normalize(raw)

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

	output := strings.ToLower(msg.ContentType)

	return NormalizedData{
		ContentType: output,
		Messages:    msgs,
	}, nil
}
