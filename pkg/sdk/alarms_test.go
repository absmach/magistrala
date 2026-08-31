// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/absmach/magistrala/pkg/sdk"
	"github.com/stretchr/testify/require"
)

func TestPatchAlarmPreservesExplicitZeroValues(t *testing.T) {
	var got sdk.AlarmUpdate
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		require.Equal(t, http.MethodPatch, req.Method)
		require.Equal(t, "/workspace-1/alarms/alarm-1", req.URL.Path)
		require.NoError(t, json.NewDecoder(req.Body).Decode(&got))
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(sdk.Alarm{ID: "alarm-1"}))
	}))
	defer server.Close()

	status := "active"
	severity := uint8(0)
	assignee := ""
	metadata := sdk.Metadata{}
	client := sdk.NewSDK(sdk.Config{AlarmsURL: server.URL})

	alarm, err := client.PatchAlarm(context.Background(), "alarm-1", sdk.AlarmUpdate{
		Status:     &status,
		Severity:   &severity,
		AssigneeID: &assignee,
		Metadata:   &metadata,
	}, "workspace-1", "token")

	require.Nil(t, err)
	require.Equal(t, "alarm-1", alarm.ID)
	require.NotNil(t, got.Status)
	require.Equal(t, status, *got.Status)
	require.NotNil(t, got.Severity)
	require.Zero(t, *got.Severity)
	require.NotNil(t, got.AssigneeID)
	require.Empty(t, *got.AssigneeID)
	require.NotNil(t, got.Metadata)
	require.Empty(t, *got.Metadata)
}
