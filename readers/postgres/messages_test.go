// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"fmt"
	"testing"
	"time"

	pwriter "github.com/mainflux/mainflux/consumers/writers/postgres"
	"github.com/mainflux/mainflux/pkg/transformers/senml"
	uuidProvider "github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/readers"
	preader "github.com/mainflux/mainflux/readers/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	subtopic    = "subtopic"
	msgsNum     = 100
	limit       = 10
	valueFields = 5
	mqttProt    = "mqtt"
	httpProt    = "http"
	msgName     = "temperature"
)

var (
	v   float64 = 5
	vs          = "value"
	vb          = true
	vd          = "dataValue"
	sum float64 = 42
)

func TestReadSenml(t *testing.T) {
	writer := pwriter.New(db)

	chanID, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	pubID, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	pub2ID, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	wrongID, err := uuidProvider.New().ID()
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
			msg.Publisher = pub2ID
			msg.Name = msgName
			queryMsgs = append(queryMsgs, msg)
		}

		messages = append(messages, msg)
	}

	err = writer.Consume(messages)
	assert.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))

	reader := preader.New(db)

	// Since messages are not saved in natural order,
	// cases that return subset of messages are only
	// checking data result set size, but not content.
	cases := map[string]struct {
		chanID string
		offset uint64
		limit  uint64
		query  map[string]string
		page   readers.MessagesPage
	}{
		"read message page for existing channel": {
			chanID: chanID,
			offset: 0,
			limit:  msgsNum,
			page: readers.MessagesPage{
				Total:    msgsNum,
				Offset:   0,
				Limit:    msgsNum,
				Messages: fromSenml(messages),
			},
		},
		"read message page for non-existent channel": {
			chanID: wrongID,
			offset: 0,
			limit:  msgsNum,
			page: readers.MessagesPage{
				Total:    0,
				Offset:   0,
				Limit:    msgsNum,
				Messages: []readers.Message{},
			},
		},
		"read message last page": {
			chanID: chanID,
			offset: msgsNum - 20,
			limit:  msgsNum,
			page: readers.MessagesPage{
				Total:    msgsNum,
				Offset:   msgsNum - 20,
				Limit:    msgsNum,
				Messages: fromSenml(messages[msgsNum-20 : msgsNum]),
			},
		},
		"read message with non-existent subtopic": {
			chanID: chanID,
			offset: 0,
			limit:  msgsNum,
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
			offset: 0,
			limit:  uint64(len(queryMsgs)),
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
			offset: 0,
			limit:  uint64(len(queryMsgs)),
			query:  map[string]string{"publisher": pub2ID},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Offset:   0,
				Limit:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read message with protocol": {
			chanID: chanID,
			offset: 0,
			limit:  uint64(len(queryMsgs)),
			query:  map[string]string{"protocol": httpProt},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Offset:   0,
				Limit:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read message with name": {
			chanID: chanID,
			offset: 0,
			limit:  limit,
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
			offset: 0,
			limit:  limit,
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
			offset: 0,
			limit:  limit,
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
			offset: 0,
			limit:  limit,
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
			offset: 0,
			limit:  limit,
			query:  map[string]string{"vd": vd},
			page: readers.MessagesPage{
				Total:    uint64(len(dataMsgs)),
				Offset:   0,
				Limit:    limit,
				Messages: fromSenml(dataMsgs[0:limit]),
			},
		},
		"read message with from/to": {
			chanID: chanID,
			offset: 0,
			limit:  limit,
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
		result, err := reader.ReadAll(tc.chanID, tc.offset, tc.limit, tc.query)
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
