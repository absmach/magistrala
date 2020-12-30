// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cassandra_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/mainflux/mainflux/pkg/transformers/senml"
	"github.com/mainflux/mainflux/writers/cassandra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	keyspace    = "mainflux"
	msgsNum     = 42
	valueFields = 5
	subtopic    = "topic"
)

var addr = "localhost"

var (
	v       float64 = 5
	stringV         = "value"
	boolV           = true
	dataV           = "base64"
	sum     float64 = 42
)

func TestSave(t *testing.T) {
	session, err := cassandra.Connect(cassandra.DBConfig{
		Hosts:    []string{addr},
		Keyspace: keyspace,
	})
	require.Nil(t, err, fmt.Sprintf("failed to connect to Cassandra: %s", err))

	repo := cassandra.New(session)
	now := time.Now().Unix()
	msg := senml.Message{
		Channel:   "1",
		Publisher: "1",
		Protocol:  "mqtt",
	}
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

	err = repo.Save(msgs)
	assert.Nil(t, err, fmt.Sprintf("expected no error, got %s", err))
}
