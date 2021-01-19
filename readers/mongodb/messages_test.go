// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mongodb_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	writer "github.com/mainflux/mainflux/consumers/writers/mongodb"
	"github.com/mainflux/mainflux/pkg/transformers/senml"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/readers"
	reader "github.com/mainflux/mainflux/readers/mongodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	testDB      = "test"
	collection  = "messages"
	subtopic    = "subtopic"
	msgsNum     = 100
	limit       = 10
	valueFields = 5
	mqttProt    = "mqtt"
	httpProt    = "http"
	msgName     = "temperature"
)

var (
	port string
	addr string

	v   float64 = 5
	vs          = "value"
	vb          = true
	vd          = "dataValue"
	sum float64 = 42
)

func TestReadSenml(t *testing.T) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	require.Nil(t, err, fmt.Sprintf("Creating new MongoDB client expected to succeed: %s.\n", err))

	db := client.Database(testDB)
	writer := writer.New(db)

	chanID, err := uuid.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	pubID, err := uuid.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	pubID2, err := uuid.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	m := senml.Message{
		Channel:   chanID,
		Publisher: pubID,
		Protocol:  mqttProt,
	}

	messages := []senml.Message{}
	valueMsgs := []senml.Message{}
	boolMsgs := []senml.Message{}
	stringMsgs := []senml.Message{}
	dataMsgs := []senml.Message{}
	queryMsgs := []senml.Message{}
	now := time.Now().Unix()

	for i := 0; i < msgsNum; i++ {
		// Mix possible values as well as value sum.
		msg := m
		msg.Time = float64(now - int64(i))

		count := i % valueFields
		switch count {
		case 0:
			msg.Value = &v
			valueMsgs = append(valueMsgs, msg)
		case 1:
			msg.BoolValue = &vb
			boolMsgs = append(boolMsgs, msg)
		case 2:
			msg.StringValue = &vs
			stringMsgs = append(stringMsgs, msg)
		case 3:
			msg.DataValue = &vd
			dataMsgs = append(dataMsgs, msg)
		case 4:
			msg.Sum = &sum
			msg.Subtopic = subtopic
			msg.Protocol = httpProt
			msg.Publisher = pubID2
			msg.Name = msgName
			queryMsgs = append(queryMsgs, msg)
		}
		messages = append(messages, msg)
	}
	err = writer.Consume(messages)
	require.Nil(t, err, fmt.Sprintf("failed to store message to MongoDB: %s", err))
	reader := reader.New(db)

	cases := map[string]struct {
		chanID string
		query  map[string]string
		page   readers.MessagesPage
	}{
		"read message page for existing channel": {
			chanID: chanID,
			page: readers.MessagesPage{
				Total:    msgsNum,
				Offset:   0,
				Limit:    11,
				Messages: fromSenml(messages[0:11]),
			},
		},
		"read message page for non-existent channel": {
			chanID: "2",
			page: readers.MessagesPage{
				Total:    0,
				Offset:   0,
				Limit:    10,
				Messages: []readers.Message{},
			},
		},
		"read message last page": {
			chanID: chanID,
			page: readers.MessagesPage{
				Total:    msgsNum,
				Offset:   95,
				Limit:    limit,
				Messages: fromSenml(messages[95:msgsNum]),
			},
		},
		"read message with non-existent subtopic": {
			chanID: chanID,
			query:  map[string]string{"subtopic": "not-present"},
			page: readers.MessagesPage{
				Total:    0,
				Offset:   0,
				Limit:    msgsNum,
				Messages: []readers.Message{},
			},
		},
		"read message with subtopic": {
			chanID: chanID,
			query:  map[string]string{"subtopic": subtopic},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Offset:   0,
				Limit:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read message with publisher": {
			chanID: chanID,
			query:  map[string]string{"publisher": pubID2},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Offset:   0,
				Limit:    limit,
				Messages: fromSenml(queryMsgs[0:limit]),
			},
		},
		"read message with protocol": {
			chanID: chanID,
			query:  map[string]string{"protocol": httpProt},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Offset:   0,
				Limit:    limit,
				Messages: fromSenml(queryMsgs[0:limit]),
			},
		},
		"read message with name": {
			chanID: chanID,
			query:  map[string]string{"name": msgName},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Offset:   0,
				Limit:    limit,
				Messages: fromSenml(queryMsgs[0:limit]),
			},
		},
		"read message with value": {
			chanID: chanID,
			query:  map[string]string{"v": fmt.Sprintf("%f", v)},
			page: readers.MessagesPage{
				Total:    uint64(len(valueMsgs)),
				Offset:   0,
				Limit:    limit,
				Messages: fromSenml(valueMsgs[0:limit]),
			},
		},
		"read message with boolean value": {
			chanID: chanID,
			query:  map[string]string{"vb": fmt.Sprintf("%t", vb)},
			page: readers.MessagesPage{
				Total:    uint64(len(boolMsgs)),
				Offset:   0,
				Limit:    limit,
				Messages: fromSenml(boolMsgs[0:limit]),
			},
		},
		"read message with string value": {
			chanID: chanID,
			query:  map[string]string{"vs": vs},
			page: readers.MessagesPage{
				Total:    uint64(len(stringMsgs)),
				Offset:   0,
				Limit:    limit,
				Messages: fromSenml(stringMsgs[0:limit]),
			},
		},
		"read message with data value": {
			chanID: chanID,
			query:  map[string]string{"vd": vd},
			page: readers.MessagesPage{
				Total:    uint64(len(dataMsgs)),
				Offset:   0,
				Limit:    limit,
				Messages: fromSenml(dataMsgs[0:limit]),
			},
		},
		"read message with from": {
			chanID: chanID,
			query: map[string]string{
				"from": fmt.Sprintf("%f", messages[20].Time),
			},
			page: readers.MessagesPage{
				Total:    uint64(len(messages[0:21])),
				Offset:   0,
				Limit:    uint64(len(messages[0:21])),
				Messages: fromSenml(messages[0:21]),
			},
		},
		"read message with to": {
			chanID: chanID,
			query: map[string]string{
				"to": fmt.Sprintf("%f", messages[20].Time),
			},
			page: readers.MessagesPage{
				Total:    uint64(len(messages[21:])),
				Offset:   0,
				Limit:    uint64(len(messages[21:])),
				Messages: fromSenml(messages[21:]),
			},
		},
		"read message with from/to": {
			chanID: chanID,
			query: map[string]string{
				"from": fmt.Sprintf("%f", messages[5].Time),
				"to":   fmt.Sprintf("%f", messages[0].Time),
			},
			page: readers.MessagesPage{
				Total:    5,
				Offset:   0,
				Limit:    limit,
				Messages: fromSenml(messages[1:6]),
			},
		},
	}

	for desc, tc := range cases {
		result, err := reader.ReadAll(tc.chanID, tc.page.Offset, tc.page.Limit, tc.query)

		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %s", desc, err))
		assert.ElementsMatch(t, tc.page.Messages, result.Messages, fmt.Sprintf("%s: expected %v got %v", desc, tc.page.Messages, result.Messages))
		assert.Equal(t, tc.page.Total, result.Total, fmt.Sprintf("%s: expected %v got %v", desc, tc.page.Total, result.Total))
	}
}

func fromSenml(in []senml.Message) []readers.Message {
	var ret []readers.Message
	for _, m := range in {
		ret = append(ret, m)
	}
	return ret
}
