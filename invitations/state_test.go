// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package invitations_test

import (
	"testing"

	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/invitations"
	"github.com/stretchr/testify/assert"
)

func TestState_String(t *testing.T) {
	tests := []struct {
		name     string
		state    invitations.State
		expected string
	}{
		{"Pending", invitations.Pending, "pending"},
		{"Accepted", invitations.Accepted, "accepted"},
		{"All", invitations.All, "all"},
		{"Unknown", invitations.State(100), "unknown"},
	}

	for _, tt := range tests {
		got := tt.state.String()
		assert.Equal(t, tt.expected, got, "State.String() = %v, expected %v", got, tt.expected)
	}
}

func TestToState(t *testing.T) {
	tests := []struct {
		name   string
		status string
		state  invitations.State
		err    error
	}{
		{"Pending", "pending", invitations.Pending, nil},
		{"Accepted", "accepted", invitations.Accepted, nil},
		{"All", "all", invitations.All, nil},
		{"Unknown", "unknown", invitations.State(0), apiutil.ErrInvitationState},
	}

	for _, tt := range tests {
		got, err := invitations.ToState(tt.status)
		assert.Equal(t, tt.err, err, "ToState() error = %v, expected %v", err, tt.err)
		assert.Equal(t, tt.state, got, "ToState() = %v, expected %v", got, tt.state)
	}
}

func TestState_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		state    invitations.State
		expected []byte
		err      error
	}{
		{"Pending", invitations.Pending, []byte(`"pending"`), nil},
		{"Accepted", invitations.Accepted, []byte(`"accepted"`), nil},
		{"All", invitations.All, []byte(`"all"`), nil},
		{"Unknown", invitations.State(100), []byte(`"unknown"`), nil},
	}

	for _, tt := range tests {
		got, err := tt.state.MarshalJSON()
		assert.Equal(t, tt.expected, got, "State.MarshalJSON() = %v, expected %v", got, tt.expected)
		assert.Equal(t, tt.err, err, "State.MarshalJSON() error = %v, expected %v", err, tt.err)
	}
}

func TestState_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		data  []byte
		state invitations.State
		err   error
	}{
		{"Pending", []byte(`"pending"`), invitations.Pending, nil},
		{"Accepted", []byte(`"accepted"`), invitations.Accepted, nil},
		{"All", []byte(`"all"`), invitations.All, nil},
		{"Unknown", []byte(`"unknown"`), invitations.State(0), apiutil.ErrInvitationState},
	}

	for _, tt := range tests {
		var state invitations.State
		err := state.UnmarshalJSON(tt.data)
		assert.Equal(t, tt.err, err, "State.UnmarshalJSON() error = %v, expected %v", err, tt.err)
		assert.Equal(t, tt.state, state, "State.UnmarshalJSON() = %v, expected %v", state, tt.state)
	}
}
