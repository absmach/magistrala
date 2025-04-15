// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package alarms_test

import (
	"fmt"
	"testing"

	"github.com/absmach/magistrala/alarms"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAlarms(t *testing.T) {
	cases := []struct {
		desc  string
		alarm alarms.Alarm
		err   error
	}{
		{
			desc: "valid alarm",
			alarm: alarms.Alarm{
				RuleID:      testsutil.GenerateUUID(t),
				DomainID:    testsutil.GenerateUUID(t),
				ChannelID:   testsutil.GenerateUUID(t),
				ClientID:    testsutil.GenerateUUID(t),
				Subtopic:    "subtopic",
				Measurement: "measurement",
				Value:       "value",
				Unit:        "unit",
				Cause:       "cause",
				Severity:    100,
			},
			err: nil,
		},
		{
			desc: "missing rule_id",
			alarm: alarms.Alarm{
				DomainID:    testsutil.GenerateUUID(t),
				ChannelID:   testsutil.GenerateUUID(t),
				ClientID:    testsutil.GenerateUUID(t),
				Subtopic:    "subtopic",
				Measurement: "measurement",
				Value:       "value",
				Unit:        "unit",
				Cause:       "cause",
				Severity:    100,
			},
			err: errors.New("rule_id is required"),
		},
		{
			desc: "missing domain_id",
			alarm: alarms.Alarm{
				RuleID:      testsutil.GenerateUUID(t),
				ChannelID:   testsutil.GenerateUUID(t),
				ClientID:    testsutil.GenerateUUID(t),
				Subtopic:    "subtopic",
				Measurement: "measurement",
				Value:       "value",
				Unit:        "unit",
				Cause:       "cause",
				Severity:    100,
			},
			err: errors.New("domain_id is required"),
		},
		{
			desc: "missing channel_id",
			alarm: alarms.Alarm{
				RuleID:      testsutil.GenerateUUID(t),
				DomainID:    testsutil.GenerateUUID(t),
				ClientID:    testsutil.GenerateUUID(t),
				Subtopic:    "subtopic",
				Measurement: "measurement",
				Value:       "value",
				Unit:        "unit",
				Cause:       "cause",
				Severity:    100,
			},
			err: errors.New("channel_id is required"),
		},
		{
			desc: "missing client_id",
			alarm: alarms.Alarm{
				RuleID:      testsutil.GenerateUUID(t),
				DomainID:    testsutil.GenerateUUID(t),
				ChannelID:   testsutil.GenerateUUID(t),
				Subtopic:    "subtopic",
				Measurement: "measurement",
				Value:       "value",
				Unit:        "unit",
				Cause:       "cause",
				Severity:    100,
			},
			err: errors.New("client_id is required"),
		},
		{
			desc: "missing subtopic",
			alarm: alarms.Alarm{
				RuleID:      testsutil.GenerateUUID(t),
				DomainID:    testsutil.GenerateUUID(t),
				ChannelID:   testsutil.GenerateUUID(t),
				ClientID:    testsutil.GenerateUUID(t),
				Measurement: "measurement",
				Value:       "value",
				Unit:        "unit",
				Cause:       "cause",
				Severity:    100,
			},
			err: errors.New("subtopic is required"),
		},
		{
			desc: "missing measurement",
			alarm: alarms.Alarm{
				RuleID:    testsutil.GenerateUUID(t),
				DomainID:  testsutil.GenerateUUID(t),
				ChannelID: testsutil.GenerateUUID(t),
				ClientID:  testsutil.GenerateUUID(t),
				Subtopic:  "subtopic",
				Value:     "value",
				Unit:      "unit",
				Cause:     "cause",
				Severity:  100,
			},
			err: errors.New("measurement is required"),
		},
		{
			desc: "missing value",
			alarm: alarms.Alarm{
				RuleID:      testsutil.GenerateUUID(t),
				DomainID:    testsutil.GenerateUUID(t),
				ChannelID:   testsutil.GenerateUUID(t),
				ClientID:    testsutil.GenerateUUID(t),
				Subtopic:    "subtopic",
				Measurement: "measurement",
				Unit:        "unit",
				Cause:       "cause",
				Severity:    100,
			},
			err: errors.New("value is required"),
		},
		{
			desc: "missing unit",
			alarm: alarms.Alarm{
				RuleID:      testsutil.GenerateUUID(t),
				DomainID:    testsutil.GenerateUUID(t),
				ChannelID:   testsutil.GenerateUUID(t),
				ClientID:    testsutil.GenerateUUID(t),
				Subtopic:    "subtopic",
				Measurement: "measurement",
				Value:       "value",
				Cause:       "cause",
				Severity:    100,
			},
			err: errors.New("unit is required"),
		},
		{
			desc: "missing cause",
			alarm: alarms.Alarm{
				RuleID:      testsutil.GenerateUUID(t),
				DomainID:    testsutil.GenerateUUID(t),
				ChannelID:   testsutil.GenerateUUID(t),
				ClientID:    testsutil.GenerateUUID(t),
				Subtopic:    "subtopic",
				Measurement: "measurement",
				Value:       "value",
				Unit:        "unit",
				Severity:    100,
			},
			err: errors.New("cause is required"),
		},
		{
			desc: "higher severity",
			alarm: alarms.Alarm{
				RuleID:      testsutil.GenerateUUID(t),
				DomainID:    testsutil.GenerateUUID(t),
				ChannelID:   testsutil.GenerateUUID(t),
				ClientID:    testsutil.GenerateUUID(t),
				Subtopic:    "subtopic",
				Measurement: "measurement",
				Value:       "value",
				Unit:        "unit",
				Cause:       "cause",
				Severity:    alarms.SeverityMax + 1,
			},
			err: alarms.ErrInvalidSeverity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.alarm.Validate()
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		})
	}
}
