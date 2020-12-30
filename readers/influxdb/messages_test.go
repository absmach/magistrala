package influxdb_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	influxdata "github.com/influxdata/influxdb/client/v2"
	"github.com/mainflux/mainflux/pkg/transformers/senml"
	"github.com/mainflux/mainflux/readers"
	reader "github.com/mainflux/mainflux/readers/influxdb"
	writer "github.com/mainflux/mainflux/writers/influxdb"

	log "github.com/mainflux/mainflux/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testDB     = "test"
	chanID     = "1"
	subtopic   = "topic"
	msgsNum    = 100
	msgsValNum = 20
	limit      = 10
)

var (
	v   float64 = 5
	vs          = "value"
	vb          = true
	vd          = "dataValue"
	sum float64 = 42
)

var (
	valueFields = 5
	port        string
	client      influxdata.Client
	testLog, _  = log.New(os.Stdout, log.Info.String())

	clientCfg = influxdata.HTTPConfig{
		Username: "test",
		Password: "test",
	}

	m = senml.Message{
		Channel:    chanID,
		Publisher:  "1",
		Protocol:   "mqtt",
		Name:       "name",
		Unit:       "U",
		Time:       123456,
		UpdateTime: 1234,
	}
)

func TestReadAll(t *testing.T) {
	writer := writer.New(client, testDB)

	messages := []senml.Message{}
	valSubtopicMsgs := []senml.Message{}
	boolMsgs := []senml.Message{}
	stringMsgs := []senml.Message{}
	dataMsgs := []senml.Message{}
	now := float64(time.Now().UTC().Second())

	for i := 0; i < msgsNum; i++ {
		// Mix possible values as well as value sum.
		msg := m
		msg.Time = now - float64(i)

		count := i % valueFields
		switch count {
		case 0:
			msg.Subtopic = subtopic
			msg.Value = &v
			valSubtopicMsgs = append(valSubtopicMsgs, msg)
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
		}

		messages = append(messages, msg)
	}

	err := writer.Save(messages)
	require.Nil(t, err, fmt.Sprintf("failed to store message to InfluxDB: %s", err))

	reader := reader.New(client, testDB)
	require.Nil(t, err, fmt.Sprintf("Creating new InfluxDB reader expected to succeed: %s.\n", err))

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
			limit:  limit,
			page: readers.MessagesPage{
				Total:    msgsNum,
				Offset:   0,
				Limit:    limit,
				Messages: fromSenml(messages[0:limit]),
			},
		},
		"read message page for non-existent channel": {
			chanID: "2",
			offset: 0,
			limit:  limit,
			page: readers.MessagesPage{
				Total:    0,
				Offset:   0,
				Limit:    limit,
				Messages: []readers.Message{},
			},
		},
		"read message last page": {
			chanID: chanID,
			offset: 95,
			limit:  limit,
			page: readers.MessagesPage{
				Total:    msgsNum,
				Offset:   95,
				Limit:    limit,
				Messages: fromSenml(messages[95:msgsNum]),
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
			limit:  limit,
			query:  map[string]string{"subtopic": subtopic},
			page: readers.MessagesPage{
				Total:    uint64(len(valSubtopicMsgs)),
				Offset:   0,
				Limit:    limit,
				Messages: fromSenml(valSubtopicMsgs[0:limit]),
			},
		},
		"read message with value": {
			chanID: chanID,
			offset: 0,
			limit:  limit,
			query:  map[string]string{"v": fmt.Sprintf("%f", v)},
			page: readers.MessagesPage{
				Total:    msgsValNum,
				Offset:   0,
				Limit:    limit,
				Messages: fromSenml(valSubtopicMsgs[0:limit]),
			},
		},
		"read message with boolean value": {
			chanID: chanID,
			offset: 0,
			limit:  limit,
			query:  map[string]string{"vb": fmt.Sprintf("%t", vb)},
			page: readers.MessagesPage{
				Total:    msgsValNum,
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
				Total:    msgsValNum,
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
				Total:    msgsValNum,
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
		assert.ElementsMatch(t, tc.page.Messages, result.Messages, fmt.Sprintf("%s: expected: %v, got: %v", desc, tc.page.Messages, result.Messages))

		assert.Equal(t, tc.page.Total, result.Total, fmt.Sprintf("%s: expected %d got %d", desc, tc.page.Total, result.Total))
	}
}

func fromSenml(in []senml.Message) []readers.Message {
	var ret []readers.Message
	for _, m := range in {
		ret = append(ret, m)
	}
	return ret
}
