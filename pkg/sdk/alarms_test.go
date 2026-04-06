// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/absmach/magistrala/alarms"
	"github.com/absmach/magistrala/alarms/api"
	amocks "github.com/absmach/magistrala/alarms/mocks"
	mglog "github.com/absmach/magistrala/logger"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	authnmocks "github.com/absmach/magistrala/pkg/authn/mocks"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/sdk"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func setupAlarms() (*httptest.Server, *amocks.Service, *authnmocks.Authentication) {
	asvc := new(amocks.Service)
	logger := mglog.NewMock()
	authn := new(authnmocks.Authentication)
	am := smqauthn.NewAuthNMiddleware(authn, smqauthn.WithAllowUnverifiedUser(true))
	idp := uuid.NewMock()
	mux := api.MakeHandler(asvc, logger, idp, "", am)
	return httptest.NewServer(mux), asvc, authn
}

func TestUpdateAlarm(t *testing.T) {
	as, asvc, auth := setupAlarms()
	defer as.Close()

	conf := sdk.Config{
		AlarmsURL: as.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	updated := testAlarm
	updated.Status = "cleared"

	svcAlarm := alarms.Alarm{
		ID:          alarmID,
		RuleID:      "rule-1",
		DomainID:    domainID,
		ChannelID:   "chan-1",
		ClientID:    "client-1",
		Subtopic:    "subtopic",
		Status:      alarms.ClearedStatus,
		Measurement: "temperature",
		Value:       "30.5",
		Unit:        "C",
		Threshold:   "25",
		Cause:       "threshold_exceeded",
		Severity:    80,
		AssigneeID:  "user-1",
		Metadata:    alarms.Metadata{"key": "value"},
	}

	cases := []struct {
		desc            string
		alarm           sdk.Alarm
		token           string
		session         smqauthn.Session
		svcRes          alarms.Alarm
		svcErr          error
		authenticateErr error
		wantErr         bool
		resp            sdk.Alarm
	}{
		{
			desc:   "update alarm successfully",
			alarm:  updated,
			token:  validToken,
			svcRes: svcAlarm,
			resp:   testAlarm,
		},
		{
			desc:    "update alarm with empty token",
			alarm:   updated,
			token:   "",
			wantErr: true,
		},
		{
			desc:    "update non-existent alarm",
			alarm:   sdk.Alarm{ID: "non-existent"},
			token:   validToken,
			svcErr:  errors.New("not found"),
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := asvc.On("UpdateAlarm", mock.Anything, tc.session, mock.Anything).Return(tc.svcRes, tc.svcErr)
			result, err := mgsdk.UpdateAlarm(context.Background(), tc.alarm, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.NotEmpty(t, result.ID)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestViewAlarm(t *testing.T) {
	as, asvc, auth := setupAlarms()
	defer as.Close()

	conf := sdk.Config{
		AlarmsURL: as.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	svcAlarm := alarms.Alarm{
		ID:          alarmID,
		RuleID:      "rule-1",
		DomainID:    domainID,
		ChannelID:   "chan-1",
		ClientID:    "client-1",
		Subtopic:    "subtopic",
		Status:      alarms.ActiveStatus,
		Measurement: "temperature",
		Value:       "30.5",
		Unit:        "C",
		Threshold:   "25",
		Cause:       "threshold_exceeded",
		Severity:    80,
		AssigneeID:  "user-1",
		Metadata:    alarms.Metadata{"key": "value"},
	}

	cases := []struct {
		desc            string
		id              string
		token           string
		session         smqauthn.Session
		svcRes          alarms.Alarm
		svcErr          error
		authenticateErr error
		wantErr         bool
	}{
		{
			desc:   "view alarm successfully",
			id:     alarmID,
			token:  validToken,
			svcRes: svcAlarm,
		},
		{
			desc:    "view alarm with empty token",
			id:      alarmID,
			token:   "",
			wantErr: true,
		},
		{
			desc:    "view non-existent alarm",
			id:      "non-existent",
			token:   validToken,
			svcErr:  errors.New("not found"),
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := asvc.On("ViewAlarm", mock.Anything, tc.session, tc.id).Return(tc.svcRes, tc.svcErr)
			result, err := mgsdk.ViewAlarm(context.Background(), tc.id, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.NotEmpty(t, result.ID)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListAlarms(t *testing.T) {
	as, asvc, auth := setupAlarms()
	defer as.Close()

	conf := sdk.Config{
		AlarmsURL: as.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	svcAlarm := alarms.Alarm{
		ID:          alarmID,
		RuleID:      "rule-1",
		DomainID:    domainID,
		ChannelID:   "chan-1",
		ClientID:    "client-1",
		Subtopic:    "subtopic",
		Status:      alarms.ActiveStatus,
		Measurement: "temperature",
		Value:       "30.5",
		Unit:        "C",
		Threshold:   "25",
		Cause:       "threshold_exceeded",
		Severity:    80,
		AssigneeID:  "user-1",
		Metadata:    alarms.Metadata{"key": "value"},
	}

	svcAlarmsPage := alarms.AlarmsPage{
		Total:  2,
		Offset: 0,
		Limit:  10,
		Alarms: []alarms.Alarm{svcAlarm},
	}

	cases := []struct {
		desc            string
		pm              sdk.PageMetadata
		token           string
		session         smqauthn.Session
		svcRes          alarms.AlarmsPage
		svcErr          error
		authenticateErr error
		wantErr         bool
	}{
		{
			desc:   "list alarms successfully",
			pm:     sdk.PageMetadata{Offset: 0, Limit: 10},
			token:  validToken,
			svcRes: svcAlarmsPage,
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
			token:  validToken,
			svcRes: svcAlarmsPage,
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
			token:  validToken,
			svcRes: svcAlarmsPage,
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
			token:  validToken,
			svcRes: svcAlarmsPage,
		},
		{
			desc:   "list alarms with empty metadata excludes severity",
			pm:     sdk.PageMetadata{},
			token:  validToken,
			svcRes: alarms.AlarmsPage{},
		},
		{
			desc:   "list alarms with zero severity excluded",
			pm:     sdk.PageMetadata{Status: "active", Severity: 0},
			token:  validToken,
			svcRes: alarms.AlarmsPage{},
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
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := asvc.On("ListAlarms", mock.Anything, tc.session, mock.Anything).Return(tc.svcRes, tc.svcErr)
			result, err := mgsdk.ListAlarms(context.Background(), tc.pm, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.Equal(t, tc.svcRes.Total, result.Total)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDeleteAlarm(t *testing.T) {
	as, asvc, auth := setupAlarms()
	defer as.Close()

	conf := sdk.Config{
		AlarmsURL: as.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		id              string
		token           string
		session         smqauthn.Session
		svcErr          error
		authenticateErr error
		wantErr         bool
	}{
		{
			desc:  "delete alarm successfully",
			id:    alarmID,
			token: validToken,
		},
		{
			desc:    "delete alarm with empty token",
			id:      alarmID,
			token:   "",
			wantErr: true,
		},
		{
			desc:    "delete non-existent alarm",
			id:      "non-existent",
			token:   validToken,
			svcErr:  errors.New("not found"),
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := asvc.On("DeleteAlarm", mock.Anything, tc.session, tc.id).Return(tc.svcErr)
			err := mgsdk.DeleteAlarm(context.Background(), tc.id, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			svcCall.Unset()
			authCall.Unset()
		})
	}
}
