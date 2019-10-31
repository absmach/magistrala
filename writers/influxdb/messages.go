// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package influxdb

import (
	"math"
	"strconv"
	"time"

	"github.com/mainflux/mainflux/writers"

	influxdata "github.com/influxdata/influxdb/client/v2"
	"github.com/mainflux/mainflux"
)

const pointName = "messages"

var _ writers.MessageRepository = (*influxRepo)(nil)

type influxRepo struct {
	client influxdata.Client
	cfg    influxdata.BatchPointsConfig
}

type fields map[string]interface{}
type tags map[string]string

// New returns new InfluxDB writer.
func New(client influxdata.Client, database string) writers.MessageRepository {
	return &influxRepo{
		client: client,
		cfg: influxdata.BatchPointsConfig{
			Database: database,
		},
	}
}

func (repo *influxRepo) Save(messages ...mainflux.Message) error {
	pts, err := influxdata.NewBatchPoints(repo.cfg)
	if err != nil {
		return err
	}
	for _, msg := range messages {
		tgs, flds := repo.tagsOf(&msg), repo.fieldsOf(&msg)

		sec, dec := math.Modf(msg.Time)
		t := time.Unix(int64(sec), int64(dec*(1e9)))

		pt, err := influxdata.NewPoint(pointName, tgs, flds, t)
		if err != nil {
			return err
		}
		pts.AddPoint(pt)
	}

	return repo.client.Write(pts)
}

func (repo *influxRepo) tagsOf(msg *mainflux.Message) tags {
	return tags{
		"channel":   msg.Channel,
		"subtopic":  msg.Subtopic,
		"publisher": msg.Publisher,
		"name":      msg.Name,
	}
}

func (repo *influxRepo) fieldsOf(msg *mainflux.Message) fields {
	updateTime := strconv.FormatFloat(msg.UpdateTime, 'f', -1, 64)
	ret := fields{
		"protocol":   msg.Protocol,
		"unit":       msg.Unit,
		"link":       msg.Link,
		"updateTime": updateTime,
	}

	switch msg.Value.(type) {
	case *mainflux.Message_FloatValue:
		ret["value"] = msg.GetFloatValue()
	case *mainflux.Message_StringValue:
		ret["stringValue"] = msg.GetStringValue()
	case *mainflux.Message_DataValue:
		ret["dataValue"] = msg.GetDataValue()
	case *mainflux.Message_BoolValue:
		ret["boolValue"] = msg.GetBoolValue()
	}

	if msg.ValueSum != nil {
		ret["valueSum"] = msg.GetValueSum().GetValue()
	}

	return ret
}
