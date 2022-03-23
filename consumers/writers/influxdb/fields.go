package influxdb

import (
	"github.com/mainflux/mainflux/pkg/transformers/senml"
)

type fields map[string]interface{}

func senmlFields(msg senml.Message) fields {
	ret := fields{
		"protocol":   msg.Protocol,
		"unit":       msg.Unit,
		"updateTime": msg.UpdateTime,
	}

	switch {
	case msg.Value != nil:
		ret["value"] = *msg.Value
	case msg.StringValue != nil:
		ret["stringValue"] = *msg.StringValue
	case msg.DataValue != nil:
		ret["dataValue"] = *msg.DataValue
	case msg.BoolValue != nil:
		ret["boolValue"] = *msg.BoolValue
	}

	if msg.Sum != nil {
		ret["sum"] = *msg.Sum
	}

	return ret
}
