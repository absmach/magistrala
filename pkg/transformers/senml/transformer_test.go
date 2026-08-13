// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package senml_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/magistrala/pkg/transformers/senml"
	mgsenml "github.com/absmach/senml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransformJSON(t *testing.T) {
	// Following hex-encoded bytes correspond to the content of:
	// [{"bn":"base-name","bt":100,"bu":"base-unit","bver":10,"bv":10,"bs":100,"n":"name","u":"unit","t":300,"ut":150,"v":42,"s":10}]
	// For more details for mapping SenML labels to integers, please take a look here: https://tools.ietf.org/html/rfc8428#page-19.
	jsonBytes, err := hex.DecodeString("5b7b22626e223a22626173652d6e616d65222c226274223a3130302c226275223a22626173652d756e6974222c2262766572223a31302c226276223a31302c226273223a3130302c226e223a226e616d65222c2275223a22756e6974222c2274223a3330302c227574223a3135302c2276223a34322c2273223a31307d5d")
	assert.Nil(t, err, "Decoding JSON expected to succeed")

	tr := senml.New(senml.JSON)
	msg := &messaging.Message{
		Channel:   "channel",
		Subtopic:  "subtopic",
		Publisher: "publisher",
		Protocol:  "protocol",
		Payload:   jsonBytes,
	}

	jsonPld := msg
	jsonPld.Payload = jsonBytes

	val := 52.0
	sum := 110.0
	msgs := []senml.Message{
		{
			Channel:    "channel",
			Subtopic:   "subtopic",
			Publisher:  "publisher",
			Protocol:   "protocol",
			Name:       "base-namename",
			Unit:       "unit",
			Time:       400,
			UpdateTime: 150,
			Value:      &val,
			Sum:        &sum,
			DeviceId:   "base-name",
		},
	}

	cases := []struct {
		desc string
		msg  *messaging.Message
		msgs any
		err  error
	}{
		{
			desc: "test normalize JSON",
			msg:  jsonPld,
			msgs: msgs,
			err:  nil,
		},
		{
			desc: "test normalize defaults to JSON",
			msg:  msg,
			msgs: msgs,
			err:  nil,
		},
	}

	for _, tc := range cases {
		msgs, err := tr.Transform(tc.msg)
		assert.Equal(t, tc.msgs, msgs, fmt.Sprintf("%s expected %v, got %v", tc.desc, tc.msgs, msgs))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s, got %s", tc.desc, tc.err, err))
	}
}

func TestTransformCBOR(t *testing.T) {
	// Following hex-encoded bytes correspond to the content of:
	// [{-2: "base-name", -3: 100.0, -4: "base-unit", -1: 10, -5: 10.0, -6: 100.0, 0: "name", 1: "unit", 6: 300.0, 7: 150.0, 2: 42.0, 5: 10.0}]
	// For more details for mapping SenML labels to integers, please take a look here: https://tools.ietf.org/html/rfc8428#page-19.
	cborBytes, err := hex.DecodeString("81ac2169626173652d6e616d6522fb40590000000000002369626173652d756e6974200a24fb402400000000000025fb405900000000000000646e616d650164756e697406fb4072c0000000000007fb4062c0000000000002fb404500000000000005fb4024000000000000")
	assert.Nil(t, err, "Decoding CBOR expected to succeed")

	tooManyBytes, err := hex.DecodeString("82AD2169626173652D6E616D6522F956402369626173652D756E6974200A24F9490025F9564000646E616D650164756E697406F95CB0036331323307F958B002F9514005F94900AA2169626173652D6E616D6522F956402369626173652D756E6974200A24F9490025F9564000646E616D6506F95CB007F958B005F94900")
	assert.Nil(t, err, "Decoding CBOR expected to succeed")

	tr := senml.New(senml.CBOR)

	cborPld := &messaging.Message{
		Channel:   "channel",
		Subtopic:  "subtopic",
		Publisher: "publisher",
		Protocol:  "protocol",
		Payload:   cborBytes,
	}

	tooManyMsg := &messaging.Message{
		Channel:   "channel",
		Subtopic:  "subtopic",
		Publisher: "publisher",
		Protocol:  "protocol",
		Payload:   tooManyBytes,
	}

	val := 52.0
	sum := 110.0
	msgs := []senml.Message{
		{
			Channel:    "channel",
			Subtopic:   "subtopic",
			Publisher:  "publisher",
			Protocol:   "protocol",
			Name:       "base-namename",
			Unit:       "unit",
			Time:       400,
			UpdateTime: 150,
			Value:      &val,
			Sum:        &sum,
			DeviceId:   "base-name",
		},
	}

	cases := []struct {
		desc string
		msg  *messaging.Message
		msgs any
		err  error
	}{
		{
			desc: "test normalize CBOR",
			msg:  cborPld,
			msgs: msgs,
			err:  nil,
		},
		{
			desc: "test invalid payload",
			msg:  tooManyMsg,
			msgs: nil,
			err:  mgsenml.ErrTooManyValues,
		},
	}

	for _, tc := range cases {
		msgs, err := tr.Transform(tc.msg)
		assert.Equal(t, tc.msgs, msgs, fmt.Sprintf("%s expected %v, got %v", tc.desc, tc.msgs, msgs))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s, got %s", tc.desc, tc.err, err))
	}
}

// TestTransformDeviceIDAccumulation covers MG-05: device_id resolved per
// record from the pack's accumulated bn (RFC 8428 §4.6), not per message.
func TestTransformDeviceIDAccumulation(t *testing.T) {
	tr := senml.New(senml.JSON)

	t.Run("no bn anywhere leaves DeviceId empty", func(t *testing.T) {
		payload := []byte(`[
			{"n":"temp","t":1,"v":21.5},
			{"n":"humidity","t":2,"v":55}
		]`)
		msg := &messaging.Message{Channel: "c", Publisher: "p", Payload: payload}

		got, err := tr.Transform(msg)
		require.NoError(t, err)
		msgs, ok := got.([]senml.Message)
		require.True(t, ok)
		require.Len(t, msgs, 2)
		for _, m := range msgs {
			assert.Empty(t, m.DeviceId)
		}
	})

	t.Run("bn set once is inherited across every following record", func(t *testing.T) {
		payload := []byte(`[
			{"bn":"dev-1","n":"temp","t":1,"v":21.5},
			{"n":"humidity","t":2,"v":55},
			{"n":"pressure","t":3,"v":1013}
		]`)
		msg := &messaging.Message{Channel: "c", Publisher: "p", Payload: payload}

		got, err := tr.Transform(msg)
		require.NoError(t, err)
		msgs, ok := got.([]senml.Message)
		require.True(t, ok)
		require.Len(t, msgs, 3)
		for _, m := range msgs {
			assert.Equal(t, "dev-1", m.DeviceId)
		}
	})

	t.Run("bn changing mid-pack attributes each group to its own device", func(t *testing.T) {
		// A concentrating gateway's single flush spanning two meters — the
		// primary scenario this PRD exists for.
		payload := []byte(`[
			{"bn":"dev-1","n":"temp","t":1,"v":21.5},
			{"n":"humidity","t":2,"v":55},
			{"bn":"dev-2","n":"temp","t":3,"v":19.0},
			{"n":"humidity","t":4,"v":60}
		]`)
		msg := &messaging.Message{Channel: "c", Publisher: "p", Payload: payload}

		got, err := tr.Transform(msg)
		require.NoError(t, err)
		msgs, ok := got.([]senml.Message)
		require.True(t, ok)
		require.Len(t, msgs, 4)

		// Keyed by Name (unique per record here) rather than slice position:
		// Normalize sorts by Time, and ties are not guaranteed stable.
		byName := make(map[string]string, len(msgs))
		for _, m := range msgs {
			byName[m.Name] = m.DeviceId
		}
		assert.Equal(t, map[string]string{
			"dev-1temp":     "dev-1",
			"dev-1humidity": "dev-1",
			"dev-2temp":     "dev-2",
			"dev-2humidity": "dev-2",
		}, byName)
	})

	t.Run("an explicit empty bn does not reset accumulation", func(t *testing.T) {
		payload := []byte(`[
			{"bn":"dev-1","n":"temp","t":1,"v":21.5},
			{"bn":"","n":"humidity","t":2,"v":55},
			{"n":"pressure","t":3,"v":1013}
		]`)
		msg := &messaging.Message{Channel: "c", Publisher: "p", Payload: payload}

		got, err := tr.Transform(msg)
		require.NoError(t, err)
		msgs, ok := got.([]senml.Message)
		require.True(t, ok)
		require.Len(t, msgs, 3)
		for _, m := range msgs {
			assert.Equal(t, "dev-1", m.DeviceId)
		}
	})

	t.Run("device id is carried verbatim", func(t *testing.T) {
		payload := []byte(`[{"bn":"DEV.7-A:07_x","n":"temp","t":1,"v":21.5}]`)
		msg := &messaging.Message{Channel: "c", Publisher: "p", Payload: payload}

		got, err := tr.Transform(msg)
		require.NoError(t, err)
		msgs, ok := got.([]senml.Message)
		require.True(t, ok)
		require.Len(t, msgs, 1)
		assert.Equal(t, "DEV.7-A:07_x", msgs[0].DeviceId)
	})
}
