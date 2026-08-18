// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	pwriter "github.com/absmach/magistrala/consumers/writers/postgres"
	"github.com/absmach/magistrala/pkg/testsutil"
	"github.com/absmach/magistrala/pkg/transformers/senml"
	"github.com/absmach/magistrala/readers"
	preader "github.com/absmach/magistrala/readers/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// statsByID indexes a DeviceStat page by its identity (a device serial, or a
// gateway's publisher id, depending on direction) for order-independent
// assertions.
func statsByID(stats []readers.DeviceStat) map[string]readers.DeviceStat {
	out := make(map[string]readers.DeviceStat, len(stats))
	for _, s := range stats {
		out[s.ID] = s
	}
	return out
}

// Criterion 1: a gateway publishing for three devices yields exactly those
// three, with accurate last_seen and counts. A direct (non-relayed) message
// from the same publisher — no device_id at all — must not surface as a
// spurious "" roster entry.
func TestListGatewayDevicesCounts(t *testing.T) {
	writer := pwriter.New(db)
	reader := preader.New(db)

	chanID := testsutil.GenerateUUID(t)
	gatewayID := testsutil.GenerateUUID(t)

	now := float64(time.Now().Unix())
	msg := func(name, deviceID string, at float64) senml.Message {
		return senml.Message{
			Channel:   chanID,
			Publisher: gatewayID,
			Protocol:  mqttProt,
			Name:      name,
			Time:      at,
			Value:     &v,
			DeviceId:  deviceID,
		}
	}

	all := []senml.Message{
		msg("a0", "device-a", now),
		msg("a1", "device-a", now+1),
		msg("a2", "device-a", now+5),
		msg("b0", "device-b", now+2),
		msg("b1", "device-b", now+3),
		msg("c0", "device-c", now+4),
		// The gateway publishing for itself: no device_id.
		msg("self", "", now),
	}
	require.NoError(t, writer.ConsumeBlocking(context.TODO(), all))

	page, err := reader.ListGatewayDevices(chanID, gatewayID, readers.PageMetadata{
		Offset: 0, Limit: 100, From: now - 10, To: now + 100,
	})
	require.NoError(t, err)
	require.Equal(t, uint64(3), page.Total)

	byID := statsByID(page.Stats)
	require.Contains(t, byID, "device-a")
	assert.Equal(t, uint64(3), byID["device-a"].MessageCount)
	assert.Equal(t, now+5, byID["device-a"].LastSeen)

	require.Contains(t, byID, "device-b")
	assert.Equal(t, uint64(2), byID["device-b"].MessageCount)
	assert.Equal(t, now+3, byID["device-b"].LastSeen)

	require.Contains(t, byID, "device-c")
	assert.Equal(t, uint64(1), byID["device-c"].MessageCount)
	assert.Equal(t, now+4, byID["device-c"].LastSeen)

	assert.NotContains(t, byID, "", "a direct publish must not surface as a device")
}

// Criterion 2: a device published by two gateways appears in both gateways'
// rosters, and the inverse query returns both publishers.
func TestListGatewayDevicesAndInverseAcrossMultipleGateways(t *testing.T) {
	writer := pwriter.New(db)
	reader := preader.New(db)

	chanID := testsutil.GenerateUUID(t)
	gatewayA := testsutil.GenerateUUID(t)
	gatewayB := testsutil.GenerateUUID(t)
	deviceID := "shared-meter"

	now := float64(time.Now().Unix())
	msgs := []senml.Message{
		{Channel: chanID, Publisher: gatewayA, Protocol: mqttProt, Name: "r0", Time: now, Value: &v, DeviceId: deviceID},
		{Channel: chanID, Publisher: gatewayB, Protocol: mqttProt, Name: "r1", Time: now + 1, Value: &v, DeviceId: deviceID},
	}
	require.NoError(t, writer.ConsumeBlocking(context.TODO(), msgs))

	pm := readers.PageMetadata{Offset: 0, Limit: 100, From: now - 1, To: now + 10}

	pageA, err := reader.ListGatewayDevices(chanID, gatewayA, pm)
	require.NoError(t, err)
	require.Len(t, pageA.Stats, 1)
	assert.Equal(t, deviceID, pageA.Stats[0].ID)

	pageB, err := reader.ListGatewayDevices(chanID, gatewayB, pm)
	require.NoError(t, err)
	require.Len(t, pageB.Stats, 1)
	assert.Equal(t, deviceID, pageB.Stats[0].ID)

	inverse, err := reader.ListDeviceGateways(chanID, deviceID, pm)
	require.NoError(t, err)
	require.Len(t, inverse.Stats, 2)

	var publishers []string
	for _, s := range inverse.Stats {
		publishers = append(publishers, s.ID)
	}
	assert.ElementsMatch(t, []string{gatewayA, gatewayB}, publishers)
}

// Criterion 3: time bounds narrow correctly; a device silent in the window
// is absent from the result.
func TestListGatewayDevicesTimeBoundsExcludeSilentDevice(t *testing.T) {
	writer := pwriter.New(db)
	reader := preader.New(db)

	chanID := testsutil.GenerateUUID(t)
	gatewayID := testsutil.GenerateUUID(t)

	now := float64(time.Now().Unix())
	msgs := []senml.Message{
		{Channel: chanID, Publisher: gatewayID, Protocol: mqttProt, Name: "old", Time: now - 1000, Value: &v, DeviceId: "silent-device"},
		{Channel: chanID, Publisher: gatewayID, Protocol: mqttProt, Name: "recent", Time: now, Value: &v, DeviceId: "active-device"},
	}
	require.NoError(t, writer.ConsumeBlocking(context.TODO(), msgs))

	page, err := reader.ListGatewayDevices(chanID, gatewayID, readers.PageMetadata{
		Offset: 0, Limit: 100, From: now - 10, To: now + 10,
	})
	require.NoError(t, err)

	byID := statsByID(page.Stats)
	assert.Contains(t, byID, "active-device")
	assert.NotContains(t, byID, "silent-device")

	// Widening the window brings it back.
	wide, err := reader.ListGatewayDevices(chanID, gatewayID, readers.PageMetadata{
		Offset: 0, Limit: 100, From: now - 2000, To: now + 10,
	})
	require.NoError(t, err)
	assert.Contains(t, statsByID(wide.Stats), "silent-device")
}

// Criterion 4: an orphan device_id — no matching Atom entity, which this
// package has no way to know and must not try to filter — still appears.
func TestListGatewayDevicesIncludesOrphanDevice(t *testing.T) {
	writer := pwriter.New(db)
	reader := preader.New(db)

	chanID := testsutil.GenerateUUID(t)
	gatewayID := testsutil.GenerateUUID(t)
	const orphan = "UNREGISTERED-99"

	now := float64(time.Now().Unix())
	require.NoError(t, writer.ConsumeBlocking(context.TODO(), []senml.Message{
		{Channel: chanID, Publisher: gatewayID, Protocol: mqttProt, Name: "r0", Time: now, Value: &v, DeviceId: orphan},
	}))

	page, err := reader.ListGatewayDevices(chanID, gatewayID, readers.PageMetadata{
		Offset: 0, Limit: 100, From: now - 10, To: now + 10,
	})
	require.NoError(t, err)
	require.Len(t, page.Stats, 1)
	assert.Equal(t, orphan, page.Stats[0].ID)
}

// A gateway relaying for two customers' devices: DeviceScope narrows the
// roster to the caller's own devices, and an empty scope excludes
// everything — the same convention DeviceScope already follows on a plain
// message read.
func TestListGatewayDevicesAppliesDeviceScope(t *testing.T) {
	writer := pwriter.New(db)
	reader := preader.New(db)

	chanID := testsutil.GenerateUUID(t)
	gatewayID := testsutil.GenerateUUID(t)

	now := float64(time.Now().Unix())
	require.NoError(t, writer.ConsumeBlocking(context.TODO(), []senml.Message{
		{Channel: chanID, Publisher: gatewayID, Protocol: mqttProt, Name: "r0", Time: now, Value: &v, DeviceId: "meter-1"},
		{Channel: chanID, Publisher: gatewayID, Protocol: mqttProt, Name: "r1", Time: now, Value: &v, DeviceId: "meter-2"},
	}))

	scoped, err := reader.ListGatewayDevices(chanID, gatewayID, readers.PageMetadata{
		Offset: 0, Limit: 100, From: now - 10, To: now + 10,
		DeviceScope: &readers.DeviceScope{DeviceIDs: []string{"meter-1"}},
	})
	require.NoError(t, err)
	require.Len(t, scoped.Stats, 1)
	assert.Equal(t, "meter-1", scoped.Stats[0].ID)

	empty, err := reader.ListGatewayDevices(chanID, gatewayID, readers.PageMetadata{
		Offset: 0, Limit: 100, From: now - 10, To: now + 10,
		DeviceScope: &readers.DeviceScope{},
	})
	require.NoError(t, err)
	assert.Empty(t, empty.Stats)
}

// The inverse direction does not narrow by DeviceScope at all: deviceID is
// itself the authorization boundary, so a scope naming an unrelated
// publisher must not hide the relaying gateway — narrowing on the publisher
// projection here would wrongly empty out exactly the gateway-relayed case
// this feature exists to serve.
func TestListDeviceGatewaysIgnoresPublisherScope(t *testing.T) {
	writer := pwriter.New(db)
	reader := preader.New(db)

	chanID := testsutil.GenerateUUID(t)
	gatewayID := testsutil.GenerateUUID(t)
	deviceID := "meter-x"

	now := float64(time.Now().Unix())
	require.NoError(t, writer.ConsumeBlocking(context.TODO(), []senml.Message{
		{Channel: chanID, Publisher: gatewayID, Protocol: mqttProt, Name: "r0", Time: now, Value: &v, DeviceId: deviceID},
	}))

	scope := &readers.DeviceScope{PublisherIDs: []string{"unrelated-publisher"}, DeviceIDs: []string{deviceID}}
	page, err := reader.ListDeviceGateways(chanID, deviceID, readers.PageMetadata{
		Offset: 0, Limit: 100, From: now - 10, To: now + 10, DeviceScope: scope,
	})
	require.NoError(t, err)
	require.Len(t, page.Stats, 1)
	assert.Equal(t, gatewayID, page.Stats[0].ID)
}

// Cardinality: pagination bounds the roster and total still reflects the
// full count.
func TestListGatewayDevicesPagination(t *testing.T) {
	writer := pwriter.New(db)
	reader := preader.New(db)

	chanID := testsutil.GenerateUUID(t)
	gatewayID := testsutil.GenerateUUID(t)

	now := float64(time.Now().Unix())
	var msgs []senml.Message
	for i := 0; i < 5; i++ {
		msgs = append(msgs, senml.Message{
			Channel: chanID, Publisher: gatewayID, Protocol: mqttProt,
			Name: fmt.Sprintf("r%d", i), Time: now, Value: &v,
			DeviceId: fmt.Sprintf("device-%d", i),
		})
	}
	require.NoError(t, writer.ConsumeBlocking(context.TODO(), msgs))

	page, err := reader.ListGatewayDevices(chanID, gatewayID, readers.PageMetadata{
		Offset: 0, Limit: 2, From: now - 10, To: now + 10,
	})
	require.NoError(t, err)
	assert.Len(t, page.Stats, 2)
	assert.Equal(t, uint64(5), page.Total)

	rest, err := reader.ListGatewayDevices(chanID, gatewayID, readers.PageMetadata{
		Offset: 4, Limit: 2, From: now - 10, To: now + 10,
	})
	require.NoError(t, err)
	assert.Len(t, rest.Stats, 1)
	assert.Equal(t, uint64(5), rest.Total)
}

// Criterion 8: the gateway->devices query uses idx_channel_publisher_device_id
// rather than a full scan of the channel's rows. A few hundred thousand rows
// is disproportionate for a local sanity check, so this seeds a modest
// number of "other gateways" on the same busy channel — enough that channel
// alone is not a selective predicate, so only an index that also covers
// publisher keeps this from degrading into scanning every row on the
// channel to find the one gateway asked for. Full-scale performance
// validation against a realistic fleet is a follow-up, not something
// asserted here.
func TestListGatewayDevicesUsesIndex(t *testing.T) {
	writer := pwriter.New(db)

	chanID := testsutil.GenerateUUID(t)
	gatewayID := testsutil.GenerateUUID(t)
	now := float64(time.Now().Unix())

	const noiseRows = 10000
	noise := make([]senml.Message, 0, noiseRows)
	for i := 0; i < noiseRows; i++ {
		noise = append(noise, senml.Message{
			// Same busy channel, many other gateways: this is what makes
			// (channel, publisher, device_id) matter over (channel alone) or
			// (channel, device_id, publisher).
			Channel:   chanID,
			Publisher: testsutil.GenerateUUID(t),
			Protocol:  mqttProt,
			Name:      "noise",
			Time:      now,
			Value:     &v,
			DeviceId:  fmt.Sprintf("noise-device-%d", i),
		})
	}
	require.NoError(t, writer.ConsumeBlocking(context.TODO(), noise))
	require.NoError(t, writer.ConsumeBlocking(context.TODO(), []senml.Message{
		{Channel: chanID, Publisher: gatewayID, Protocol: mqttProt, Name: "r0", Time: now, Value: &v, DeviceId: "device-a"},
	}))

	_, err := db.Exec("ANALYZE messages")
	require.NoError(t, err)

	rows, err := db.Query(`EXPLAIN SELECT device_id, MAX(time) AS last_seen, COUNT(*) AS message_count
		FROM messages
		WHERE channel = $1 AND publisher = $2 AND device_id <> ''
		GROUP BY device_id
		ORDER BY MAX(time) DESC, device_id ASC
		LIMIT 10 OFFSET 0`, chanID, gatewayID)
	require.NoError(t, err)
	defer rows.Close()

	var plan strings.Builder
	for rows.Next() {
		var line string
		require.NoError(t, rows.Scan(&line))
		plan.WriteString(line)
		plan.WriteString("\n")
	}

	assert.Contains(t, plan.String(), "idx_channel_publisher_device_id",
		"expected the new index to be used, got plan:\n%s", plan.String())
}
