// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package json_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/magistrala/pkg/transformers/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	validPayload     = `{"key1": "val1", "key2": 123, "key3": "val3", "key4": {"key5": "val5"}}`
	tsPayload        = `{"custom_ts_key": "1638310819", "key1": "val1", "key2": 123, "key3": "val3", "key4": {"key5": "val5"}}`
	microsPayload    = `{"custom_ts_micro_key": "1638310819000000", "key1": "val1", "key2": 123, "key3": "val3", "key4": {"key5": "val5"}}`
	invalidTSPayload = `{"custom_ts_key": "abc", "key1": "val1", "key2": 123, "key3": "val3", "key4": {"key5": "val5"}}`
	listPayload      = `[{"key1": "val1", "key2": 123, "keylist3": "val3", "key4": {"key5": "val5"}}, {"key1": "val1", "key2": 123, "key3": "val3", "key4": {"key5": "val5"}}]`
	invalidPayload   = `{"key1": }`
)

func TestTransformJSON(t *testing.T) {
	now := time.Now().Unix()
	ts := []json.TimeField{
		{
			FieldName:   "custom_ts_key",
			FieldFormat: "unix",
		}, {
			FieldName:   "custom_ts_micro_key",
			FieldFormat: "unix_us",
		},
	}
	tr := json.New(ts)
	msg := messaging.Message{
		Channel:   "channel-1",
		Subtopic:  "subtopic-1",
		Publisher: "publisher-1",
		Protocol:  "protocol",
		Payload:   []byte(validPayload),
		Created:   now,
	}
	invalid := messaging.Message{
		Channel:   "channel-1",
		Subtopic:  "subtopic-1",
		Publisher: "publisher-1",
		Protocol:  "protocol",
		Payload:   []byte(invalidPayload),
		Created:   now,
	}

	listMsg := messaging.Message{
		Channel:   "channel-1",
		Subtopic:  "subtopic-1",
		Publisher: "publisher-1",
		Protocol:  "protocol",
		Payload:   []byte(listPayload),
		Created:   now,
	}

	tsMsg := messaging.Message{
		Channel:   "channel-1",
		Subtopic:  "subtopic-1",
		Publisher: "publisher-1",
		Protocol:  "protocol",
		Payload:   []byte(tsPayload),
		Created:   now,
	}

	microsMsg := messaging.Message{
		Channel:   "channel-1",
		Subtopic:  "subtopic-1",
		Publisher: "publisher-1",
		Protocol:  "protocol",
		Payload:   []byte(microsPayload),
		Created:   now,
	}

	invalidFmt := messaging.Message{
		Channel:   "channel-1",
		Subtopic:  "",
		Publisher: "publisher-1",
		Protocol:  "protocol",
		Payload:   []byte(validPayload),
		Created:   now,
	}

	invalidTimeField := messaging.Message{
		Channel:   "channel-1",
		Subtopic:  "subtopic-1",
		Publisher: "publisher-1",
		Protocol:  "protocol",
		Payload:   []byte(invalidTSPayload),
		Created:   now,
	}

	jsonMsgs := json.Messages{
		Data: []json.Message{
			{
				Channel:   msg.Channel,
				Subtopic:  msg.Subtopic,
				Publisher: msg.Publisher,
				Protocol:  msg.Protocol,
				Created:   msg.Created,
				Payload: map[string]any{
					"key1": "val1",
					"key2": float64(123),
					"key3": "val3",
					"key4": map[string]any{
						"key5": "val5",
					},
				},
			},
		},
		Format: msg.Subtopic,
	}

	jsonTSMsgs := json.Messages{
		Data: []json.Message{
			{
				Channel:   msg.Channel,
				Subtopic:  msg.Subtopic,
				Publisher: msg.Publisher,
				Protocol:  msg.Protocol,
				Created:   int64(1638310819000000000),
				Payload: map[string]any{
					"custom_ts_key": "1638310819",
					"key1":          "val1",
					"key2":          float64(123),
					"key3":          "val3",
					"key4": map[string]any{
						"key5": "val5",
					},
				},
			},
		},
		Format: msg.Subtopic,
	}

	jsonMicrosMsgs := json.Messages{
		Data: []json.Message{
			{
				Channel:   msg.Channel,
				Subtopic:  msg.Subtopic,
				Publisher: msg.Publisher,
				Protocol:  msg.Protocol,
				Created:   int64(1638310819000000000),
				Payload: map[string]any{
					"custom_ts_micro_key": "1638310819000000",
					"key1":                "val1",
					"key2":                float64(123),
					"key3":                "val3",
					"key4": map[string]any{
						"key5": "val5",
					},
				},
			},
		},
		Format: msg.Subtopic,
	}

	listJSON := json.Messages{
		Data: []json.Message{
			{
				Channel:   msg.Channel,
				Subtopic:  msg.Subtopic,
				Publisher: msg.Publisher,
				Protocol:  msg.Protocol,
				Created:   msg.Created,
				Payload: map[string]any{
					"key1":     "val1",
					"key2":     float64(123),
					"keylist3": "val3",
					"key4": map[string]any{
						"key5": "val5",
					},
				},
			},
			{
				Channel:   msg.Channel,
				Subtopic:  msg.Subtopic,
				Publisher: msg.Publisher,
				Protocol:  msg.Protocol,
				Created:   msg.Created,
				Payload: map[string]any{
					"key1": "val1",
					"key2": float64(123),
					"key3": "val3",
					"key4": map[string]any{
						"key5": "val5",
					},
				},
			},
		},
		Format: msg.Subtopic,
	}

	cases := []struct {
		desc string
		msg  *messaging.Message
		json any
		err  error
	}{
		{
			desc: "test transform JSON",
			msg:  &msg,
			json: jsonMsgs,
			err:  nil,
		},
		{
			desc: "test transform JSON with an invalid subtopic",
			msg:  &invalidFmt,
			json: nil,
			err:  json.ErrTransform,
		},
		{
			desc: "test transform JSON array",
			msg:  &listMsg,
			json: listJSON,
			err:  nil,
		},
		{
			desc: "test transform JSON with invalid payload",
			msg:  &invalid,
			json: nil,
			err:  json.ErrTransform,
		},
		{
			desc: "test transform JSON with timestamp transformation",
			msg:  &tsMsg,
			json: jsonTSMsgs,
			err:  nil,
		},
		{
			desc: "test transform JSON with timestamp transformation in micros",
			msg:  &microsMsg,
			json: jsonMicrosMsgs,
			err:  nil,
		},
		{
			desc: "test transform JSON with invalid timestamp transformation in micros",
			msg:  &invalidTimeField,
			json: nil,
			err:  json.ErrInvalidTimeField,
		},
	}

	for _, tc := range cases {
		m, err := tr.Transform(tc.msg)
		assert.Equal(t, tc.json, m, fmt.Sprintf("%s got incorrect json response from Transform()", tc.desc))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s, got %s", tc.desc, tc.err, err))
	}
}

// TestTransformDeviceID covers MG-05: device_id popped from the reserved
// "device_id" key, accumulating forward across a batch the same way SenML's
// bn does (RFC 8428 §4.6).
func TestTransformDeviceID(t *testing.T) {
	tr := json.New(nil)
	newMsg := func(payload string) *messaging.Message {
		return &messaging.Message{
			Channel:   "channel-1",
			Subtopic:  "subtopic-1",
			Publisher: "publisher-1",
			Protocol:  "protocol",
			Payload:   []byte(payload),
		}
	}

	t.Run("single object: device_id is popped from Payload and set on the message", func(t *testing.T) {
		got, err := tr.Transform(newMsg(`{"device_id":"dev-1","temp":21.5}`))
		require.NoError(t, err)
		msgs, ok := got.(json.Messages)
		require.True(t, ok)
		require.Len(t, msgs.Data, 1)
		assert.Equal(t, "dev-1", msgs.Data[0].DeviceId)
		assert.NotContains(t, msgs.Data[0].Payload, "device_id")
		assert.Equal(t, map[string]any{"temp": 21.5}, map[string]any(msgs.Data[0].Payload))
	})

	t.Run("single object: no device_id leaves DeviceId empty and Payload untouched", func(t *testing.T) {
		got, err := tr.Transform(newMsg(`{"temp":21.5}`))
		require.NoError(t, err)
		msgs, ok := got.(json.Messages)
		require.True(t, ok)
		require.Len(t, msgs.Data, 1)
		assert.Empty(t, msgs.Data[0].DeviceId)
		assert.Equal(t, map[string]any{"temp": 21.5}, map[string]any(msgs.Data[0].Payload))
	})

	t.Run("array: device_id set once is inherited across every following element", func(t *testing.T) {
		got, err := tr.Transform(newMsg(`[{"device_id":"dev-1","temp":21.5},{"humidity":55},{"pressure":1013}]`))
		require.NoError(t, err)
		msgs, ok := got.(json.Messages)
		require.True(t, ok)
		require.Len(t, msgs.Data, 3)
		for _, m := range msgs.Data {
			assert.Equal(t, "dev-1", m.DeviceId)
		}
	})

	t.Run("array: device_id changing mid-array attributes each element to its own device", func(t *testing.T) {
		// A concentrating gateway's single flush spanning two meters — the
		// primary scenario this PRD exists for.
		got, err := tr.Transform(newMsg(`[
			{"device_id":"dev-1","temp":21.5},
			{"humidity":55},
			{"device_id":"dev-2","temp":19.0},
			{"humidity":60}
		]`))
		require.NoError(t, err)
		msgs, ok := got.(json.Messages)
		require.True(t, ok)
		require.Len(t, msgs.Data, 4)
		want := []string{"dev-1", "dev-1", "dev-2", "dev-2"}
		for i, m := range msgs.Data {
			assert.Equal(t, want[i], m.DeviceId, "element %d", i)
		}
	})

	t.Run("array: an explicit empty device_id does not reset accumulation", func(t *testing.T) {
		got, err := tr.Transform(newMsg(`[
			{"device_id":"dev-1","temp":21.5},
			{"device_id":"","humidity":55},
			{"pressure":1013}
		]`))
		require.NoError(t, err)
		msgs, ok := got.(json.Messages)
		require.True(t, ok)
		require.Len(t, msgs.Data, 3)
		for _, m := range msgs.Data {
			assert.Equal(t, "dev-1", m.DeviceId)
		}
	})

	t.Run("device id is carried verbatim, including unicode", func(t *testing.T) {
		got, err := tr.Transform(newMsg(`{"device_id":"dév-测试_1","temp":21.5}`))
		require.NoError(t, err)
		msgs, ok := got.(json.Messages)
		require.True(t, ok)
		require.Len(t, msgs.Data, 1)
		assert.Equal(t, "dév-测试_1", msgs.Data[0].DeviceId)
	})
}
