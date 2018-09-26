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
	testDB        = "test"
	chanID        = 1
	numOfMessages = 101
)

var (
	port      string
	client    influxdata.Client
	clientCfg = influxdata.HTTPConfig{
		Username: "test",
		Password: "test",
	}
	msg = mainflux.Message{
		Channel:   chanID,
		Publisher: 1,
		Protocol:  "mqtt",
	}
	testLog, _ = log.New(os.Stdout, log.Info.String())
)

func TestReadAll(t *testing.T) {
	client, err := influxdata.NewHTTPClient(clientCfg)
	require.Nil(t, err, fmt.Sprintf("Creating new InfluxDB client expected to succeed: %s.\n", err))

	writer, err := writer.New(client, testDB, 1, time.Second)
	require.Nil(t, err, fmt.Sprintf("Creating new InfluxDB writer expected to succeed: %s.\n", err))

	messages := []mainflux.Message{}
	for i := 0; i < numOfMessages; i++ {
		err := writer.Save(msg)
		require.Nil(t, err, fmt.Sprintf("failed to store message to InfluxDB: %s", err))
		messages = append(messages, msg)
	}

	reader, err := reader.New(client, testDB)
	require.Nil(t, err, fmt.Sprintf("Creating new InfluxDB reader expected to succeed: %s.\n", err))

	cases := map[string]struct {
		chanID   uint64
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
			chanID:   2,
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
		assert.ElementsMatch(t, tc.messages, result, fmt.Sprintf("%s: expected %v got %v", desc, tc.messages, result))
	}
}
