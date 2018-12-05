//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package influxdb_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	influxdata "github.com/influxdata/influxdb/client/v2"
	"github.com/mainflux/mainflux"
	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/writers"
	writer "github.com/mainflux/mainflux/writers/influxdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const valueFields = 6

var (
	port          string
	testLog, _    = log.New(os.Stdout, log.Info.String())
	testDB        = "test"
	saveTimeout   = 2 * time.Second
	saveBatchSize = 20
	streamsSize   = 250
	selectMsgs    = fmt.Sprintf("SELECT * FROM test..messages")
	dropMsgs      = fmt.Sprintf("DROP SERIES FROM messages")
	client        influxdata.Client
	clientCfg     = influxdata.HTTPConfig{
		Username: "test",
		Password: "test",
	}

	msg = mainflux.Message{
		Channel:    "45",
		Publisher:  "2580",
		Protocol:   "http",
		Name:       "test name",
		Unit:       "km",
		Value:      &mainflux.Message_FloatValue{FloatValue: 24},
		ValueSum:   &mainflux.SumValue{Value: 22},
		UpdateTime: 5456565466,
		Link:       "link",
	}
)

// This is utility function to query the database.
func queryDB(cmd string) ([][]interface{}, error) {
	q := influxdata.Query{
		Command:  cmd,
		Database: testDB,
	}
	response, err := client.Query(q)
	if err != nil {
		return nil, err
	}
	if response.Error() != nil {
		return nil, response.Error()
	}
	if len(response.Results[0].Series) == 0 {
		return nil, nil
	}
	// There is only one query, so only one result and
	// all data are stored in the same series.
	return response.Results[0].Series[0].Values, nil
}

func TestNew(t *testing.T) {
	cases := []struct {
		desc         string
		batchSize    int
		err          error
		batchTimeout time.Duration
		errText      string
	}{
		{
			desc:         "Create writer with zero value batch size",
			batchSize:    0,
			batchTimeout: time.Duration(5 * time.Second),
			errText:      "zero value batch size",
		},
		{
			desc:         "Create writer with zero value batch timeout",
			batchSize:    5,
			batchTimeout: time.Duration(0 * time.Second),
			errText:      "zero value batch timeout",
		},
	}

	for _, tc := range cases {
		_, err := writer.New(client, testDB, tc.batchSize, tc.batchTimeout)
		assert.Equal(t, tc.errText, err.Error(), fmt.Sprintf("%s expected to have error \"%s\", but got \"%s\"", tc.desc, tc.errText, err))
	}
}

func TestSave(t *testing.T) {
	// Set batch size to 1 to simulate single point insert.
	repo, err := writer.New(client, testDB, 1, saveTimeout)
	require.Nil(t, err, fmt.Sprintf("Creating new InfluxDB repo expected to succeed: %s.\n", err))

	// Set batch size to value > 1 to simulate real batch.
	repo1, err := writer.New(client, testDB, saveBatchSize, saveTimeout)
	require.Nil(t, err, fmt.Sprintf("Creating new InfluxDB repo expected to succeed: %s.\n", err))

	cases := []struct {
		desc         string
		repo         writers.MessageRepository
		msgsNum      int
		expectedSize int
		isBatch      bool
	}{
		{
			desc:         "save a single message",
			repo:         repo,
			msgsNum:      1,
			expectedSize: 1,
			isBatch:      false,
		},
		{
			desc:         "save a batch of messages",
			repo:         repo1,
			msgsNum:      streamsSize,
			expectedSize: streamsSize - (streamsSize % saveBatchSize),
			isBatch:      true,
		},
	}

	for _, tc := range cases {
		// Clean previously saved messages.
		_, err := queryDB(dropMsgs)
		require.Nil(t, err, fmt.Sprintf("Cleaning data from InfluxDB expected to succeed: %s.\n", err))

		now := time.Now().Unix()
		for i := 0; i < tc.msgsNum; i++ {
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

			err := tc.repo.Save(msg)
			assert.Nil(t, err, fmt.Sprintf("Save operation expected to succeed: %s.\n", err))
		}

		row, err := queryDB(selectMsgs)
		assert.Nil(t, err, fmt.Sprintf("Querying InfluxDB to retrieve data expected to succeed: %s.\n", err))

		count := len(row)
		assert.Equal(t, tc.expectedSize, count, fmt.Sprintf("Expected to have %d messages saved, found %d instead.\n", tc.expectedSize, count))

		if tc.isBatch {
			// Sleep for `saveBatchTime` to trigger ticker and check if the reset of the data is saved.
			time.Sleep(saveTimeout)

			row, err = queryDB(selectMsgs)
			assert.Nil(t, err, fmt.Sprintf("Querying InfluxDB to retrieve data count expected to succeed: %s.\n", err))
			count = len(row)
			assert.Equal(t, tc.msgsNum, count, fmt.Sprintf("Expected to have %d messages, found %d instead.\n", tc.msgsNum, count))
		}
	}
}
