package influxdb_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	influxdata "github.com/influxdata/influxdb/client/v2"
	"github.com/mainflux/mainflux"
	reader "github.com/mainflux/mainflux/readers/influxdb"
	writer "github.com/mainflux/mainflux/writers/influxdb"

	log "github.com/mainflux/mainflux/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testDB      = "test"
	chanID      = "1"
	msgsNum     = 101
	valueFields = 6
)

var (
	port       string
	client     influxdata.Client
	testLog, _ = log.New(os.Stdout, log.Info.String())

	clientCfg = influxdata.HTTPConfig{
		Username: "test",
		Password: "test",
	}

	msg = mainflux.Message{
		Channel:    chanID,
		Publisher:  "1",
		Protocol:   "mqtt",
		Name:       "name",
		Unit:       "U",
		Value:      &mainflux.Message_FloatValue{FloatValue: 5},
		ValueSum:   &mainflux.SumValue{Value: 45},
		Time:       123456,
		UpdateTime: 1234,
		Link:       "link",
	}
)

func TestReadAll(t *testing.T) {
	writer, err := writer.New(client, testDB, 1, time.Second)
	require.Nil(t, err, fmt.Sprintf("Creating new InfluxDB writer expected to succeed: %s.\n", err))

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
		require.Nil(t, err, fmt.Sprintf("failed to store message to InfluxDB: %s", err))
		messages = append(messages, msg)
	}

	reader, err := reader.New(client, testDB)
	require.Nil(t, err, fmt.Sprintf("Creating new InfluxDB reader expected to succeed: %s.\n", err))

	cases := map[string]struct {
		chanID   string
		offset   uint64
		limit    uint64
		messages []mainflux.Message
	}{
		"read message page for existing channel": {
			chanID:   chanID,
			offset:   0,
			limit:    10,
			messages: messages[0:10],
		},
		"read message page for too large limit": {
			chanID:   chanID,
			offset:   0,
			limit:    101,
			messages: messages[0:100],
		},
		"read message page for non-existent channel": {
			chanID:   "2",
			offset:   0,
			limit:    10,
			messages: []mainflux.Message{},
		},
		"read message last page": {
			chanID:   chanID,
			offset:   95,
			limit:    10,
			messages: messages[95:101],
		},
	}

	for desc, tc := range cases {
		result := reader.ReadAll(tc.chanID, tc.offset, tc.limit)
		assert.ElementsMatch(t, tc.messages, result, fmt.Sprintf("%s: expected: %v \n-------------\n got: %v", desc, tc.messages, result))
	}
}
