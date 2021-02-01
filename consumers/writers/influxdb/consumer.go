// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package influxdb

import (
	"math"
	"time"

	"github.com/mainflux/mainflux/consumers"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/transformers/json"
	"github.com/mainflux/mainflux/pkg/transformers/senml"

	influxdata "github.com/influxdata/influxdb/client/v2"
)

const senmlPoints = "messages"

var errSaveMessage = errors.New("failed to save message to influxdb database")

var _ consumers.Consumer = (*influxRepo)(nil)

type influxRepo struct {
	client influxdata.Client
	cfg    influxdata.BatchPointsConfig
}

// New returns new InfluxDB writer.
func New(client influxdata.Client, database string) consumers.Consumer {
	return &influxRepo{
		client: client,
		cfg: influxdata.BatchPointsConfig{
			Database: database,
		},
	}
}

func (repo *influxRepo) Consume(message interface{}) error {
	pts, err := influxdata.NewBatchPoints(repo.cfg)
	if err != nil {
		return errors.Wrap(errSaveMessage, err)
	}
	switch m := message.(type) {
	case json.Messages:
		pts, err = repo.jsonPoints(pts, m)
	default:
		pts, err = repo.senmlPoints(pts, m)
	}
	if err != nil {
		return err
	}

	if err := repo.client.Write(pts); err != nil {
		return errors.Wrap(errSaveMessage, err)
	}
	return nil
}

func (repo *influxRepo) senmlPoints(pts influxdata.BatchPoints, messages interface{}) (influxdata.BatchPoints, error) {
	msgs, ok := messages.([]senml.Message)
	if !ok {
		return nil, errSaveMessage
	}

	for _, msg := range msgs {
		tgs, flds := senmlTags(msg), senmlFields(msg)

		sec, dec := math.Modf(msg.Time)
		t := time.Unix(int64(sec), int64(dec*(1e9)))

		pt, err := influxdata.NewPoint(senmlPoints, tgs, flds, t)
		if err != nil {
			return nil, errors.Wrap(errSaveMessage, err)
		}
		pts.AddPoint(pt)
	}

	return pts, nil
}

func (repo *influxRepo) jsonPoints(pts influxdata.BatchPoints, msgs json.Messages) (influxdata.BatchPoints, error) {
	for i, m := range msgs.Data {
		t := time.Unix(0, m.Created+int64(i))

		// Copy first-level fields so that the original Payload is unchanged.
		fields := make(map[string]interface{})
		for k, v := range m.Payload {
			fields[k] = v
		}
		// At least one known field need to exist so that COUNT can be performed.
		fields["protocol"] = m.Protocol
		pt, err := influxdata.NewPoint(msgs.Format, jsonTags(m), fields, t)
		if err != nil {
			return nil, errors.Wrap(errSaveMessage, err)
		}
		pts.AddPoint(pt)
	}

	return pts, nil
}
