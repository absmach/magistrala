// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/0x6flab/namegenerator"
	"github.com/absmach/magistrala/pkg/schedule"
	"github.com/absmach/magistrala/re"
	"github.com/absmach/magistrala/re/outputs"
	"github.com/absmach/magistrala/re/postgres"
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

func TestAddRule(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM rules")
		require.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	rule := re.Rule{
		ID:           generateUUID(t),
		Name:         namegen.Generate(),
		DomainID:     generateUUID(t),
		Tags:         []string{"test", "rule"},
		InputChannel: generateUUID(t),
		InputTopic:   "temperature",
		Logic: re.Script{
			Type:  re.LuaType,
			Value: "return true",
		},
		Status:    re.EnabledStatus,
		CreatedAt: time.Now().UTC(),
		CreatedBy: generateUUID(t),
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: generateUUID(t),
		Metadata: map[string]any{
			"key": "value",
		},
	}

	cases := []struct {
		desc string
		rule re.Rule
		err  error
	}{
		{
			desc: "valid rule",
			rule: rule,
			err:  nil,
		},
		{
			desc: "duplicate rule",
			rule: rule,
			err:  repoerr.ErrConflict,
		},
		{
			desc: "rule with schedule",
			rule: re.Rule{
				ID:           generateUUID(t),
				Name:         namegen.Generate(),
				DomainID:     generateUUID(t),
				InputChannel: generateUUID(t),
				InputTopic:   "humidity",
				Logic: re.Script{
					Type:  re.LuaType,
					Value: "return value > 50",
				},
				Schedule: schedule.Schedule{
					StartDateTime:   time.Now().UTC().Add(time.Hour),
					Time:            time.Now().UTC().Add(2 * time.Hour),
					Recurring:       schedule.Daily,
					RecurringPeriod: 1,
				},
				Status:    re.EnabledStatus,
				CreatedAt: time.Now().UTC(),
				CreatedBy: generateUUID(t),
				UpdatedAt: time.Now().UTC(),
				UpdatedBy: generateUUID(t),
			},
			err: nil,
		},
		{
			desc: "rule with outputs",
			rule: re.Rule{
				ID:           generateUUID(t),
				Name:         namegen.Generate(),
				DomainID:     generateUUID(t),
				InputChannel: generateUUID(t),
				Logic: re.Script{
					Type:  re.GoType,
					Value: "func() bool { return true }",
				},
				Outputs: re.Outputs{
					&outputs.Alarm{
						RuleID: generateUUID(t),
					},
				},
				Status:    re.EnabledStatus,
				CreatedAt: time.Now().UTC(),
				CreatedBy: generateUUID(t),
				UpdatedAt: time.Now().UTC(),
				UpdatedBy: generateUUID(t),
			},
			err: nil,
		},
		{
			desc: "invalid metadata",
			rule: re.Rule{
				ID:           generateUUID(t),
				Name:         namegen.Generate(),
				DomainID:     generateUUID(t),
				InputChannel: generateUUID(t),
				Logic: re.Script{
					Type:  re.LuaType,
					Value: "return true",
				},
				Metadata: map[string]any{
					"key": make(chan int),
				},
				Status:    re.EnabledStatus,
				CreatedAt: time.Now().UTC(),
				CreatedBy: generateUUID(t),
			},
			err: errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			rule, err := repo.AddRule(context.Background(), tc.rule)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			require.NotEmpty(t, rule.ID)
			require.Equal(t, tc.rule.Name, rule.Name)
			require.Equal(t, tc.rule.DomainID, rule.DomainID)
			require.Equal(t, tc.rule.InputChannel, rule.InputChannel)
			require.Equal(t, tc.rule.Logic.Type, rule.Logic.Type)
			require.Equal(t, tc.rule.Logic.Value, rule.Logic.Value)
			require.Equal(t, tc.rule.Status, rule.Status)
		})
	}
}

func TestViewRule(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM rules")
		require.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	rule := re.Rule{
		ID:           generateUUID(t),
		Name:         namegen.Generate(),
		DomainID:     generateUUID(t),
		InputChannel: generateUUID(t),
		InputTopic:   "temperature",
		Logic: re.Script{
			Type:  re.LuaType,
			Value: "return true",
		},
		Status:    re.EnabledStatus,
		CreatedAt: time.Now().UTC(),
		CreatedBy: generateUUID(t),
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: generateUUID(t),
		Metadata: map[string]any{
			"key": "value",
		},
	}
	rule, err := repo.AddRule(context.Background(), rule)
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
			id:   generateUUID(t),
			err:  repoerr.ErrViewEntity,
		},
		{
			desc: "empty id",
			id:   "",
			err:  repoerr.ErrViewEntity,
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

func TestUpdateRule(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM rules")
		require.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	rule := re.Rule{
		ID:           generateUUID(t),
		Name:         namegen.Generate(),
		DomainID:     generateUUID(t),
		InputChannel: generateUUID(t),
		InputTopic:   "temperature",
		Logic: re.Script{
			Type:  re.LuaType,
			Value: "return true",
		},
		Status:    re.EnabledStatus,
		CreatedAt: time.Now().UTC(),
		CreatedBy: generateUUID(t),
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: generateUUID(t),
		Metadata: map[string]any{
			"key": "value",
		},
	}
	rule, err := repo.AddRule(context.Background(), rule)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc string
		rule re.Rule
		err  error
	}{
		{
			desc: "valid rule update",
			rule: re.Rule{
				ID:           rule.ID,
				Name:         "updated-name",
				InputChannel: generateUUID(t),
				InputTopic:   "humidity",
				Logic: re.Script{
					Type:  re.LuaType,
					Value: "return value > 30",
				},
				UpdatedAt: time.Now().UTC(),
				UpdatedBy: generateUUID(t),
				Metadata: map[string]any{
					"updated": "metadata",
				},
			},
			err: nil,
		},
		{
			desc: "update non-existing rule",
			rule: re.Rule{
				ID:           generateUUID(t),
				Name:         namegen.Generate(),
				InputChannel: generateUUID(t),
				UpdatedAt:    time.Now().UTC(),
				UpdatedBy:    generateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "update with invalid metadata",
			rule: re.Rule{
				ID:           rule.ID,
				InputChannel: generateUUID(t),
				Metadata: map[string]any{
					"key": make(chan int),
				},
				UpdatedAt: time.Now().UTC(),
				UpdatedBy: generateUUID(t),
			},
			err: repoerr.ErrUpdateEntity,
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
			require.Equal(t, tc.rule.ID, rule.ID)
			if tc.rule.Name != "" {
				require.Equal(t, tc.rule.Name, rule.Name)
			}
		})
	}
}

func TestUpdateRuleStatus(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM rules")
		require.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	rule := re.Rule{
		ID:           generateUUID(t),
		Name:         namegen.Generate(),
		DomainID:     generateUUID(t),
		InputChannel: generateUUID(t),
		Logic: re.Script{
			Type:  re.LuaType,
			Value: "return true",
		},
		Status:    re.EnabledStatus,
		CreatedAt: time.Now().UTC(),
		CreatedBy: generateUUID(t),
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: generateUUID(t),
	}
	rule, err := repo.AddRule(context.Background(), rule)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc   string
		rule   re.Rule
		status re.Status
		err    error
	}{
		{
			desc: "disable rule",
			rule: re.Rule{
				ID:        rule.ID,
				Status:    re.DisabledStatus,
				UpdatedAt: time.Now().UTC(),
				UpdatedBy: generateUUID(t),
			},
			status: re.DisabledStatus,
			err:    nil,
		},
		{
			desc: "enable rule",
			rule: re.Rule{
				ID:        rule.ID,
				Status:    re.EnabledStatus,
				UpdatedAt: time.Now().UTC(),
				UpdatedBy: generateUUID(t),
			},
			status: re.EnabledStatus,
			err:    nil,
		},
		{
			desc: "update non-existing rule status",
			rule: re.Rule{
				ID:        generateUUID(t),
				Status:    re.DisabledStatus,
				UpdatedAt: time.Now().UTC(),
				UpdatedBy: generateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			rule, err := repo.UpdateRuleStatus(context.Background(), tc.rule)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			require.NotEmpty(t, rule.ID)
			require.Equal(t, tc.status, rule.Status)
		})
	}
}

func TestUpdateRuleTags(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM rules")
		require.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	rule := re.Rule{
		ID:           generateUUID(t),
		Name:         namegen.Generate(),
		DomainID:     generateUUID(t),
		InputChannel: generateUUID(t),
		Tags:         []string{"tag1", "tag2"},
		Logic: re.Script{
			Type:  re.LuaType,
			Value: "return true",
		},
		Status:    re.EnabledStatus,
		CreatedAt: time.Now().UTC(),
		CreatedBy: generateUUID(t),
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: generateUUID(t),
	}
	rule, err := repo.AddRule(context.Background(), rule)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc string
		rule re.Rule
		tags []string
		err  error
	}{
		{
			desc: "update tags",
			rule: re.Rule{
				ID:        rule.ID,
				Tags:      []string{"newtag1", "newtag2", "newtag3"},
				UpdatedAt: time.Now().UTC(),
				UpdatedBy: generateUUID(t),
			},
			tags: []string{"newtag1", "newtag2", "newtag3"},
			err:  nil,
		},
		{
			desc: "update non-existing rule tags",
			rule: re.Rule{
				ID:        generateUUID(t),
				Tags:      []string{"tag"},
				UpdatedAt: time.Now().UTC(),
				UpdatedBy: generateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			rule, err := repo.UpdateRuleTags(context.Background(), tc.rule)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			require.NotEmpty(t, rule.ID)
			require.Equal(t, tc.tags, rule.Tags)
		})
	}
}

func TestUpdateRuleSchedule(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM rules")
		require.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	rule := re.Rule{
		ID:           generateUUID(t),
		Name:         namegen.Generate(),
		DomainID:     generateUUID(t),
		InputChannel: generateUUID(t),
		Logic: re.Script{
			Type:  re.LuaType,
			Value: "return true",
		},
		Status:    re.EnabledStatus,
		CreatedAt: time.Now().UTC(),
		CreatedBy: generateUUID(t),
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: generateUUID(t),
	}
	rule, err := repo.AddRule(context.Background(), rule)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	newSchedule := schedule.Schedule{
		StartDateTime:   time.Now().UTC().Add(time.Hour),
		Time:            time.Now().UTC().Add(2 * time.Hour),
		Recurring:       schedule.Weekly,
		RecurringPeriod: 2,
	}

	cases := []struct {
		desc     string
		rule     re.Rule
		schedule schedule.Schedule
		err      error
	}{
		{
			desc: "update schedule",
			rule: re.Rule{
				ID:        rule.ID,
				Schedule:  newSchedule,
				UpdatedAt: time.Now().UTC(),
				UpdatedBy: generateUUID(t),
			},
			schedule: newSchedule,
			err:      nil,
		},
		{
			desc: "update non-existing rule schedule",
			rule: re.Rule{
				ID:        generateUUID(t),
				Schedule:  newSchedule,
				UpdatedAt: time.Now().UTC(),
				UpdatedBy: generateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			rule, err := repo.UpdateRuleSchedule(context.Background(), tc.rule)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			require.NotEmpty(t, rule.ID)
			require.Equal(t, tc.schedule.Recurring, rule.Schedule.Recurring)
			require.Equal(t, tc.schedule.RecurringPeriod, rule.Schedule.RecurringPeriod)
		})
	}
}

func TestUpdateRuleDue(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM rules")
		require.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	rule := re.Rule{
		ID:           generateUUID(t),
		Name:         namegen.Generate(),
		DomainID:     generateUUID(t),
		InputChannel: generateUUID(t),
		Logic: re.Script{
			Type:  re.LuaType,
			Value: "return true",
		},
		Schedule: schedule.Schedule{
			Time: time.Now().UTC().Add(time.Hour),
		},
		Status:    re.EnabledStatus,
		CreatedAt: time.Now().UTC(),
		CreatedBy: generateUUID(t),
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: generateUUID(t),
	}
	rule, err := repo.AddRule(context.Background(), rule)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	newDue := time.Now().UTC().Add(3 * time.Hour)

	cases := []struct {
		desc string
		id   string
		due  time.Time
		err  error
	}{
		{
			desc: "update due time",
			id:   rule.ID,
			due:  newDue,
			err:  nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			rule, err := repo.UpdateRuleDue(context.Background(), tc.id, tc.due)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			require.NotEmpty(t, rule.ID)
			require.True(t, rule.Schedule.Time.Sub(tc.due) < time.Second)
		})
	}
}

func TestListRules(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM rules")
		require.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	// Create test data
	domainID := generateUUID(t)
	channelID := generateUUID(t)
	items := make([]re.Rule, 100)

	for i := range 100 {
		items[i] = re.Rule{
			ID:           generateUUID(t),
			Name:         namegen.Generate(),
			DomainID:     domainID,
			InputChannel: channelID,
			Tags:         []string{fmt.Sprintf("tag%d", i%10)},
			Logic: re.Script{
				Type:  re.LuaType,
				Value: "return true",
			},
			Status:    re.EnabledStatus,
			CreatedAt: time.Now().UTC().Add(time.Duration(i) * time.Minute),
			CreatedBy: generateUUID(t),
			UpdatedAt: time.Now().UTC().Add(time.Duration(i) * time.Minute),
			UpdatedBy: generateUUID(t),
		}
		if i%2 == 0 {
			items[i].Status = re.DisabledStatus
		}
		if i%3 == 0 {
			items[i].Schedule = schedule.Schedule{
				Time:      time.Now().UTC().Add(time.Duration(i) * time.Hour),
				Recurring: schedule.Daily,
			}
		}
		rule, err := repo.AddRule(context.Background(), items[i])
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		items[i].ID = rule.ID
	}

	cases := []struct {
		desc  string
		pm    re.PageMeta
		count int
		err   error
	}{
		{
			desc: "list first page",
			pm: re.PageMeta{
				Offset: 0,
				Limit:  10,
				Status: re.AllStatus,
			},
			count: 10,
			err:   nil,
		},
		{
			desc: "list with offset",
			pm: re.PageMeta{
				Offset: 10,
				Limit:  20,
				Status: re.AllStatus,
			},
			count: 20,
			err:   nil,
		},
		{
			desc: "list by domain",
			pm: re.PageMeta{
				Domain: domainID,
				Offset: 0,
				Limit:  200,
				Status: re.AllStatus,
			},
			count: 100,
			err:   nil,
		},
		{
			desc: "list by channel",
			pm: re.PageMeta{
				InputChannel: channelID,
				Offset:       0,
				Limit:        200,
				Status:       re.AllStatus,
			},
			count: 100,
			err:   nil,
		},
		{
			desc: "list enabled rules",
			pm: re.PageMeta{
				Status: re.EnabledStatus,
				Offset: 0,
				Limit:  200,
			},
			count: 50,
			err:   nil,
		},
		{
			desc: "list disabled rules",
			pm: re.PageMeta{
				Status: re.DisabledStatus,
				Offset: 0,
				Limit:  200,
			},
			count: 50,
			err:   nil,
		},
		{
			desc: "list by tag",
			pm: re.PageMeta{
				Tag:    "tag1",
				Offset: 0,
				Limit:  200,
				Status: re.AllStatus,
			},
			count: 10,
			err:   nil,
		},
		{
			desc: "list with zero limit returns all",
			pm: re.PageMeta{
				Status: re.AllStatus,
			},
			count: 100,
			err:   nil,
		},
		{
			desc: "list non-existing domain",
			pm: re.PageMeta{
				Domain: generateUUID(t),
				Offset: 0,
				Limit:  10,
				Status: re.AllStatus,
			},
			count: 0,
			err:   nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			page, err := repo.ListRules(context.Background(), tc.pm)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			require.Equal(t, tc.count, len(page.Rules), fmt.Sprintf("%s: expected %d rules, got %d", tc.desc, tc.count, len(page.Rules)))
		})
	}
}

func TestRemoveRule(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM rules")
		require.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	rule := re.Rule{
		ID:           generateUUID(t),
		Name:         namegen.Generate(),
		DomainID:     generateUUID(t),
		InputChannel: generateUUID(t),
		Logic: re.Script{
			Type:  re.LuaType,
			Value: "return true",
		},
		Status:    re.EnabledStatus,
		CreatedAt: time.Now().UTC(),
		CreatedBy: generateUUID(t),
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: generateUUID(t),
	}
	rule, err := repo.AddRule(context.Background(), rule)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "remove existing rule",
			id:   rule.ID,
			err:  nil,
		},
		{
			desc: "remove non-existing rule",
			id:   generateUUID(t),
			err:  repoerr.ErrNotFound,
		},
		{
			desc: "remove already removed rule",
			id:   rule.ID,
			err:  repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.RemoveRule(context.Background(), tc.id)
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
