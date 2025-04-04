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

func TestCreateRule(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM rules")
		require.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
	})

	repo := postgres.NewAlarmsRepo(db)

	alarm := alarms.Rule{
		ID:        generateUUID(&testing.T{}),
		Name:      namegen.Generate(),
		UserID:    generateUUID(&testing.T{}),
		DomainID:  generateUUID(&testing.T{}),
		Condition: "temperature > 30",
		Channel:   generateUUID(&testing.T{}),
		CreatedAt: time.Now().Local(),
		CreatedBy: generateUUID(&testing.T{}),
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}

	cases := []struct {
		desc string
		rule alarms.Rule
		err  error
	}{
		{
			desc: "valid rule",
			rule: alarm,
			err:  nil,
		},
		{
			desc: "duplicate rule",
			rule: alarm,
			err:  repoerr.ErrConflict,
		},
		{
			desc: "missing name",
			rule: alarms.Rule{
				ID:        generateUUID(&testing.T{}),
				UserID:    generateUUID(&testing.T{}),
				DomainID:  generateUUID(&testing.T{}),
				Condition: "temperature > 30",
				Channel:   generateUUID(&testing.T{}),
				CreatedAt: time.Now().Local(),
				CreatedBy: generateUUID(&testing.T{}),
				Metadata: map[string]interface{}{
					"key": "value",
				},
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "invalid rule",
			rule: alarms.Rule{
				ID:        generateUUID(&testing.T{}),
				Name:      strings.Repeat("a", 255),
				UserID:    generateUUID(&testing.T{}),
				DomainID:  generateUUID(&testing.T{}),
				Condition: "temperature > 30",
				Channel:   generateUUID(&testing.T{}),
				CreatedAt: time.Now().Local(),
				CreatedBy: generateUUID(&testing.T{}),
				Metadata: map[string]interface{}{
					"key": "value",
				},
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "empty rule",
			rule: alarms.Rule{},
			err:  repoerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			rule, err := repo.CreateRule(context.Background(), tc.rule)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			require.NotEmpty(t, rule.ID)
			require.Equal(t, tc.rule.Name, rule.Name)
			require.Equal(t, tc.rule.UserID, rule.UserID)
			require.Equal(t, tc.rule.DomainID, rule.DomainID)
			require.Equal(t, tc.rule.Condition, rule.Condition)
			require.Equal(t, tc.rule.Channel, rule.Channel)
			require.Equal(t, tc.rule.CreatedBy, rule.CreatedBy)
			require.Equal(t, tc.rule.Metadata, rule.Metadata)
		})
	}
}

func TestUpdateRule(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM rules")
		require.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
	})

	repo := postgres.NewAlarmsRepo(db)

	rule := alarms.Rule{
		ID:        generateUUID(&testing.T{}),
		Name:      namegen.Generate(),
		UserID:    generateUUID(&testing.T{}),
		DomainID:  generateUUID(&testing.T{}),
		Condition: "temperature > 30",
		Channel:   generateUUID(&testing.T{}),
		CreatedAt: time.Now().Local(),
		CreatedBy: generateUUID(&testing.T{}),
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}
	rule, err := repo.CreateRule(context.Background(), rule)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc string
		rule alarms.Rule
		err  error
	}{
		{
			desc: "valid rule",
			rule: alarms.Rule{
				ID:   rule.ID,
				Name: namegen.Generate(),
			},
			err: nil,
		},
		{
			desc: "non existing rule",
			rule: alarms.Rule{
				ID:   generateUUID(&testing.T{}),
				Name: namegen.Generate(),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "invalid rule",
			rule: alarms.Rule{
				ID:   rule.ID,
				Name: strings.Repeat("a", 255),
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "empty rule",
			rule: alarms.Rule{},
			err:  repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			rule, err := repo.UpdateRule(context.Background(), tc.rule)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			require.NotEmpty(t, rule.ID)
			require.Equal(t, tc.rule.Name, rule.Name)
		})
	}
}

func TestViewRule(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM rules")
		require.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
	})

	repo := postgres.NewAlarmsRepo(db)

	rule := alarms.Rule{
		ID:        generateUUID(&testing.T{}),
		Name:      namegen.Generate(),
		UserID:    generateUUID(&testing.T{}),
		DomainID:  generateUUID(&testing.T{}),
		Condition: "temperature > 30",
		Channel:   generateUUID(&testing.T{}),
		CreatedAt: time.Now().Local(),
		CreatedBy: generateUUID(&testing.T{}),
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}
	rule, err := repo.CreateRule(context.Background(), rule)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "valid rule",
			id:   rule.ID,
			err:  nil,
		},
		{
			desc: "non existing rule",
			id:   generateUUID(&testing.T{}),
			err:  repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			rule, err := repo.ViewRule(context.Background(), tc.id)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			require.NotEmpty(t, rule.ID)
			require.Equal(t, tc.id, rule.ID)
		})
	}
}

func TestListRules(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM rules")
		require.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
	})

	repo := postgres.NewAlarmsRepo(db)
	items := make([]alarms.Rule, 1000)
	for i := 0; i < 1000; i++ {
		rule := alarms.Rule{
			ID:        generateUUID(&testing.T{}),
			Name:      namegen.Generate(),
			UserID:    generateUUID(&testing.T{}),
			DomainID:  generateUUID(&testing.T{}),
			Condition: "temperature > 30",
			Channel:   generateUUID(&testing.T{}),
			CreatedAt: time.Now().Local(),
			CreatedBy: generateUUID(&testing.T{}),
			Metadata: map[string]interface{}{
				"key": "value",
			},
		}
		rule, err := repo.CreateRule(context.Background(), rule)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		items[i] = rule
	}

	cases := []struct {
		desc     string
		pm       alarms.PageMetadata
		response []alarms.Rule
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
			response: []alarms.Rule{},
			err:      nil,
		},
		{
			desc: "invalid page",
			pm: alarms.PageMetadata{
				Offset: 1000,
				Limit:  10,
			},
			response: []alarms.Rule{},
			err:      nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			rules, err := repo.ListRules(context.Background(), tc.pm)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			require.Equal(t, len(tc.response), len(rules.Rules))
		})
	}
}

func TestDeleteRule(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM rules")
		require.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
	})

	repo := postgres.NewAlarmsRepo(db)

	rule := alarms.Rule{
		ID:        generateUUID(&testing.T{}),
		Name:      namegen.Generate(),
		UserID:    generateUUID(&testing.T{}),
		DomainID:  generateUUID(&testing.T{}),
		Condition: "temperature > 30",
		Channel:   generateUUID(&testing.T{}),
		CreatedAt: time.Now().Local(),
		CreatedBy: generateUUID(&testing.T{}),
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}
	rule, err := repo.CreateRule(context.Background(), rule)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "valid rule",
			id:   rule.ID,
			err:  nil,
		},
		{
			desc: "non existing rule",
			id:   generateUUID(&testing.T{}),
			err:  repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.DeleteRule(context.Background(), tc.id)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		})
	}
}

func TestCreateAlarm(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM alarms")
		require.Nil(t, err, fmt.Sprintf("clean alarms unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM rules")
		require.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
	})

	repo := postgres.NewAlarmsRepo(db)

	rule := alarms.Rule{
		ID:        generateUUID(&testing.T{}),
		Name:      namegen.Generate(),
		UserID:    generateUUID(&testing.T{}),
		DomainID:  generateUUID(&testing.T{}),
		Condition: "temperature > 30",
		Channel:   generateUUID(&testing.T{}),
		CreatedAt: time.Now().Local(),
		CreatedBy: generateUUID(&testing.T{}),
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}
	rule, err := repo.CreateRule(context.Background(), rule)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	alarm := alarms.Alarm{
		ID:         generateUUID(&testing.T{}),
		RuleID:     rule.ID,
		Message:    namegen.Generate(),
		Status:     0,
		UserID:     generateUUID(&testing.T{}),
		DomainID:   generateUUID(&testing.T{}),
		AssigneeID: generateUUID(&testing.T{}),
		CreatedAt:  time.Now().Local(),
		CreatedBy:  generateUUID(&testing.T{}),
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
				ID:         generateUUID(&testing.T{}),
				Message:    namegen.Generate(),
				Status:     0,
				UserID:     generateUUID(&testing.T{}),
				DomainID:   generateUUID(&testing.T{}),
				AssigneeID: generateUUID(&testing.T{}),
				CreatedAt:  time.Now().Local(),
				CreatedBy:  generateUUID(&testing.T{}),
				Metadata: map[string]interface{}{
					"key": "value",
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
			require.Equal(t, tc.alarm.Message, alarm.Message)
			require.Equal(t, tc.alarm.Status, alarm.Status)
			require.Equal(t, tc.alarm.UserID, alarm.UserID)
			require.Equal(t, tc.alarm.DomainID, alarm.DomainID)
			require.Equal(t, tc.alarm.AssigneeID, alarm.AssigneeID)
			require.Equal(t, tc.alarm.CreatedBy, alarm.CreatedBy)
			require.Equal(t, tc.alarm.Metadata, alarm.Metadata)
		})
	}
}

func TestUpdateAlarm(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM alarms")
		require.Nil(t, err, fmt.Sprintf("clean alarms unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM rules")
		require.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
	})

	repo := postgres.NewAlarmsRepo(db)

	rule := alarms.Rule{
		ID:        generateUUID(&testing.T{}),
		Name:      namegen.Generate(),
		UserID:    generateUUID(&testing.T{}),
		DomainID:  generateUUID(&testing.T{}),
		Condition: "temperature > 30",
		Channel:   generateUUID(&testing.T{}),
		CreatedAt: time.Now().Local(),
		CreatedBy: generateUUID(&testing.T{}),
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}
	rule, err := repo.CreateRule(context.Background(), rule)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	alarm := alarms.Alarm{
		ID:         generateUUID(&testing.T{}),
		RuleID:     rule.ID,
		Message:    namegen.Generate(),
		Status:     0,
		UserID:     generateUUID(&testing.T{}),
		DomainID:   generateUUID(&testing.T{}),
		AssigneeID: generateUUID(&testing.T{}),
		CreatedAt:  time.Now().Local(),
		CreatedBy:  generateUUID(&testing.T{}),
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}
	alarm, err = repo.CreateAlarm(context.Background(), alarm)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

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
				RuleID:     rule.ID,
				Message:    strings.Repeat("a", 255),
				Status:     0,
				UserID:     strings.Repeat("a", 40),
				DomainID:   generateUUID(&testing.T{}),
				AssigneeID: generateUUID(&testing.T{}),
				CreatedAt:  time.Now().Local(),
				CreatedBy:  generateUUID(&testing.T{}),
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
			require.Equal(t, tc.alarm.RuleID, alarm.RuleID)
			require.Equal(t, tc.alarm.Message, alarm.Message)
			require.Equal(t, tc.alarm.Status, alarm.Status)
			require.Equal(t, tc.alarm.UserID, alarm.UserID)
			require.Equal(t, tc.alarm.DomainID, alarm.DomainID)
			require.Equal(t, tc.alarm.AssigneeID, alarm.AssigneeID)
			require.Equal(t, tc.alarm.CreatedBy, alarm.CreatedBy)
			require.Equal(t, tc.alarm.Metadata, alarm.Metadata)
		})
	}
}

func TestViewAlarm(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM alarms")
		require.Nil(t, err, fmt.Sprintf("clean alarms unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM rules")
		require.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
	})

	repo := postgres.NewAlarmsRepo(db)

	rule := alarms.Rule{
		ID:        generateUUID(&testing.T{}),
		Name:      namegen.Generate(),
		UserID:    generateUUID(&testing.T{}),
		DomainID:  generateUUID(&testing.T{}),
		Condition: "temperature > 30",
		Channel:   generateUUID(&testing.T{}),
		CreatedAt: time.Now().Local(),
		CreatedBy: generateUUID(&testing.T{}),
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}
	rule, err := repo.CreateRule(context.Background(), rule)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	alarm := alarms.Alarm{
		ID:         generateUUID(&testing.T{}),
		RuleID:     rule.ID,
		Message:    namegen.Generate(),
		Status:     0,
		UserID:     generateUUID(&testing.T{}),
		DomainID:   generateUUID(&testing.T{}),
		AssigneeID: generateUUID(&testing.T{}),
		CreatedAt:  time.Now().Local(),
		CreatedBy:  generateUUID(&testing.T{}),
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}
	alarm, err = repo.CreateAlarm(context.Background(), alarm)
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
			alarm, err := repo.ViewAlarm(context.Background(), tc.id)
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
		_, err = db.Exec("DELETE FROM rules")
		require.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
	})
	repo := postgres.NewAlarmsRepo(db)
	items := make([]alarms.Alarm, 1000)
	for i := 0; i < 1000; i++ {
		rule := alarms.Rule{
			ID:        generateUUID(&testing.T{}),
			Name:      namegen.Generate(),
			UserID:    generateUUID(&testing.T{}),
			DomainID:  generateUUID(&testing.T{}),
			Condition: "temperature > 30",
			Channel:   generateUUID(&testing.T{}),
			CreatedAt: time.Now().Local(),
			CreatedBy: generateUUID(&testing.T{}),
			Metadata: map[string]interface{}{
				"key": "value",
			},
		}
		rule, err := repo.CreateRule(context.Background(), rule)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		items[i] = alarms.Alarm{
			ID:         generateUUID(&testing.T{}),
			RuleID:     rule.ID,
			Message:    namegen.Generate(),
			Status:     0,
			UserID:     generateUUID(&testing.T{}),
			DomainID:   generateUUID(&testing.T{}),
			AssigneeID: generateUUID(&testing.T{}),
			CreatedAt:  time.Now().Local(),
			CreatedBy:  generateUUID(&testing.T{}),
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
		_, err = db.Exec("DELETE FROM rules")
		require.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
	})

	repo := postgres.NewAlarmsRepo(db)

	rule := alarms.Rule{
		ID:        generateUUID(&testing.T{}),
		Name:      namegen.Generate(),
		UserID:    generateUUID(&testing.T{}),
		DomainID:  generateUUID(&testing.T{}),
		Condition: "temperature > 30",
		Channel:   generateUUID(&testing.T{}),
		CreatedAt: time.Now().Local(),
		CreatedBy: generateUUID(&testing.T{}),
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}
	rule, err := repo.CreateRule(context.Background(), rule)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	alarm := alarms.Alarm{
		ID:         generateUUID(&testing.T{}),
		RuleID:     rule.ID,
		Message:    namegen.Generate(),
		Status:     0,
		UserID:     generateUUID(&testing.T{}),
		DomainID:   generateUUID(&testing.T{}),
		AssigneeID: generateUUID(&testing.T{}),
		CreatedAt:  time.Now().Local(),
		CreatedBy:  generateUUID(&testing.T{}),
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}
	alarm, err = repo.CreateAlarm(context.Background(), alarm)
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
