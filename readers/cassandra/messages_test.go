// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cassandra_test

import (
	"fmt"
	"testing"
	"time"

	writer "github.com/mainflux/mainflux/consumers/writers/cassandra"
	"github.com/mainflux/mainflux/pkg/transformers/senml"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/readers"
	reader "github.com/mainflux/mainflux/readers/cassandra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	keyspace    = "mainflux"
	subtopic    = "subtopic"
	msgsNum     = 100
	limit       = 10
	valueFields = 5
	mqttProt    = "mqtt"
	httpProt    = "http"
	msgName     = "temperature"
)

var (
	addr = "localhost"

	v   float64 = 5
	vs          = "value"
	vb          = true
	vd          = "base64"
	sum float64 = 42

	idProvider = uuid.New()
)

func TestReadSenml(t *testing.T) {
	session, err := reader.Connect(reader.DBConfig{
		Hosts:    []string{addr},
		Keyspace: keyspace,
	})
	require.Nil(t, err, fmt.Sprintf("failed to connect to Cassandra: %s", err))
	defer session.Close()
	writer := writer.New(session)

	chanID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	pubID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	pubID2, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	wrongID, err := idProvider.ID()
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
	now := float64(time.Now().Unix())

	for i := 0; i < msgsNum; i++ {
		// Mix possible values as well as value sum.
		msg := m
		msg.Time = now - float64(i)

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
	require.Nil(t, err, fmt.Sprintf("failed to store message to Cassandra: %s", err))

	reader := reader.New(session)

	// Since messages are not saved in natural order,
	// cases that return subset of messages are only
	// checking data result set size, but not content.
	cases := map[string]struct {
		chanID   string
		pageMeta readers.PageMetadata
		page     readers.MessagesPage
	}{
		"read message page for existing channel": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset: 0,
				Limit:  msgsNum,
			},
			page: readers.MessagesPage{
				Total:    msgsNum,
				Messages: fromSenml(messages),
			},
		},
		"read message page for non-existent channel": {
			chanID: wrongID,
			pageMeta: readers.PageMetadata{
				Offset: 0,
				Limit:  msgsNum,
			},
			page: readers.MessagesPage{
				Messages: []readers.Message{},
			},
		},
		"read message last page": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset: msgsNum - 20,
				Limit:  msgsNum,
			},
			page: readers.MessagesPage{
				Total:    msgsNum,
				Messages: fromSenml(messages[msgsNum-20 : msgsNum]),
			},
		},
		"read message with non-existent subtopic": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:   0,
				Limit:    msgsNum,
				Subtopic: "not-present",
			},
			page: readers.MessagesPage{
				Messages: []readers.Message{},
			},
		},
		"read message with subtopic": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:   0,
				Limit:    uint64(len(queryMsgs)),
				Subtopic: subtopic,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read message with publisher": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:    0,
				Limit:     uint64(len(queryMsgs)),
				Publisher: pubID2,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read message with protocol": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:   0,
				Limit:    uint64(len(queryMsgs)),
				Protocol: httpProt,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read message with name": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset: 0,
				Limit:  limit,
				Name:   msgName,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs[0:limit]),
			},
		},
		"read message with value": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset: 0,
				Limit:  limit,
				Value:  v,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(valueMsgs[0:limit]),
			},
		},
		"read message with boolean value": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:    0,
				Limit:     limit,
				BoolValue: vb,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(boolMsgs)),
				Messages: fromSenml(boolMsgs[0:limit]),
			},
		},
		"read message with string value": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:      0,
				Limit:       limit,
				StringValue: vs,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(stringMsgs)),
				Messages: fromSenml(stringMsgs[0:limit]),
			},
		},
		"read message with data value": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:    0,
				Limit:     limit,
				DataValue: vd,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(dataMsgs)),
				Messages: fromSenml(dataMsgs[0:limit]),
			},
		},
		"read message with from": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset: 0,
				Limit:  uint64(len(messages[0:21])),
				From:   messages[20].Time,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(messages[0:21])),
				Messages: fromSenml(messages[0:21]),
			},
		},
		"read message with to": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset: 0,
				Limit:  uint64(len(messages[21:])),
				To:     messages[20].Time,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(messages[21:])),
				Messages: fromSenml(messages[21:]),
			},
		},
		"read message with from/to": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset: 0,
				Limit:  limit,
				From:   messages[5].Time,
				To:     messages[0].Time,
			},
			page: readers.MessagesPage{
				Total:    5,
				Messages: fromSenml(messages[1:6]),
			},
		},
	}

	for desc, tc := range cases {
		result, err := reader.ReadAll(tc.chanID, tc.pageMeta)
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
