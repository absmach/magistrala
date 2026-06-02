// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package messaging

import "testing"

func TestClientIdentity(t *testing.T) {
	cases := []struct {
		name string
		msg  *Message
		want string
	}{
		{
			name: "nil message",
			msg:  nil,
			want: "",
		},
		{
			name: "publisher wins over transport client id",
			msg: &Message{
				Publisher: "entity-1",
				ClientId:  "amqp091:connection",
			},
			want: "entity-1",
		},
		{
			name: "fallback to transport client id for legacy messages",
			msg: &Message{
				ClientId: "legacy-client",
			},
			want: "legacy-client",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.msg.ClientIdentity(); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}
