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
)

const (
	ascDir         = "asc"
	descDir        = "desc"
	nameOrder      = "name"
	createdAtOrder = "created_at"
	updatedAtOrder = "updated_at"
)

var (
	namegen    = namegenerator.NewGenerator()
	idProvider = uuid.New()
)

func TestAddRule(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM rules")
		assert.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
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
		Outputs: re.Outputs{
			&outputs.Alarm{},
		},
		Status:    re.EnabledStatus,
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		CreatedBy: generateUUID(t),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedBy: generateUUID(t),
		Metadata: map[string]any{
			"key": "value",
		},
	}

	cases := []struct {
		desc string
		rule re.Rule
		resp re.Rule
		err  error
	}{
		{
			desc: "valid rule",
			rule: rule,
			resp: rule,
			err:  nil,
		},
		{
			desc: "duplicate rule",
			rule: rule,
			resp: re.Rule{},
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
					StartDateTime:   time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond),
					Time:            time.Now().UTC().Add(2 * time.Hour).Truncate(time.Microsecond),
					Recurring:       schedule.Daily,
					RecurringPeriod: 1,
				},
				Status:    re.EnabledStatus,
				CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
				CreatedBy: generateUUID(t),
				UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
				UpdatedBy: generateUUID(t),
			},
			resp: re.Rule{},
			err:  nil,
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
				CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
				CreatedBy: generateUUID(t),
				UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
				UpdatedBy: generateUUID(t),
			},
			resp: re.Rule{},
			err:  nil,
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
				CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
				CreatedBy: generateUUID(t),
			},
			resp: re.Rule{},
			err:  errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			addedRule, err := repo.AddRule(context.Background(), tc.rule)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				tc.resp = tc.rule
				tc.resp.ID = addedRule.ID
				if tc.resp.Metadata == nil {
					tc.resp.Metadata = re.Metadata{}
				}
				tc.resp.Outputs = addedRule.Outputs
				assert.Equal(t, tc.resp, addedRule, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, addedRule))
			}
		})
	}
}

func TestViewRule(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM rules")
		assert.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
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
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		CreatedBy: generateUUID(t),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedBy: generateUUID(t),
		Metadata: map[string]any{
			"key": "value",
		},
	}
	rule, err := repo.AddRule(context.Background(), rule)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc string
		id   string
		resp re.Rule
		err  error
	}{
		{
			desc: "valid rule",
			id:   rule.ID,
			resp: rule,
			err:  nil,
		},
		{
			desc: "non existing rule",
			id:   generateUUID(t),
			resp: re.Rule{},
			err:  repoerr.ErrViewEntity,
		},
		{
			desc: "empty id",
			id:   "",
			resp: re.Rule{},
			err:  repoerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			retrievedRule, err := repo.ViewRule(context.Background(), tc.id)
			assert.Equal(t, tc.resp, retrievedRule, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, retrievedRule))
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestUpdateRule(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM rules")
		assert.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
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
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		CreatedBy: generateUUID(t),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedBy: generateUUID(t),
		Metadata: map[string]any{
			"key": "value",
		},
	}
	rule, err := repo.AddRule(context.Background(), rule)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	newInputChannel := generateUUID(t)
	newUpdatedBy := generateUUID(t)

	cases := []struct {
		desc string
		rule re.Rule
		resp re.Rule
		err  error
	}{
		{
			desc: "valid rule update",
			rule: re.Rule{
				ID:           rule.ID,
				Name:         "updated-name",
				InputChannel: newInputChannel,
				InputTopic:   "humidity",
				Logic: re.Script{
					Type:  re.LuaType,
					Value: "return value > 30",
				},
				UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
				UpdatedBy: newUpdatedBy,
				Metadata: map[string]any{
					"updated": "metadata",
				},
			},
			resp: re.Rule{
				ID:           rule.ID,
				Name:         "updated-name",
				DomainID:     rule.DomainID,
				InputChannel: newInputChannel,
				InputTopic:   "humidity",
				Logic: re.Script{
					Type:  re.LuaType,
					Value: "return value > 30",
				},
				Status:    rule.Status,
				CreatedAt: rule.CreatedAt,
				CreatedBy: rule.CreatedBy,
				UpdatedAt: time.Time{},
				UpdatedBy: newUpdatedBy,
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
				UpdatedAt:    time.Now().UTC().Truncate(time.Microsecond),
				UpdatedBy:    generateUUID(t),
			},
			resp: re.Rule{},
			err:  repoerr.ErrNotFound,
		},
		{
			desc: "update with invalid metadata",
			rule: re.Rule{
				ID:           rule.ID,
				InputChannel: generateUUID(t),
				Metadata: map[string]any{
					"key": make(chan int),
				},
				UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
				UpdatedBy: generateUUID(t),
			},
			resp: re.Rule{},
			err:  repoerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			updatedRule, err := repo.UpdateRule(context.Background(), tc.rule)
			if tc.err == nil {
				tc.resp.UpdatedAt = updatedRule.UpdatedAt
			}
			assert.Equal(t, tc.resp, updatedRule, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, updatedRule))
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestUpdateRuleStatus(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM rules")
		assert.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
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
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		CreatedBy: generateUUID(t),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedBy: generateUUID(t),
	}
	rule, err := repo.AddRule(context.Background(), rule)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

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
				UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
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
				UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
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
				UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
				UpdatedBy: generateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			updatedRule, err := repo.UpdateRuleStatus(context.Background(), tc.rule)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.rule.ID, updatedRule.ID, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.rule.ID, updatedRule.ID))
				assert.Equal(t, tc.status, updatedRule.Status, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.status, updatedRule.Status))
				assert.Equal(t, tc.rule.UpdatedBy, updatedRule.UpdatedBy, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.rule.UpdatedBy, updatedRule.UpdatedBy))
			}
		})
	}
}

func TestUpdateRuleTags(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM rules")
		assert.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
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
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		CreatedBy: generateUUID(t),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedBy: generateUUID(t),
	}
	rule, err := repo.AddRule(context.Background(), rule)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

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
				UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
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
				UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
				UpdatedBy: generateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			updatedRule, err := repo.UpdateRuleTags(context.Background(), tc.rule)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.rule.ID, updatedRule.ID, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.rule.ID, updatedRule.ID))
				assert.Equal(t, tc.tags, updatedRule.Tags, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.tags, updatedRule.Tags))
				assert.Equal(t, tc.rule.UpdatedBy, updatedRule.UpdatedBy, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.rule.UpdatedBy, updatedRule.UpdatedBy))
			}
		})
	}
}

func TestUpdateRuleSchedule(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM rules")
		assert.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
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
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		CreatedBy: generateUUID(t),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedBy: generateUUID(t),
	}
	rule, err := repo.AddRule(context.Background(), rule)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	newSchedule := schedule.Schedule{
		StartDateTime:   time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond),
		Time:            time.Now().UTC().Add(2 * time.Hour).Truncate(time.Microsecond),
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
				UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
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
				UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
				UpdatedBy: generateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			updatedRule, err := repo.UpdateRuleSchedule(context.Background(), tc.rule)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.rule.ID, updatedRule.ID, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.rule.ID, updatedRule.ID))
				assert.Equal(t, tc.schedule.Recurring, updatedRule.Schedule.Recurring, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.schedule.Recurring, updatedRule.Schedule.Recurring))
				assert.Equal(t, tc.schedule.RecurringPeriod, updatedRule.Schedule.RecurringPeriod, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.schedule.RecurringPeriod, updatedRule.Schedule.RecurringPeriod))
				assert.Equal(t, tc.rule.UpdatedBy, updatedRule.UpdatedBy, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.rule.UpdatedBy, updatedRule.UpdatedBy))
			}
		})
	}
}

func TestUpdateRuleDue(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM rules")
		assert.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
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
			Time: time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond),
		},
		Status:    re.EnabledStatus,
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		CreatedBy: generateUUID(t),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedBy: generateUUID(t),
	}
	rule, err := repo.AddRule(context.Background(), rule)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	newDue := time.Now().UTC().Add(3 * time.Hour).Truncate(time.Microsecond)

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
			updatedRule, err := repo.UpdateRuleDue(context.Background(), tc.id, tc.due)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.id, updatedRule.ID, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.id, updatedRule.ID))
				assert.True(t, updatedRule.Schedule.Time.Sub(tc.due) < time.Second, fmt.Sprintf("%s: expected due time close to %v got %v\n", tc.desc, tc.due, updatedRule.Schedule.Time))
			}
		})
	}
}

func TestListRules(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM rules")
		assert.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

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
			CreatedAt: time.Now().UTC().Add(time.Duration(i) * time.Minute).Truncate(time.Microsecond),
			CreatedBy: generateUUID(t),
			UpdatedAt: time.Now().UTC().Add(time.Duration(i) * time.Minute).Truncate(time.Microsecond),
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
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
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
		{
			desc: "list ordered by name ascending",
			pm: re.PageMeta{
				Offset: 0,
				Limit:  10,
				Status: re.AllStatus,
				Order:  nameOrder,
				Dir:    ascDir,
			},
			count: 10,
			err:   nil,
		},
		{
			desc: "list ordered by name descending",
			pm: re.PageMeta{
				Offset: 0,
				Limit:  10,
				Status: re.AllStatus,
				Order:  nameOrder,
				Dir:    descDir,
			},
			count: 10,
			err:   nil,
		},
		{
			desc: "list ordered by created_at ascending",
			pm: re.PageMeta{
				Offset: 0,
				Limit:  10,
				Status: re.AllStatus,
				Order:  createdAtOrder,
				Dir:    ascDir,
			},
			count: 10,
			err:   nil,
		},
		{
			desc: "list ordered by created_at descending",
			pm: re.PageMeta{
				Offset: 0,
				Limit:  10,
				Status: re.AllStatus,
				Order:  createdAtOrder,
				Dir:    descDir,
			},
			count: 10,
			err:   nil,
		},
		{
			desc: "list ordered by updated_at ascending",
			pm: re.PageMeta{
				Offset: 0,
				Limit:  10,
				Status: re.AllStatus,
				Order:  updatedAtOrder,
				Dir:    ascDir,
			},
			count: 10,
			err:   nil,
		},
		{
			desc: "list ordered by updated_at descending",
			pm: re.PageMeta{
				Offset: 0,
				Limit:  10,
				Status: re.AllStatus,
				Order:  updatedAtOrder,
				Dir:    descDir,
			},
			count: 10,
			err:   nil,
		},
		{
			desc: "list with default order (updated_at desc)",
			pm: re.PageMeta{
				Offset: 0,
				Limit:  10,
				Status: re.AllStatus,
			},
			count: 10,
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
			assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			assert.Equal(t, tc.count, len(page.Rules), fmt.Sprintf("%s: expected %d rules, got %d", tc.desc, tc.count, len(page.Rules)))
			if len(page.Rules) > 1 {
				switch tc.pm.Order {
				case nameOrder:
					switch tc.pm.Dir {
					case ascDir:
						assert.True(t, page.Rules[0].Name <= page.Rules[1].Name, "Expected ascending name order")
					case descDir:
						assert.True(t, page.Rules[0].Name >= page.Rules[1].Name, "Expected descending name order")
					}
				case createdAtOrder:
					switch tc.pm.Dir {
					case ascDir:
						assert.True(t, !page.Rules[0].CreatedAt.After(page.Rules[1].CreatedAt), "Expected ascending created_at order")
					case descDir:
						assert.True(t, !page.Rules[0].CreatedAt.Before(page.Rules[1].CreatedAt), "Expected descending created_at order")
					}
				case updatedAtOrder:
					switch tc.pm.Dir {
					case ascDir:
						assert.True(t, !page.Rules[0].UpdatedAt.After(page.Rules[1].UpdatedAt), "Expected ascending updated_at order")
					case descDir:
						assert.True(t, !page.Rules[0].UpdatedAt.Before(page.Rules[1].UpdatedAt), "Expected descending updated_at order")
					}
				}
			}
		})
	}
}

func TestRemoveRule(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM rules")
		assert.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
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
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		CreatedBy: generateUUID(t),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedBy: generateUUID(t),
	}
	rule, err := repo.AddRule(context.Background(), rule)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

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
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func generateUUID(t *testing.T) string {
	ulid, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	return ulid
}
