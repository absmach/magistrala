// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package influxdb

import (
	"context"
	"math"
	"time"

	"github.com/mainflux/mainflux/consumers"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/transformers/json"
	"github.com/mainflux/mainflux/pkg/transformers/senml"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

const senmlPoints = "messages"

var errSaveMessage = errors.New("failed to save message to influxdb database")

var _ consumers.AsyncConsumer = (*influxRepo)(nil)
var _ consumers.BlockingConsumer = (*influxRepo)(nil)

type RepoConfig struct {
	Bucket string
	Org    string
}

type influxRepo struct {
	client           influxdb2.Client
	cfg              RepoConfig
	errCh            chan error
	writeAPI         api.WriteAPI
	writeAPIBlocking api.WriteAPIBlocking
}

// NewSync returns new InfluxDB writer.
func NewSync(client influxdb2.Client, config RepoConfig) consumers.BlockingConsumer {
	return &influxRepo{
		client:           client,
		cfg:              config,
		writeAPI:         nil,
		writeAPIBlocking: client.WriteAPIBlocking(config.Org, config.Bucket),
	}
}

func NewAsync(client influxdb2.Client, config RepoConfig) consumers.AsyncConsumer {
	return &influxRepo{
		client:           client,
		cfg:              config,
		errCh:            make(chan error, 1),
		writeAPI:         client.WriteAPI(config.Org, config.Bucket),
		writeAPIBlocking: nil,
	}
}

func (repo *influxRepo) ConsumeAsync(_ context.Context, message interface{}) {
	var err error
	var pts []*write.Point
	switch m := message.(type) {
	case json.Messages:
		pts, err = repo.jsonPoints(m)
	default:
		pts, err = repo.senmlPoints(m)
	}
	if err != nil {
		repo.errCh <- err
		return
	}

	done := make(chan bool)
	defer close(done)

	go func(done <-chan bool) {
		for {
			select {
			case err := <-repo.writeAPI.Errors():
				repo.errCh <- err
			case <-done:
				repo.errCh <- nil // pass nil error to the error channel
				return
			}
		}
	}(done)

	for _, pt := range pts {
		repo.writeAPI.WritePoint(pt)
	}

	repo.writeAPI.Flush()
}

func (repo *influxRepo) Errors() <-chan error {
	if repo.errCh != nil {
		return repo.errCh
	}

	return nil
}

func (repo *influxRepo) ConsumeBlocking(ctx context.Context, message interface{}) error {
	var err error
	var pts []*write.Point
	switch m := message.(type) {
	case json.Messages:
		pts, err = repo.jsonPoints(m)
	default:
		pts, err = repo.senmlPoints(m)
	}
	if err != nil {
		return err
	}

	return repo.writeAPIBlocking.WritePoint(ctx, pts...)
}

func (repo *influxRepo) senmlPoints(messages interface{}) ([]*write.Point, error) {
	msgs, ok := messages.([]senml.Message)
	if !ok {
		return nil, errSaveMessage
	}
	var pts []*write.Point
	for _, msg := range msgs {
		tgs, flds := senmlTags(msg), senmlFields(msg)

		sec, dec := math.Modf(msg.Time)
		t := time.Unix(int64(sec), int64(dec*(1e9)))

		pt := influxdb2.NewPoint(senmlPoints, tgs, flds, t)
		pts = append(pts, pt)
	}

	return pts, nil
}

func (repo *influxRepo) jsonPoints(msgs json.Messages) ([]*write.Point, error) {
	var pts []*write.Point
	for i, m := range msgs.Data {
		t := time.Unix(0, m.Created+int64(i))

		flat, err := json.Flatten(m.Payload)
		if err != nil {
			return nil, errors.Wrap(json.ErrTransform, err)
		}
		m.Payload = flat

		// Copy first-level fields so that the original Payload is unchanged.
		fields := make(map[string]interface{})
		for k, v := range m.Payload {
			fields[k] = v
		}
		// At least one known field need to exist so that COUNT can be performed.
		fields["protocol"] = m.Protocol
		pt := influxdb2.NewPoint(msgs.Format, jsonTags(m), fields, t)
		pts = append(pts, pt)
	}

	return pts, nil
}
