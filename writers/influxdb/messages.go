//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package influxdb

import (
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/mainflux/mainflux/writers"

	influxdata "github.com/influxdata/influxdb/client/v2"
	"github.com/mainflux/mainflux"
)

const pointName = "messages"

var _ writers.MessageRepository = (*influxRepo)(nil)

var (
	errZeroValueSize    = errors.New("zero value batch size")
	errZeroValueTimeout = errors.New("zero value batch timeout")
	errNilBatch         = errors.New("nil batch")
)

type influxRepo struct {
	client    influxdata.Client
	batch     influxdata.BatchPoints
	batchSize int
	mu        sync.Mutex
	tick      <-chan time.Time
	cfg       influxdata.BatchPointsConfig
}

type fields map[string]interface{}
type tags map[string]string

// New returns new InfluxDB writer.
func New(client influxdata.Client, database string, batchSize int, batchTimeout time.Duration) (writers.MessageRepository, error) {
	if batchSize <= 0 {
		return &influxRepo{}, errZeroValueSize
	}

	if batchTimeout <= 0 {
		return &influxRepo{}, errZeroValueTimeout
	}

	repo := &influxRepo{
		client: client,
		cfg: influxdata.BatchPointsConfig{
			Database: database,
		},
		batchSize: batchSize,
	}

	var err error
	repo.batch, err = influxdata.NewBatchPoints(repo.cfg)
	if err != nil {
		return &influxRepo{}, err
	}

	repo.tick = time.NewTicker(batchTimeout).C
	go func() {
		for {
			<-repo.tick
			// Nil point indicates that savePoint method is triggered by the ticker.
			repo.savePoint(nil)
		}
	}()

	return repo, nil
}

func (repo *influxRepo) savePoint(point *influxdata.Point) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if repo.batch == nil {
		return errNilBatch
	}

	// Ignore ticker if there is nothing to save.
	if len(repo.batch.Points()) == 0 && point == nil {
		return nil
	}

	if point != nil {
		repo.batch.AddPoint(point)
	}

	if len(repo.batch.Points())%repo.batchSize == 0 || point == nil {
		if err := repo.client.Write(repo.batch); err != nil {
			return err
		}
		// It would be nice to reset ticker at this point, which
		// implies creating a new ticker and goroutine. It would
		// introduce unnecessary complexity with no justified benefits.
		var err error
		repo.batch, err = influxdata.NewBatchPoints(repo.cfg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (repo *influxRepo) Save(msg mainflux.Message) error {
	tgs, flds := repo.tagsOf(&msg), repo.fieldsOf(&msg)
	t := time.Unix(int64(msg.Time), 0)

	pt, err := influxdata.NewPoint(pointName, tgs, flds, t)
	if err != nil {
		return err
	}

	return repo.savePoint(pt)
}

func (repo *influxRepo) tagsOf(msg *mainflux.Message) tags {
	return tags{
		"channel":   msg.Channel,
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
