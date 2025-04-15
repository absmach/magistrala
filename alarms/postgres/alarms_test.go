// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/0x6flab/namegenerator"
	"github.com/absmach/magistrala/alarms"
	"github.com/absmach/magistrala/alarms/postgres"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	namegen    = namegenerator.NewGenerator()
	idProvider = uuid.New()
)

func TestCreateAlarm(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM alarms")
		require.Nil(t, err, fmt.Sprintf("clean alarms unexpected error: %s", err))
	})

	repo := postgres.NewAlarmsRepo(db)

	alarm := alarms.Alarm{
		ID:          generateUUID(&testing.T{}),
		RuleID:      generateUUID(&testing.T{}),
		DomainID:    generateUUID(&testing.T{}),
		ChannelID:   generateUUID(&testing.T{}),
		ClientID:    generateUUID(&testing.T{}),
		Subtopic:    namegen.Generate(),
		Measurement: namegen.Generate(),
		Value:       namegen.Generate(),
		Unit:        namegen.Generate(),
		Threshold:   namegen.Generate(),
		Cause:       namegen.Generate(),
		Status:      0,
		AssigneeID:  generateUUID(&testing.T{}),
		CreatedAt:   time.Now().Local(),
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}

	cases := []struct {
		desc  string
		alarm alarms.Alarm
		err   error
	}{
		{
			desc:  "valid alarm",
			alarm: alarm,
			err:   nil,
		},
		{
			desc:  "duplicate alarm",
			alarm: alarm,
			err:   repoerr.ErrConflict,
		},
		{
			desc: "missing rule id",
			alarm: alarms.Alarm{
				ID:          generateUUID(&testing.T{}),
				DomainID:    generateUUID(&testing.T{}),
				ChannelID:   generateUUID(&testing.T{}),
				ClientID:    generateUUID(&testing.T{}),
				Subtopic:    namegen.Generate(),
				Measurement: namegen.Generate(),
				Value:       namegen.Generate(),
				Unit:        namegen.Generate(),
				Threshold:   namegen.Generate(),
				Cause:       namegen.Generate(),
				Status:      0,
				AssigneeID:  generateUUID(&testing.T{}),
				CreatedAt:   time.Now().Local(),

				Metadata: map[string]interface{}{
					"key": "value",
				},
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "invalid alarm",
			alarm: alarms.Alarm{
				ID:          generateUUID(&testing.T{}),
				DomainID:    generateUUID(&testing.T{}),
				ChannelID:   generateUUID(&testing.T{}),
				ClientID:    generateUUID(&testing.T{}),
				Subtopic:    namegen.Generate(),
				Measurement: namegen.Generate(),
				Value:       namegen.Generate(),
				Unit:        namegen.Generate(),
				Threshold:   namegen.Generate(),
				Cause:       namegen.Generate(),
				Status:      0,
				AssigneeID:  generateUUID(&testing.T{}),
				CreatedAt:   time.Now().Local(),

				Metadata: map[string]interface{}{
					"key": make(chan int),
				},
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc:  "empty alarm",
			alarm: alarms.Alarm{},
			err:   repoerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			alarm, err := repo.CreateAlarm(context.Background(), tc.alarm)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			require.NotEmpty(t, alarm.ID)
			require.Equal(t, tc.alarm.RuleID, alarm.RuleID)
			require.Equal(t, tc.alarm.Measurement, alarm.Measurement)
			require.Equal(t, tc.alarm.Value, alarm.Value)
			require.Equal(t, tc.alarm.Unit, alarm.Unit)
			require.Equal(t, tc.alarm.Cause, alarm.Cause)
			require.Equal(t, tc.alarm.Status, alarm.Status)
			require.Equal(t, tc.alarm.DomainID, alarm.DomainID)
			require.Equal(t, tc.alarm.AssigneeID, alarm.AssigneeID)
			require.Equal(t, tc.alarm.Metadata, alarm.Metadata)
		})
	}
}

func TestUpdateAlarm(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM alarms")
		require.Nil(t, err, fmt.Sprintf("clean alarms unexpected error: %s", err))
	})

	repo := postgres.NewAlarmsRepo(db)

	alarm := alarms.Alarm{
		ID:          generateUUID(&testing.T{}),
		RuleID:      generateUUID(&testing.T{}),
		DomainID:    generateUUID(&testing.T{}),
		ChannelID:   generateUUID(&testing.T{}),
		ClientID:    generateUUID(&testing.T{}),
		Measurement: namegen.Generate(),
		Value:       namegen.Generate(),
		Unit:        namegen.Generate(),
		Threshold:   namegen.Generate(),
		Cause:       namegen.Generate(),
		Status:      0,
		AssigneeID:  generateUUID(&testing.T{}),
		CreatedAt:   time.Now().Local(),
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}
	alarm, err := repo.CreateAlarm(context.Background(), alarm)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc  string
		alarm alarms.Alarm
		err   error
	}{
		{
			desc: "valid alarm",
			alarm: alarms.Alarm{
				ID:         alarm.ID,
				Status:     alarms.ActiveStatus,
				DomainID:   alarm.DomainID,
				AssigneeID: generateUUID(&testing.T{}),
				CreatedAt:  alarm.CreatedAt,
				UpdatedAt:  time.Now().Local(),
				UpdatedBy:  generateUUID(&testing.T{}),
				ResolvedAt: time.Now().Local(),
				ResolvedBy: generateUUID(&testing.T{}),
				Metadata: map[string]interface{}{
					"key": "value",
				},
			},
			err: nil,
		},
		{
			desc: "non existing alarm",
			alarm: alarms.Alarm{
				ID: generateUUID(&testing.T{}),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "invalid alarm",
			alarm: alarms.Alarm{
				ID:         alarm.ID,
				RuleID:     generateUUID(&testing.T{}),
				Status:     0,
				DomainID:   generateUUID(&testing.T{}),
				AssigneeID: strings.Repeat("a", 40),
				CreatedAt:  time.Now().Local(),
				Metadata: map[string]interface{}{
					"key": "value",
				},
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc:  "empty alarm",
			alarm: alarms.Alarm{},
			err:   repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			alarm, err := repo.UpdateAlarm(context.Background(), tc.alarm)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			require.NotEmpty(t, alarm.ID)
			require.Equal(t, tc.alarm.Status, alarm.Status)
			require.Equal(t, tc.alarm.DomainID, alarm.DomainID)
			require.Equal(t, tc.alarm.AssigneeID, alarm.AssigneeID)
			require.Equal(t, tc.alarm.Metadata, alarm.Metadata)
		})
	}
}

func TestViewAlarm(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM alarms")
		require.Nil(t, err, fmt.Sprintf("clean alarms unexpected error: %s", err))
	})

	repo := postgres.NewAlarmsRepo(db)

	alarm := alarms.Alarm{
		ID:          generateUUID(&testing.T{}),
		RuleID:      generateUUID(&testing.T{}),
		DomainID:    generateUUID(&testing.T{}),
		ChannelID:   generateUUID(&testing.T{}),
		ClientID:    generateUUID(&testing.T{}),
		Measurement: namegen.Generate(),
		Value:       namegen.Generate(),
		Unit:        namegen.Generate(),
		Threshold:   namegen.Generate(),
		Cause:       namegen.Generate(),
		Status:      0,
		AssigneeID:  generateUUID(&testing.T{}),
		CreatedAt:   time.Now().Local(),
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}
	alarm, err := repo.CreateAlarm(context.Background(), alarm)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc     string
		id       string
		domainID string
		err      error
	}{
		{
			desc:     "valid alarm",
			id:       alarm.ID,
			domainID: alarm.DomainID,
			err:      nil,
		},
		{
			desc:     "non existing alarm id",
			id:       generateUUID(&testing.T{}),
			domainID: alarm.DomainID,
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "non existing domain id",
			id:       alarm.ID,
			domainID: generateUUID(&testing.T{}),
			err:      repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			alarm, err := repo.ViewAlarm(context.Background(), tc.id, tc.domainID)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			require.NotEmpty(t, alarm.ID)
			require.Equal(t, tc.id, alarm.ID)
		})
	}
}

func TestListAlarms(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM alarms")
		require.Nil(t, err, fmt.Sprintf("clean alarms unexpected error: %s", err))
	})
	repo := postgres.NewAlarmsRepo(db)
	items := make([]alarms.Alarm, 1000)
	for i := range 1000 {
		items[i] = alarms.Alarm{
			ID:          generateUUID(&testing.T{}),
			RuleID:      generateUUID(&testing.T{}),
			DomainID:    generateUUID(&testing.T{}),
			ChannelID:   generateUUID(&testing.T{}),
			ClientID:    generateUUID(&testing.T{}),
			Measurement: namegen.Generate(),
			Value:       namegen.Generate(),
			Unit:        namegen.Generate(),
			Threshold:   namegen.Generate(),
			Cause:       namegen.Generate(),
			Status:      0,
			AssigneeID:  generateUUID(&testing.T{}),
			CreatedAt:   time.Now().Local(),
			Metadata: map[string]interface{}{
				"key": "value",
			},
		}
		alarm, err := repo.CreateAlarm(context.Background(), items[i])
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		items[i].ID = alarm.ID
	}

	cases := []struct {
		desc     string
		pm       alarms.PageMetadata
		response []alarms.Alarm
		err      error
	}{
		{
			desc: "valid page",
			pm: alarms.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			response: items[:10],
			err:      nil,
		},
		{
			desc: "offset and limit",
			pm: alarms.PageMetadata{
				Offset: 10,
				Limit:  50,
			},
			response: items[10:60],
			err:      nil,
		},
		{
			desc:     "empty page",
			pm:       alarms.PageMetadata{},
			response: []alarms.Alarm{},
			err:      nil,
		},
		{
			desc: "invalid page",
			pm: alarms.PageMetadata{
				Offset: 1000,
				Limit:  10,
			},
			response: []alarms.Alarm{},
			err:      nil,
		},
		{
			desc: "invalid assignee id",
			pm: alarms.PageMetadata{
				Offset:     0,
				Limit:      10,
				AssigneeID: generateUUID(&testing.T{}),
			},
			response: []alarms.Alarm{},
			err:      nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			alarms, err := repo.ListAlarms(context.Background(), tc.pm)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			require.Equal(t, len(tc.response), len(alarms.Alarms))
		})
	}
}

func TestDeleteAlarm(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM alarms")
		require.Nil(t, err, fmt.Sprintf("clean alarms unexpected error: %s", err))
	})

	repo := postgres.NewAlarmsRepo(db)

	alarm := alarms.Alarm{
		ID:          generateUUID(&testing.T{}),
		RuleID:      generateUUID(&testing.T{}),
		DomainID:    generateUUID(&testing.T{}),
		ChannelID:   generateUUID(&testing.T{}),
		ClientID:    generateUUID(&testing.T{}),
		Measurement: namegen.Generate(),
		Value:       namegen.Generate(),
		Unit:        namegen.Generate(),
		Threshold:   namegen.Generate(),
		Cause:       namegen.Generate(),
		Status:      0,
		AssigneeID:  generateUUID(&testing.T{}),
		CreatedAt:   time.Now().Local(),
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}
	alarm, err := repo.CreateAlarm(context.Background(), alarm)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "valid alarm",
			id:   alarm.ID,
			err:  nil,
		},
		{
			desc: "non existing alarm",
			id:   generateUUID(&testing.T{}),
			err:  repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.DeleteAlarm(context.Background(), tc.id)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		})
	}
}

func generateUUID(t *testing.T) string {
	ulid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	return ulid
}
