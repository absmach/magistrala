// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package json_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/pkg/transformers/json"
	"github.com/stretchr/testify/assert"
)

const (
	validPayload   = `{"key1": "val1", "key2": 123, "key3": "val3", "key4": {"key5": "val5"}}`
	listPayload    = `[{"key1": "val1", "key2": 123, "keylist3": "val3", "key4": {"key5": "val5"}}, {"key1": "val1", "key2": 123, "key3": "val3", "key4": {"key5": "val5"}}]`
	invalidPayload = `{"key1": "val1", "key2": 123, "key3/1": "val3", "key4": {"key5": "val5"}}`
)

func TestTransformJSON(t *testing.T) {
	now := time.Now().Unix()
	tr := json.New()
	msg := messaging.Message{
		Channel:   "channel-1",
		Subtopic:  "subtopic-1",
		Publisher: "publisher-1",
		Protocol:  "protocol",
		Payload:   []byte(validPayload),
		Created:   now,
	}
	invalid := msg
	invalid.Payload = []byte(invalidPayload)

	listMsg := msg
	listMsg.Payload = []byte(listPayload)

	jsonMsg := json.Messages{
		Data: []json.Message{
			{
				Channel:   msg.Channel,
				Subtopic:  msg.Subtopic,
				Publisher: msg.Publisher,
				Protocol:  msg.Protocol,
				Created:   msg.Created,
				Payload: map[string]interface{}{
					"key1":      "val1",
					"key2":      float64(123),
					"key3":      "val3",
					"key4/key5": "val5",
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
				Payload: map[string]interface{}{
					"key1":      "val1",
					"key2":      float64(123),
					"keylist3":  "val3",
					"key4/key5": "val5",
				},
			},
			{
				Channel:   msg.Channel,
				Subtopic:  msg.Subtopic,
				Publisher: msg.Publisher,
				Protocol:  msg.Protocol,
				Created:   msg.Created,
				Payload: map[string]interface{}{
					"key1":      "val1",
					"key2":      float64(123),
					"key3":      "val3",
					"key4/key5": "val5",
				},
			},
		},
		Format: msg.Subtopic,
	}

	cases := []struct {
		desc string
		msg  messaging.Message
		json interface{}
		err  error
	}{
		{
			desc: "test transform JSON",
			msg:  msg,
			json: jsonMsg,
			err:  nil,
		},
		{
			desc: "test transform JSON array",
			msg:  listMsg,
			json: listJSON,
			err:  nil,
		},
		{
			desc: "test transform JSON with invalid payload",
			msg:  invalid,
			json: nil,
			err:  json.ErrTransform,
		},
	}

	for _, tc := range cases {
		m, err := tr.Transform(tc.msg)
		assert.Equal(t, tc.json, m, fmt.Sprintf("%s expected %v, got %v", tc.desc, tc.json, m))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s, got %s", tc.desc, tc.err, err))
	}
}
