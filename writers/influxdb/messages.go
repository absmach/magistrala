// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package influxdb

import (
	"math"
	"strconv"
	"time"

	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/transformers/senml"
	"github.com/mainflux/mainflux/writers"

	influxdata "github.com/influxdata/influxdb/client/v2"
)

const pointName = "messages"

var errSaveMessage = errors.New("faled to save message to influxdb database")

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

func (repo *influxRepo) Save(messages ...senml.Message) error {
	pts, err := influxdata.NewBatchPoints(repo.cfg)
	if err != nil {
		return errors.Wrap(errSaveMessage, err)
	}

	for _, msg := range messages {
		tgs, flds := repo.tagsOf(&msg), repo.fieldsOf(&msg)

		sec, dec := math.Modf(msg.Time)
		t := time.Unix(int64(sec), int64(dec*(1e9)))

		pt, err := influxdata.NewPoint(pointName, tgs, flds, t)
		if err != nil {
			return errors.Wrap(errSaveMessage, err)
		}
		pts.AddPoint(pt)
	}
	if err := repo.client.Write(pts); err != nil {
		return errors.Wrap(errSaveMessage, err)
	}
	return nil
}

func (repo *influxRepo) tagsOf(msg *senml.Message) tags {
	return tags{
		"channel":   msg.Channel,
		"subtopic":  msg.Subtopic,
		"publisher": msg.Publisher,
		"name":      msg.Name,
	}
}

func (repo *influxRepo) fieldsOf(msg *senml.Message) fields {
	updateTime := strconv.FormatFloat(msg.UpdateTime, 'f', -1, 64)
	ret := fields{
		"protocol":   msg.Protocol,
		"unit":       msg.Unit,
		"updateTime": updateTime,
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
