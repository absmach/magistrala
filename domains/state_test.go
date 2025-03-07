// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package domains_test

import (
	"testing"

	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/domains"
	"github.com/stretchr/testify/assert"
)

func TestState_String(t *testing.T) {
	tests := []struct {
		name     string
		state    domains.State
		expected string
	}{
		{"Pending", domains.Pending, "pending"},
		{"Accepted", domains.Accepted, "accepted"},
		{"Rejected", domains.Rejected, "rejected"},
		{"All", domains.AllState, "all"},
		{"Unknown", domains.State(100), "unknown"},
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
		state  domains.State
		err    error
	}{
		{"Pending", "pending", domains.Pending, nil},
		{"Accepted", "accepted", domains.Accepted, nil},
		{"Rejected", "rejected", domains.Rejected, nil},
		{"All", "all", domains.AllState, nil},
		{"Unknown", "unknown", domains.State(0), apiutil.ErrInvitationState},
	}

	for _, tt := range tests {
		got, err := domains.ToState(tt.status)
		assert.Equal(t, tt.err, err, "ToState() error = %v, expected %v", err, tt.err)
		assert.Equal(t, tt.state, got, "ToState() = %v, expected %v", got, tt.state)
	}
}

func TestState_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		state    domains.State
		expected []byte
		err      error
	}{
		{"Pending", domains.Pending, []byte(`"pending"`), nil},
		{"Accepted", domains.Accepted, []byte(`"accepted"`), nil},
		{"Rejected", domains.Rejected, []byte(`"rejected"`), nil},
		{"All", domains.AllState, []byte(`"all"`), nil},
		{"Unknown", domains.State(100), []byte(`"unknown"`), nil},
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
		state domains.State
		err   error
	}{
		{"Pending", []byte(`"pending"`), domains.Pending, nil},
		{"Accepted", []byte(`"accepted"`), domains.Accepted, nil},
		{"Rejected", []byte(`"rejected"`), domains.Rejected, nil},
		{"All", []byte(`"all"`), domains.AllState, nil},
		{"Unknown", []byte(`"unknown"`), domains.State(0), apiutil.ErrInvitationState},
	}

	for _, tt := range tests {
		var state domains.State
		err := state.UnmarshalJSON(tt.data)
		assert.Equal(t, tt.err, err, "State.UnmarshalJSON() error = %v, expected %v", err, tt.err)
		assert.Equal(t, tt.state, state, "State.UnmarshalJSON() = %v, expected %v", state, tt.state)
	}
}
