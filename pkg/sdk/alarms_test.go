// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/absmach/magistrala/pkg/sdk"
	"github.com/stretchr/testify/assert"
)

const alarmID = "alarm-1"

var testAlarm = sdk.Alarm{
	ID:          alarmID,
	RuleID:      "rule-1",
	DomainID:    domainID,
	ChannelID:   "chan-1",
	ClientID:    "client-1",
	Subtopic:    "subtopic",
	Status:      "active",
	Measurement: "temperature",
	Value:       "30.5",
	Unit:        "C",
	Threshold:   "25",
	Cause:       "threshold_exceeded",
	Severity:    80,
	AssigneeID:  "user-1",
	Metadata:    sdk.Metadata{"key": "value"},
}

func TestUpdateAlarm(t *testing.T) {
	updated := testAlarm
	updated.Status = "acknowledged"

	cases := []struct {
		desc    string
		alarm   sdk.Alarm
		token   string
		handler http.HandlerFunc
		wantErr bool
		resp    sdk.Alarm
	}{
		{
			desc:  "update alarm successfully",
			alarm: testAlarm,
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPut, r.Method)
				assert.Equal(t, fmt.Sprintf("/%s/alarms/%s", domainID, testAlarm.ID), r.URL.Path)
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(updated)
			},
			resp: updated,
		},
		{
			desc:  "update alarm with empty token",
			alarm: testAlarm,
			token: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantErr: true,
		},
		{
			desc:  "update non-existent alarm",
			alarm: sdk.Alarm{ID: "non-existent"},
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			mgsdk := sdk.NewSDK(sdk.Config{AlarmsURL: server.URL})
			result, err := mgsdk.UpdateAlarm(context.Background(), tc.alarm, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.Equal(t, tc.resp, result)
			}
		})
	}
}

func TestViewAlarm(t *testing.T) {
	cases := []struct {
		desc    string
		id      string
		token   string
		handler http.HandlerFunc
		wantErr bool
		resp    sdk.Alarm
	}{
		{
			desc:  "view alarm successfully",
			id:    alarmID,
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, fmt.Sprintf("/%s/alarms/%s", domainID, alarmID), r.URL.Path)
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(testAlarm)
			},
			resp: testAlarm,
		},
		{
			desc:  "view alarm with empty token",
			id:    alarmID,
			token: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantErr: true,
		},
		{
			desc:  "view non-existent alarm",
			id:    "non-existent",
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			mgsdk := sdk.NewSDK(sdk.Config{AlarmsURL: server.URL})
			result, err := mgsdk.ViewAlarm(context.Background(), tc.id, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.Equal(t, tc.resp, result)
			}
		})
	}
}

func TestListAlarms(t *testing.T) {
	alarms := []sdk.Alarm{testAlarm, {ID: "alarm-2", ChannelID: "chan-2", Status: "resolved"}}
	alarmsPage := sdk.AlarmsPage{Total: 2, Offset: 0, Limit: 10, Alarms: alarms}

	cases := []struct {
		desc    string
		pm      sdk.PageMetadata
		token   string
		checkQ  func(t *testing.T, r *http.Request)
		wantErr bool
		resp    sdk.AlarmsPage
	}{
		{
			desc:  "list alarms successfully",
			pm:    sdk.PageMetadata{Offset: 0, Limit: 10},
			token: validToken,
			checkQ: func(t *testing.T, r *http.Request) {
				assert.Equal(t, "10", r.URL.Query().Get("limit"))
			},
			resp: alarmsPage,
		},
		{
			desc: "list alarms with status and entity filters",
			pm: sdk.PageMetadata{
				Limit:      5,
				Status:     "active",
				ChannelID:  "chan-1",
				ClientID:   "client-1",
				RuleID:     "rule-1",
				AssigneeID: "user-1",
				Severity:   80,
			},
			token: validToken,
			checkQ: func(t *testing.T, r *http.Request) {
				q := r.URL.Query()
				assert.Equal(t, "active", q.Get("status"))
				assert.Equal(t, "chan-1", q.Get("channel_id"))
				assert.Equal(t, "client-1", q.Get("client_id"))
				assert.Equal(t, "rule-1", q.Get("rule_id"))
				assert.Equal(t, "user-1", q.Get("assignee_id"))
				assert.Equal(t, "80", q.Get("severity"))
			},
			resp: alarmsPage,
		},
		{
			desc: "list alarms with time range and sorting",
			pm: sdk.PageMetadata{
				Limit:       10,
				CreatedFrom: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				CreatedTo:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
				Order:       "created_at",
				Dir:         "asc",
			},
			token: validToken,
			checkQ: func(t *testing.T, r *http.Request) {
				q := r.URL.Query()
				assert.Equal(t, "2024-01-01T00:00:00Z", q.Get("created_from"))
				assert.Equal(t, "2024-12-31T00:00:00Z", q.Get("created_to"))
				assert.Equal(t, "created_at", q.Get("order"))
				assert.Equal(t, "asc", q.Get("dir"))
			},
			resp: alarmsPage,
		},
		{
			desc: "list alarms with actor filters",
			pm: sdk.PageMetadata{
				Limit:          10,
				UpdatedBy:      "user-2",
				AssignedBy:     "user-3",
				AcknowledgedBy: "user-4",
				ResolvedBy:     "user-5",
				Subtopic:       "subtopic-1",
			},
			token: validToken,
			checkQ: func(t *testing.T, r *http.Request) {
				q := r.URL.Query()
				assert.Equal(t, "user-2", q.Get("updated_by"))
				assert.Equal(t, "user-3", q.Get("assigned_by"))
				assert.Equal(t, "user-4", q.Get("acknowledged_by"))
				assert.Equal(t, "user-5", q.Get("resolved_by"))
				assert.Equal(t, "subtopic-1", q.Get("subtopic"))
			},
			resp: alarmsPage,
		},
		{
			desc:  "list alarms with empty metadata excludes severity",
			pm:    sdk.PageMetadata{},
			token: validToken,
			checkQ: func(t *testing.T, r *http.Request) {
				assert.NotContains(t, r.URL.RawQuery, "severity")
			},
			resp: sdk.AlarmsPage{},
		},
		{
			desc:  "list alarms with zero severity excluded",
			pm:    sdk.PageMetadata{Status: "active", Severity: 0},
			token: validToken,
			checkQ: func(t *testing.T, r *http.Request) {
				assert.Equal(t, "active", r.URL.Query().Get("status"))
				assert.NotContains(t, r.URL.RawQuery, "severity")
			},
			resp: sdk.AlarmsPage{},
		},
		{
			desc:    "list alarms with empty token",
			pm:      sdk.PageMetadata{Limit: 10},
			token:   "",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, fmt.Sprintf("/%s/alarms", domainID), r.URL.Path)
				if r.Header.Get("Authorization") == "" || r.Header.Get("Authorization") == "Bearer " {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				if tc.checkQ != nil {
					tc.checkQ(t, r)
				}
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(tc.resp)
			}))
			defer server.Close()

			mgsdk := sdk.NewSDK(sdk.Config{AlarmsURL: server.URL})
			result, err := mgsdk.ListAlarms(context.Background(), tc.pm, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.Equal(t, tc.resp, result)
			}
		})
	}
}

func TestDeleteAlarm(t *testing.T) {
	cases := []struct {
		desc    string
		id      string
		token   string
		handler http.HandlerFunc
		wantErr bool
	}{
		{
			desc:  "delete alarm successfully",
			id:    alarmID,
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, fmt.Sprintf("/%s/alarms/%s", domainID, alarmID), r.URL.Path)
				w.WriteHeader(http.StatusNoContent)
			},
		},
		{
			desc:  "delete alarm with empty token",
			id:    alarmID,
			token: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantErr: true,
		},
		{
			desc:  "delete non-existent alarm",
			id:    "non-existent",
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			mgsdk := sdk.NewSDK(sdk.Config{AlarmsURL: server.URL})
			err := mgsdk.DeleteAlarm(context.Background(), tc.id, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
		})
	}
}

