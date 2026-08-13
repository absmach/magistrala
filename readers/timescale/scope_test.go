// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package timescale_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	twriter "github.com/absmach/magistrala/consumers/writers/timescale"
	"github.com/absmach/magistrala/pkg/testsutil"
	"github.com/absmach/magistrala/pkg/transformers/senml"
	"github.com/absmach/magistrala/readers"
	treader "github.com/absmach/magistrala/readers/timescale"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReadWithDeviceScope exercises the authorization boundary against a real
// table, on the row shapes that make the OR necessary: a device publishing for
// itself is named by publisher and carries no device_id at all, while a device a
// gateway relays for is named by device_id and carries the gateway's publisher.
// A scope that AND'd the two would return neither.
func TestReadWithDeviceScope(t *testing.T) {
	writer := twriter.New(db)
	reader := treader.New(db)

	chanID := testsutil.GenerateUUID(t)
	meter1UUID := testsutil.GenerateUUID(t)
	meter3UUID := testsutil.GenerateUUID(t)
	gatewayUUID := testsutil.GenerateUUID(t)

	const (
		meter1Serial = "Meter.1-01:X"
		meter2Serial = "Meter.2-02:Y"
		meter3Serial = "Meter.3-03:Z"
		orphanSerial = "UNREGISTERED-99"
	)

	now := float64(time.Now().Unix())
	msg := func(i int, publisher, deviceID string) senml.Message {
		return senml.Message{
			Channel:   chanID,
			Publisher: publisher,
			Protocol:  mqttProt,
			Name:      fmt.Sprintf("reading-%d", i),
			Time:      now + float64(i),
			Value:     &v,
			DeviceId:  deviceID,
		}
	}

	// meter 1 publishes for itself: no bn in scope, so no device_id.
	direct := []senml.Message{msg(0, meter1UUID, ""), msg(1, meter1UUID, "")}
	// meters 2 and 3 are relayed by a gateway: the gateway owns the publisher id.
	relayed2 := []senml.Message{msg(2, gatewayUUID, meter2Serial)}
	relayed3 := []senml.Message{msg(3, gatewayUUID, meter3Serial), msg(4, gatewayUUID, meter3Serial)}
	// a serial with no device entity at all.
	orphan := []senml.Message{msg(5, gatewayUUID, orphanSerial)}

	all := append(append(append(append([]senml.Message{}, direct...), relayed2...), relayed3...), orphan...)
	require.Nil(t, writer.ConsumeBlocking(context.TODO(), all))

	// The customer of the PRD's worked example: granted meters 1 and 3.
	customerScope := &readers.DeviceScope{
		PublisherIDs: []string{meter1UUID, meter3UUID},
		DeviceIDs:    []string{meter1Serial, meter3Serial},
	}

	cases := []struct {
		desc     string
		pageMeta readers.PageMetadata
		expected []senml.Message
	}{
		{
			desc:     "no scope leaves the read unbounded",
			pageMeta: readers.PageMetadata{Offset: 0, Limit: 100},
			expected: all,
		},
		{
			desc:     "a granted customer reads their own devices by either identity",
			pageMeta: readers.PageMetadata{Offset: 0, Limit: 100, DeviceScope: customerScope},
			expected: append(append([]senml.Message{}, direct...), relayed3...),
		},
		{
			desc:     "an empty scope matches nothing rather than everything",
			pageMeta: readers.PageMetadata{Offset: 0, Limit: 100, DeviceScope: &readers.DeviceScope{}},
			expected: nil,
		},
		{
			desc: "a scope with only the publisher projection reaches direct rows only",
			pageMeta: readers.PageMetadata{Offset: 0, Limit: 100, DeviceScope: &readers.DeviceScope{
				PublisherIDs: []string{meter1UUID},
			}},
			expected: direct,
		},
		{
			desc: "a scope with only the serial projection reaches relayed rows only",
			pageMeta: readers.PageMetadata{Offset: 0, Limit: 100, DeviceScope: &readers.DeviceScope{
				DeviceIDs: []string{meter3Serial},
			}},
			expected: relayed3,
		},
		{
			desc: "the caller's own device filter narrows within the scope",
			pageMeta: readers.PageMetadata{
				Offset:      0,
				Limit:       100,
				DeviceScope: customerScope,
				DeviceIDs:   []string{meter3Serial},
			},
			expected: relayed3,
		},
		{
			desc: "the caller's own publisher filter narrows within the scope",
			pageMeta: readers.PageMetadata{
				Offset:      0,
				Limit:       100,
				DeviceScope: customerScope,
				Publishers:  []string{meter1UUID},
			},
			expected: direct,
		},
		{
			desc:     "orphan rows are outside every customer scope",
			pageMeta: readers.PageMetadata{Offset: 0, Limit: 100, DeviceScope: customerScope, DeviceIDs: []string{orphanSerial}},
			expected: nil,
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

	t.Run("scope bounds total under pagination too", func(t *testing.T) {
		page, err := reader.ReadAll(chanID, readers.PageMetadata{Offset: 0, Limit: 1, DeviceScope: customerScope})
		assert.Nil(t, err, fmt.Sprintf("expected no error got %s", err))
		assert.Len(t, page.Messages, 1)
		assert.Equal(t, uint64(len(direct)+len(relayed3)), page.Total)
	})
}
