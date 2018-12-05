//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package cassandra_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/mainflux/mainflux"
	readers "github.com/mainflux/mainflux/readers/cassandra"
	writers "github.com/mainflux/mainflux/writers/cassandra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	keyspace    = "mainflux"
	chanID      = "1"
	msgsNum     = 42
	valueFields = 6
)

var (
	addr = "localhost"
	msg  = mainflux.Message{
		Channel:   chanID,
		Publisher: "1",
		Protocol:  "mqtt",
	}
)

func TestReadAll(t *testing.T) {
	session, err := readers.Connect([]string{addr}, keyspace)
	require.Nil(t, err, fmt.Sprintf("failed to connect to Cassandra: %s", err))
	defer session.Close()
	writer := writers.New(session)

	messages := []mainflux.Message{}
	now := time.Now().Unix()
	for i := 0; i < msgsNum; i++ {
		// Mix possible values as well as value sum.
		count := i % valueFields
		switch count {
		case 0:
			msg.Value = &mainflux.Message_FloatValue{FloatValue: 5}
		case 1:
			msg.Value = &mainflux.Message_BoolValue{BoolValue: false}
		case 2:
			msg.Value = &mainflux.Message_StringValue{StringValue: "value"}
		case 3:
			msg.Value = &mainflux.Message_DataValue{DataValue: "base64data"}
		case 4:
			msg.ValueSum = nil
		case 5:
			msg.ValueSum = &mainflux.SumValue{Value: 45}
		}
		msg.Time = float64(now + int64(i))

		err := writer.Save(msg)
		require.Nil(t, err, fmt.Sprintf("failed to store message to Cassandra: %s", err))
		messages = append(messages, msg)
	}

	reader := readers.New(session)

	// Since messages are not saved in natural order,
	// cases that return subset of messages are only
	// checking data result set size, but not content.
	cases := map[string]struct {
		chanID   string
		offset   uint64
		limit    uint64
		messages []mainflux.Message
	}{
		"read message page for existing channel": {
			chanID:   chanID,
			offset:   0,
			limit:    msgsNum,
			messages: messages,
		},
		"read message page for non-existent channel": {
			chanID:   "2",
			offset:   0,
			limit:    msgsNum,
			messages: []mainflux.Message{},
		},
		"read message last page": {
			chanID:   chanID,
			offset:   40,
			limit:    5,
			messages: messages[40:42],
		},
	}

	for desc, tc := range cases {
		result := reader.ReadAll(tc.chanID, tc.offset, tc.limit)
		if tc.offset > 0 {
			assert.Equal(t, len(tc.messages), len(result), fmt.Sprintf("%s: expected %d messages, got %d", desc, len(tc.messages), len(result)))
			continue
		}
		assert.ElementsMatch(t, tc.messages, result, fmt.Sprintf("%s: expected %v got %v", desc, tc.messages, result))
	}
}
