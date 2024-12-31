// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package timescale_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/absmach/supermq/consumers/writers/timescale"
	"github.com/absmach/supermq/pkg/transformers/json"
	"github.com/absmach/supermq/pkg/transformers/senml"
	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
)

const (
	msgsNum     = 42
	valueFields = 5
	subtopic    = "topic"
)

var (
	v       float64 = 5
	stringV         = "value"
	boolV           = true
	dataV           = "base64"
	sum     float64 = 42
)

func TestSaveSenml(t *testing.T) {
	repo := timescale.New(db)

	chid, err := uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	msg := senml.Message{}
	msg.Channel = chid.String()

	pubid, err := uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	msg.Publisher = pubid.String()

	now := time.Now().Unix()
	var msgs []senml.Message

	for i := 0; i < msgsNum; i++ {
		// Mix possible values as well as value sum.
		count := i % valueFields
		switch count {
		case 0:
			msg.Subtopic = subtopic
			msg.Value = &v
		case 1:
			msg.BoolValue = &boolV
		case 2:
			msg.StringValue = &stringV
		case 3:
			msg.DataValue = &dataV
		case 4:
			msg.Sum = &sum
		}

		msg.Time = float64(now + int64(i))
		msgs = append(msgs, msg)
	}

	err = repo.ConsumeBlocking(context.TODO(), msgs)
	assert.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))
}

func TestSaveJSON(t *testing.T) {
	repo := timescale.New(db)

	chid, err := uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	pubid, err := uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	msg := json.Message{
		Channel:   chid.String(),
		Publisher: pubid.String(),
		Created:   time.Now().Unix(),
		Subtopic:  "subtopic/format/some_json",
		Protocol:  "mqtt",
		Payload: map[string]interface{}{
			"field_1": 123,
			"field_2": "value",
			"field_3": false,
			"field_4": 12.344,
			"field_5": map[string]interface{}{
				"field_1": "value",
				"field_2": 42,
			},
		},
	}

	now := time.Now().Unix()
	msgs := json.Messages{
		Format: "some_json",
	}

	for i := 0; i < msgsNum; i++ {
		msg.Created = now + int64(i)
		msgs.Data = append(msgs.Data, msg)
	}

	err = repo.ConsumeBlocking(context.TODO(), msgs)
	assert.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))
}
