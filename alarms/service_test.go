// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package alarms_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/absmach/magistrala/alarms"
	"github.com/absmach/magistrala/alarms/mocks"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/supermq/pkg/authn"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var idp = uuid.New()

func TestCreateAlarm(t *testing.T) {
	repo := new(mocks.Repository)
	svc := alarms.NewService(idp, repo)

	cases := []struct {
		desc  string
		alarm alarms.Alarm
		err   error
	}{
		{
			desc: "valid alarm",
			alarm: alarms.Alarm{
				RuleID:      "rule-id",
				DomainID:    "domain-id",
				ChannelID:   "channel-id",
				ClientID:    "client-id",
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
				DomainID:    "domain-id",
				ChannelID:   "channel-id",
				ClientID:    "client-id",
				Subtopic:    "subtopic",
				Measurement: "measurement",
				Value:       "value",
				Unit:        "unit",
				Cause:       "cause",
				Severity:    100,
			},
			err: errors.New("rule_id is required"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("CreateAlarm", context.Background(), mock.Anything).Return(tc.alarm, tc.err)
			repoCall1 := repo.On("ListAlarms", context.Background(), alarms.PageMetadata{
				Offset: 0, Limit: 1,
				DomainID:  tc.alarm.DomainID,
				ChannelID: tc.alarm.ChannelID,
				ClientID:  tc.alarm.ClientID,
				Subtopic:  tc.alarm.Subtopic,
				RuleID:    tc.alarm.RuleID,
				Severity:  tc.alarm.Severity,
				Status:    tc.alarm.Status,
			}).Return(alarms.AlarmsPage{}, tc.err)
			err := svc.CreateAlarm(context.Background(), tc.alarm)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

				return
			}
			repoCall.Unset()
			repoCall1.Unset()
		})
	}
}

func TestViewAlarm(t *testing.T) {
	repo := new(mocks.Repository)
	svc := alarms.NewService(idp, repo)

	cases := []struct {
		desc     string
		id       string
		domainID string
		err      error
	}{
		{
			desc:     "valid alarm",
			id:       "alarm-id",
			domainID: "domain-id",
			err:      nil,
		},
		{
			desc:     "non existing alarm id",
			id:       "alarm-id",
			domainID: "domain-id",
			err:      repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			s := authn.Session{DomainID: tc.domainID}
			repoCall := repo.On("ViewAlarm", context.Background(), tc.id, tc.domainID).Return(alarms.Alarm{}, tc.err)
			_, err := svc.ViewAlarm(context.Background(), s, tc.id)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

				return
			}
			repoCall.Unset()
		})
	}
}

func TestUpdateAlarm(t *testing.T) {
	repo := new(mocks.Repository)
	svc := alarms.NewService(idp, repo)

	cases := []struct {
		desc  string
		alarm alarms.Alarm
		err   error
	}{
		{
			desc: "valid alarm",
			alarm: alarms.Alarm{
				RuleID:      "rule-id",
				DomainID:    "domain-id",
				ChannelID:   "channel-id",
				ClientID:    "client-id",
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
			desc: "non existing alarm",
			alarm: alarms.Alarm{
				RuleID:      "rule-id",
				DomainID:    "domain-id",
				ChannelID:   "channel-id",
				ClientID:    "client-id",
				Subtopic:    "subtopic",
				Measurement: "measurement",
				Value:       "value",
				Unit:        "unit",
				Cause:       "cause",
				Severity:    100,
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			s := authn.Session{DomainID: tc.alarm.DomainID}
			repoCall := repo.On("UpdateAlarm", context.Background(), mock.Anything).Return(tc.alarm, tc.err)
			_, err := svc.UpdateAlarm(context.Background(), s, tc.alarm)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

				return
			}
			repoCall.Unset()
		})
	}
}

func TestListAlarms(t *testing.T) {
	repo := new(mocks.Repository)
	svc := alarms.NewService(idp, repo)

	cases := []struct {
		desc string
		pm   alarms.PageMetadata
		page alarms.AlarmsPage
		err  error
	}{
		{
			desc: "valid page",
			pm: alarms.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			page: alarms.AlarmsPage{
				Offset: 0,
				Limit:  10,
				Total:  10,
				Alarms: []alarms.Alarm{},
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			s := authn.Session{DomainID: tc.pm.DomainID}
			repoCall := repo.On("ListAlarms", context.Background(), tc.pm).Return(tc.page, tc.err)
			_, err := svc.ListAlarms(context.Background(), s, tc.pm)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

				return
			}
			repoCall.Unset()
		})
	}
}

func TestDeleteAlarm(t *testing.T) {
	repo := new(mocks.Repository)
	svc := alarms.NewService(idp, repo)

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "valid alarm",
			id:   "alarm-id",
			err:  nil,
		},
		{
			desc: "non existing alarm",
			id:   "alarm-id",
			err:  repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			s := authn.Session{DomainID: tc.id}
			repoCall := repo.On("DeleteAlarm", context.Background(), tc.id).Return(tc.err)
			err := svc.DeleteAlarm(context.Background(), s, tc.id)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

				return
			}
			repoCall.Unset()
		})
	}
}
