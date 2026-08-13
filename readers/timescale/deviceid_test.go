// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package timescale_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	twriter "github.com/absmach/magistrala/consumers/writers/timescale"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/magistrala/pkg/testsutil"
	"github.com/absmach/magistrala/pkg/transformers/json"
	"github.com/absmach/magistrala/pkg/transformers/senml"
	"github.com/absmach/magistrala/readers"
	treader "github.com/absmach/magistrala/readers/timescale"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Serials are format-unconstrained (MG-09): case, dots, dashes and colons all
// have to round-trip byte-identically.
const (
	deviceA = "Meter.A-01:X"
	deviceB = "meter.b-02:y"
	// Never registered as a device — orphan rows must still store and read back.
	deviceOrphan = "UNREGISTERED-99"
)

func TestReadSenmlByDeviceID(t *testing.T) {
	writer := twriter.New(db)
	reader := treader.New(db)

	chanID := testsutil.GenerateUUID(t)
	pubID := testsutil.GenerateUUID(t)

	now := float64(time.Now().Unix())
	var (
		aMsgs, bMsgs, orphanMsgs, noDeviceMsgs []senml.Message
		all                                    []senml.Message
	)
	for i := 0; i < 20; i++ {
		msg := senml.Message{
			Channel:   chanID,
			Publisher: pubID,
			Protocol:  mqttProt,
			Name:      fmt.Sprintf("reading-%d", i),
			Time:      now + float64(i),
			Value:     &v,
		}
		switch i % 4 {
		case 0:
			msg.DeviceId = deviceA
			aMsgs = append(aMsgs, msg)
		case 1:
			msg.DeviceId = deviceB
			bMsgs = append(bMsgs, msg)
		case 2:
			msg.DeviceId = deviceOrphan
			orphanMsgs = append(orphanMsgs, msg)
		case 3:
			// A direct publisher: no device, forever. Not a migration artefact.
			noDeviceMsgs = append(noDeviceMsgs, msg)
		}
		all = append(all, msg)
	}

	err := writer.ConsumeBlocking(context.TODO(), all)
	require.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))

	cases := []struct {
		desc     string
		pageMeta readers.PageMetadata
		expected []senml.Message
	}{
		{
			desc:     "no device filter returns every row, device-attributed or not",
			pageMeta: readers.PageMetadata{Offset: 0, Limit: 100},
			expected: all,
		},
		{
			desc:     "empty device ids is not a filter and returns every row",
			pageMeta: readers.PageMetadata{Offset: 0, Limit: 100, DeviceIDs: []string{}},
			expected: all,
		},
		{
			desc:     "single device id returns only that device",
			pageMeta: readers.PageMetadata{Offset: 0, Limit: 100, DeviceIDs: []string{deviceA}},
			expected: aMsgs,
		},
		{
			desc:     "multiple device ids return their union",
			pageMeta: readers.PageMetadata{Offset: 0, Limit: 100, DeviceIDs: []string{deviceA, deviceB}},
			expected: append(append([]senml.Message{}, aMsgs...), bMsgs...),
		},
		{
			desc:     "a device with no entity is stored and queryable",
			pageMeta: readers.PageMetadata{Offset: 0, Limit: 100, DeviceIDs: []string{deviceOrphan}},
			expected: orphanMsgs,
		},
		{
			desc:     "an unknown device id matches nothing",
			pageMeta: readers.PageMetadata{Offset: 0, Limit: 100, DeviceIDs: []string{"nope"}},
			expected: nil,
		},
		{
			desc:     "device ids are case sensitive",
			pageMeta: readers.PageMetadata{Offset: 0, Limit: 100, DeviceIDs: []string{"METER.A-01:X"}},
			expected: nil,
		},
		{
			desc:     "device id composes with publisher",
			pageMeta: readers.PageMetadata{Offset: 0, Limit: 100, DeviceIDs: []string{deviceA}, Publisher: pubID},
			expected: aMsgs,
		},
		{
			desc:     "device id composes with publishers",
			pageMeta: readers.PageMetadata{Offset: 0, Limit: 100, DeviceIDs: []string{deviceA}, Publishers: []string{pubID}},
			expected: aMsgs,
		},
		{
			desc:     "device id composes with a non-matching publisher",
			pageMeta: readers.PageMetadata{Offset: 0, Limit: 100, DeviceIDs: []string{deviceA}, Publisher: testsutil.GenerateUUID(t)},
			expected: nil,
		},
		{
			desc:     "device id composes with a name",
			pageMeta: readers.PageMetadata{Offset: 0, Limit: 100, DeviceIDs: []string{deviceA}, Name: "reading-0"},
			expected: aMsgs[:1],
		},
		{
			desc:     "device id composes with a time range",
			pageMeta: readers.PageMetadata{Offset: 0, Limit: 100, DeviceIDs: []string{deviceA}, From: now, To: now + 5},
			expected: aMsgs[:2],
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			page, err := reader.ReadAll(chanID, tc.pageMeta)
			assert.Nil(t, err, fmt.Sprintf("expected no error got %s", err))
			assert.ElementsMatch(t, fromSenml(tc.expected), page.Messages)
			assert.Equal(t, uint64(len(tc.expected)), page.Total)
		})
	}

	t.Run("device-less rows are excluded by any device filter", func(t *testing.T) {
		require.NotEmpty(t, noDeviceMsgs)
		page, err := reader.ReadAll(chanID, readers.PageMetadata{
			Offset:    0,
			Limit:     100,
			DeviceIDs: []string{deviceA, deviceB, deviceOrphan},
		})
		assert.Nil(t, err, fmt.Sprintf("expected no error got %s", err))
		assert.Equal(t, uint64(len(all)-len(noDeviceMsgs)), page.Total)
		for _, m := range page.Messages {
			assert.NotEmpty(t, m.(senml.Message).DeviceId)
		}
	})

	t.Run("total stays correct under pagination", func(t *testing.T) {
		pageMeta := readers.PageMetadata{Offset: 0, Limit: 2, DeviceIDs: []string{deviceA}}
		page, err := reader.ReadAll(chanID, pageMeta)
		assert.Nil(t, err, fmt.Sprintf("expected no error got %s", err))
		assert.Len(t, page.Messages, 2)
		assert.Equal(t, uint64(len(aMsgs)), page.Total)

		pageMeta.Offset = uint64(len(aMsgs) - 1)
		page, err = reader.ReadAll(chanID, pageMeta)
		assert.Nil(t, err, fmt.Sprintf("expected no error got %s", err))
		assert.Len(t, page.Messages, 1)
		assert.Equal(t, uint64(len(aMsgs)), page.Total)
	})
}

// A SenML pack fanned out to N rows must carry each row's own device_id, one per
// bn group — never one value shared across the pack.
func TestSenmlPackFanoutStoresPerRecordDeviceID(t *testing.T) {
	writer := twriter.New(db)
	reader := treader.New(db)

	chanID := testsutil.GenerateUUID(t)
	pubID := testsutil.GenerateUUID(t)

	now := time.Now().Unix()
	payload := fmt.Sprintf(`[
		{"bn":%q,"n":"voltage","t":%d,"v":1},
		{"n":"current","t":%d,"v":2},
		{"bn":%q,"n":"voltage","t":%d,"v":3}
	]`, deviceA, now, now+1, deviceB, now+2)

	transformed, err := senml.New(senml.JSON).Transform(&messaging.Message{
		Channel:   chanID,
		Publisher: pubID,
		Protocol:  mqttProt,
		Payload:   []byte(payload),
		Created:   now,
	})
	require.Nil(t, err, fmt.Sprintf("expected no error got %s", err))

	require.Nil(t, writer.ConsumeBlocking(context.TODO(), transformed))

	page, err := reader.ReadAll(chanID, readers.PageMetadata{Offset: 0, Limit: 100})
	require.Nil(t, err, fmt.Sprintf("expected no error got %s", err))
	require.Len(t, page.Messages, 3)

	byName := map[string]string{}
	for _, m := range page.Messages {
		msg, ok := m.(senml.Message)
		require.True(t, ok)
		byName[msg.Name] = msg.DeviceId
	}

	// bn accumulates forward, so the middle record belongs to the first group.
	assert.Equal(t, map[string]string{
		deviceA + "voltage": deviceA,
		deviceA + "current": deviceA,
		deviceB + "voltage": deviceB,
	}, byName)

	page, err = reader.ReadAll(chanID, readers.PageMetadata{Offset: 0, Limit: 100, DeviceIDs: []string{deviceB}})
	require.Nil(t, err, fmt.Sprintf("expected no error got %s", err))
	assert.Equal(t, uint64(1), page.Total)
}

func TestReadJSONByDeviceID(t *testing.T) {
	writer := twriter.New(db)
	reader := treader.New(db)

	chanID := testsutil.GenerateUUID(t)
	pubID := testsutil.GenerateUUID(t)

	now := time.Now().Unix()
	msgs := json.Messages{Format: "device_format"}
	var withDevice, withoutDevice []map[string]any
	for i := 0; i < 6; i++ {
		msg := json.Message{
			Channel:   chanID,
			Publisher: pubID,
			Protocol:  mqttProt,
			Created:   now + int64(i),
			Payload:   map[string]any{"field": float64(i)},
		}
		expected := toMap(msg)
		if i%2 == 0 {
			msg.DeviceId = deviceA
			expected["device_id"] = deviceA
			withDevice = append(withDevice, expected)
		} else {
			withoutDevice = append(withoutDevice, expected)
		}
		msgs.Data = append(msgs.Data, msg)
	}

	require.Nil(t, writer.ConsumeBlocking(context.TODO(), msgs))

	page, err := reader.ReadAll(chanID, readers.PageMetadata{
		Offset:    0,
		Limit:     100,
		Format:    msgs.Format,
		DeviceIDs: []string{deviceA},
	})
	require.Nil(t, err, fmt.Sprintf("expected no error got %s", err))
	stripIDs(page.Messages)
	assert.ElementsMatch(t, fromJSON(withDevice), page.Messages)
	assert.Equal(t, uint64(len(withDevice)), page.Total)

	// Rows with no device are untouched: no device_id key at all, as before.
	page, err = reader.ReadAll(chanID, readers.PageMetadata{Offset: 0, Limit: 100, Format: msgs.Format})
	require.Nil(t, err, fmt.Sprintf("expected no error got %s", err))
	stripIDs(page.Messages)
	assert.ElementsMatch(t, fromJSON(append(append([]map[string]any{}, withDevice...), withoutDevice...)), page.Messages)
}

func stripIDs(msgs []readers.Message) {
	for i := range msgs {
		if m, ok := msgs[i].(map[string]any); ok {
			delete(m, "id")
		}
	}
}
