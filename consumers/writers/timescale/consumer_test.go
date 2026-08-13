// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package timescale_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/absmach/magistrala/consumers/writers/timescale"
	"github.com/absmach/magistrala/pkg/transformers/json"
	"github.com/absmach/magistrala/pkg/transformers/senml"
	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		Payload: map[string]any{
			"field_1": 123,
			"field_2": "value",
			"field_3": false,
			"field_4": 12.344,
			"field_5": map[string]any{
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

// JSON tables are created on demand and so sit outside the migration set: one
// created before device_id existed is missing the column, and the writer has to
// catch it up rather than fail every insert from then on.
func TestSaveJSONCatchesUpLegacyTable(t *testing.T) {
	repo := timescale.New(db)

	const format = "legacy_json"
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS ` + format + ` (
            created       BIGINT NOT NULL,
            channel       VARCHAR(254),
            subtopic      VARCHAR(254),
            publisher     VARCHAR(254),
            protocol      TEXT,
            payload       JSONB,
            PRIMARY KEY (created, publisher, subtopic)
        );`)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	chid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	pubid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	msgs := json.Messages{
		Format: format,
		Data: []json.Message{{
			Channel:   chid.String(),
			Publisher: pubid.String(),
			Created:   time.Now().UnixNano(),
			Protocol:  "mqtt",
			DeviceId:  "Meter.A-01:X",
			Payload:   map[string]any{"field": 1},
		}},
	}

	err = repo.ConsumeBlocking(context.TODO(), msgs)
	assert.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))

	var stored string
	err = db.Get(&stored, `SELECT device_id FROM `+format+` WHERE channel = $1`, chid.String())
	assert.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))
	assert.Equal(t, "Meter.A-01:X", stored)
}
