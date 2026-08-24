// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"testing"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRequiredBindingsReportsAllMissingSlots(t *testing.T) {
	profile := Profile{BindingSlots: []BindingSlot{
		{Name: "temperature-sensor", Type: "client", Required: true},
		{Name: "optional-client", Type: "client", Required: false},
		{Name: "telemetry-channel", Type: "channel", Required: true},
		{Name: "command-channel", Type: "channel", Required: true},
	}}
	bindings := []BindingSnapshot{
		{Slot: "telemetry-channel", Type: "channel", ResourceID: "channel-1"},
	}

	err := validateRequiredBindings(profile, bindings)

	require.Error(t, err)
	assert.IsType(t, &errors.ServiceError{}, err)
	assert.Equal(t, "required binding slots are not bound: temperature-sensor, command-channel", err.Error())
}

func TestValidateRequestedBindingsReportsUnavailableSlotAndValidOptions(t *testing.T) {
	profile := Profile{BindingSlots: []BindingSlot{
		{Name: "temperature-sensor", Type: "client", Required: true},
		{Name: "telemetry-channel", Type: "channel", Required: true},
		{Name: "command-channel", Type: "channel", Required: true},
	}}

	err := validateRequestedBindings(profile, []BindingRequest{{
		Slot: "telemetry-channel1", Type: "channel", ResourceID: "channel-1",
	}})

	require.Error(t, err)
	assert.IsType(t, &errors.RequestError{}, err)
	assert.Equal(t,
		"binding slot \"telemetry-channel1\" is not available in the assigned profile; available slots: temperature-sensor, telemetry-channel, command-channel",
		err.Error(),
	)
}
