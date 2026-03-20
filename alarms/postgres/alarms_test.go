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
	"github.com/absmach/supermq/alarms"
	"github.com/absmach/supermq/alarms/postgres"
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
		ID:          generateUUID(t),
		RuleID:      generateUUID(t),
		DomainID:    generateUUID(t),
		ChannelID:   generateUUID(t),
		ClientID:    generateUUID(t),
		Subtopic:    namegen.Generate(),
		Measurement: namegen.Generate(),
		Value:       namegen.Generate(),
		Unit:        namegen.Generate(),
		Threshold:   namegen.Generate(),
		Cause:       namegen.Generate(),
		Status:      0,
		AssigneeID:  generateUUID(t),
		CreatedAt:   time.Now().UTC(),
		Metadata: map[string]any{
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
			err:   repoerr.ErrNotFound,
		},
		{
			desc: "missing rule id",
			alarm: alarms.Alarm{
				ID:          generateUUID(t),
				DomainID:    generateUUID(t),
				ChannelID:   generateUUID(t),
				ClientID:    generateUUID(t),
				Subtopic:    namegen.Generate(),
				Measurement: namegen.Generate(),
				Value:       namegen.Generate(),
				Unit:        namegen.Generate(),
				Threshold:   namegen.Generate(),
				Cause:       namegen.Generate(),
				Status:      0,
				AssigneeID:  generateUUID(t),
				CreatedAt:   time.Now().UTC(),

				Metadata: map[string]any{
					"key": "value",
				},
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "invalid alarm",
			alarm: alarms.Alarm{
				ID:          generateUUID(t),
				DomainID:    generateUUID(t),
				ChannelID:   generateUUID(t),
				ClientID:    generateUUID(t),
				Subtopic:    namegen.Generate(),
				Measurement: namegen.Generate(),
				Value:       namegen.Generate(),
				Unit:        namegen.Generate(),
				Threshold:   namegen.Generate(),
				Cause:       namegen.Generate(),
				Status:      0,
				AssigneeID:  generateUUID(t),
				CreatedAt:   time.Now().UTC(),

				Metadata: map[string]any{
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
			assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			assert.NotEmpty(t, alarm.ID)
			assert.Equal(t, tc.alarm.RuleID, alarm.RuleID)
			assert.Equal(t, tc.alarm.Measurement, alarm.Measurement)
			assert.Equal(t, tc.alarm.Value, alarm.Value)
			assert.Equal(t, tc.alarm.Unit, alarm.Unit)
			assert.Equal(t, tc.alarm.Cause, alarm.Cause)
			assert.Equal(t, tc.alarm.Status, alarm.Status)
			assert.Equal(t, tc.alarm.DomainID, alarm.DomainID)
			assert.Equal(t, tc.alarm.AssigneeID, alarm.AssigneeID)
			assert.Equal(t, tc.alarm.Metadata, alarm.Metadata)
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
		ID:          generateUUID(t),
		RuleID:      generateUUID(t),
		DomainID:    generateUUID(t),
		ChannelID:   generateUUID(t),
		ClientID:    generateUUID(t),
		Measurement: namegen.Generate(),
		Value:       namegen.Generate(),
		Unit:        namegen.Generate(),
		Threshold:   namegen.Generate(),
		Cause:       namegen.Generate(),
		Status:      0,
		AssigneeID:  generateUUID(t),
		CreatedAt:   time.Now().UTC(),
		Metadata: map[string]any{
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
				ID:             alarm.ID,
				Status:         alarms.ClearedStatus,
				DomainID:       alarm.DomainID,
				AssigneeID:     generateUUID(t),
				AssignedBy:     generateUUID(t),
				AssignedAt:     time.Now().UTC(),
				AcknowledgedBy: generateUUID(t),
				AcknowledgedAt: time.Now().UTC(),
				CreatedAt:      alarm.CreatedAt,
				UpdatedAt:      time.Now().UTC(),
				UpdatedBy:      generateUUID(t),
				ResolvedAt:     time.Now().UTC(),
				ResolvedBy:     generateUUID(t),
				Metadata: map[string]any{
					"key": "value",
				},
			},
			err: nil,
		},
		{
			desc: "non existing alarm",
			alarm: alarms.Alarm{
				ID: generateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "invalid alarm",
			alarm: alarms.Alarm{
				ID:         alarm.ID,
				RuleID:     generateUUID(t),
				Status:     0,
				DomainID:   generateUUID(t),
				AssigneeID: strings.Repeat("a", 40),
				CreatedAt:  time.Now().UTC(),
				Metadata: map[string]any{
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
			assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			assert.NotEmpty(t, alarm.ID)
			assert.Equal(t, tc.alarm.Status, alarm.Status)
			assert.Equal(t, tc.alarm.DomainID, alarm.DomainID)
			assert.Equal(t, tc.alarm.AssigneeID, alarm.AssigneeID)
			assert.Equal(t, tc.alarm.UpdatedBy, alarm.UpdatedBy)
			assert.Equal(t, tc.alarm.ResolvedBy, alarm.ResolvedBy)
			assert.Equal(t, tc.alarm.AcknowledgedBy, alarm.AcknowledgedBy)
			assert.Equal(t, tc.alarm.Metadata, alarm.Metadata)
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
		ID:          generateUUID(t),
		RuleID:      generateUUID(t),
		DomainID:    generateUUID(t),
		ChannelID:   generateUUID(t),
		ClientID:    generateUUID(t),
		Measurement: namegen.Generate(),
		Value:       namegen.Generate(),
		Unit:        namegen.Generate(),
		Threshold:   namegen.Generate(),
		Cause:       namegen.Generate(),
		Status:      0,
		AssigneeID:  generateUUID(t),
		CreatedAt:   time.Now().UTC(),
		Metadata: map[string]any{
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
			id:       generateUUID(t),
			domainID: alarm.DomainID,
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "non existing domain id",
			id:       alarm.ID,
			domainID: generateUUID(t),
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
			assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			assert.NotEmpty(t, alarm.ID)
			assert.Equal(t, tc.id, alarm.ID)
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
			ID:          generateUUID(t),
			RuleID:      generateUUID(t),
			DomainID:    generateUUID(t),
			ChannelID:   generateUUID(t),
			ClientID:    generateUUID(t),
			Measurement: namegen.Generate(),
			Value:       namegen.Generate(),
			Unit:        namegen.Generate(),
			Threshold:   namegen.Generate(),
			Cause:       namegen.Generate(),
			Status:      0,
			AssigneeID:  generateUUID(t),
			CreatedAt:   time.Now().UTC(),
			Metadata: map[string]any{
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
				AssigneeID: generateUUID(t),
			},
			response: []alarms.Alarm{},
			err:      nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			alarms, err := repo.ListAllAlarms(context.Background(), tc.pm)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

				return
			}
			assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			assert.Equal(t, len(tc.response), len(alarms.Alarms))
		})
	}
}

func TestListUserAlarms(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM alarms")
		require.Nil(t, err, fmt.Sprintf("clean alarms unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM rules")
		require.Nil(t, err, fmt.Sprintf("clean rules unexpected error: %s", err))
	})

	repo := postgres.NewAlarmsRepo(db)

	domainID := generateUUID(t)
	userID := generateUUID(t)
	otherUserID := generateUUID(t)
	adminUserID := generateUUID(t)

	// Create 10 rules and 10 alarms referencing them.
	// Assign userID to the first 6 rules via role membership.
	var ruleIDs []string
	var createdAlarms []alarms.Alarm
	for i := range 10 {
		ruleID := generateUUID(t)
		_, err := db.Exec(`INSERT INTO rules (id, name, domain_id, status, logic_type, logic_value) VALUES ($1, $2, $3, 0, 0, '')`,
			ruleID, fmt.Sprintf("rule-%d", i), domainID)
		require.Nil(t, err, fmt.Sprintf("insert rule unexpected error: %s", err))
		ruleIDs = append(ruleIDs, ruleID)

		alarm := alarms.Alarm{
			ID:          generateUUID(t),
			RuleID:      ruleID,
			DomainID:    domainID,
			ChannelID:   generateUUID(t),
			ClientID:    generateUUID(t),
			Measurement: namegen.Generate(),
			Value:       namegen.Generate(),
			Unit:        namegen.Generate(),
			Threshold:   namegen.Generate(),
			Cause:       namegen.Generate(),
			Status:      0,
			AssigneeID:  generateUUID(t),
			CreatedAt:   time.Now().UTC().Add(time.Duration(i) * time.Minute),
		}
		alarm, err = repo.CreateAlarm(context.Background(), alarm)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		createdAlarms = append(createdAlarms, alarm)
	}

	// Assign userID to the first 6 rules via rules_roles + rules_role_members.
	userRoleIDs := make([]string, 6)
	for i := range 6 {
		roleID := generateUUID(t)
		userRoleIDs[i] = roleID
		_, err := db.Exec(`INSERT INTO rules_roles (id, name, entity_id) VALUES ($1, $2, $3)`, roleID, "admin", ruleIDs[i])
		require.Nil(t, err, fmt.Sprintf("insert rules_roles unexpected error: %s", err))
		_, err = db.Exec(`INSERT INTO rules_role_members (role_id, member_id, entity_id) VALUES ($1, $2, $3)`, roleID, userID, ruleIDs[i])
		require.Nil(t, err, fmt.Sprintf("insert rules_role_members unexpected error: %s", err))
	}

	for i := range 10 {
		var roleID string
		if i < 6 {
			roleID = userRoleIDs[i]
		} else {
			roleID = generateUUID(t)
			_, err := db.Exec(`INSERT INTO rules_roles (id, name, entity_id) VALUES ($1, $2, $3)`, roleID, "admin", ruleIDs[i])
			require.Nil(t, err, fmt.Sprintf("insert rules_roles unexpected error: %s", err))
		}
		_, err := db.Exec(`INSERT INTO rules_role_members (role_id, member_id, entity_id) VALUES ($1, $2, $3)`, roleID, adminUserID, ruleIDs[i])
		require.Nil(t, err, fmt.Sprintf("insert rules_role_members unexpected error: %s", err))
	}

	_ = createdAlarms

	cases := []struct {
		desc   string
		userID string
		pm     alarms.PageMetadata
		count  int
		err    error
	}{
		{
			desc:   "list user alarms returns only accessible alarms",
			userID: userID,
			pm: alarms.PageMetadata{
				Offset: 0,
				Limit:  100,
			},
			count: 6,
			err:   nil,
		},
		{
			desc:   "list user alarms with limit",
			userID: userID,
			pm: alarms.PageMetadata{
				Offset: 0,
				Limit:  3,
			},
			count: 3,
			err:   nil,
		},
		{
			desc:   "list user alarms with offset",
			userID: userID,
			pm: alarms.PageMetadata{
				Offset: 4,
				Limit:  100,
			},
			count: 2,
			err:   nil,
		},
		{
			desc:   "list user alarms with domain filter",
			userID: userID,
			pm: alarms.PageMetadata{
				DomainID: domainID,
				Offset:   0,
				Limit:    100,
			},
			count: 6,
			err:   nil,
		},
		{
			desc:   "list user alarms with non-existing domain returns 0",
			userID: userID,
			pm: alarms.PageMetadata{
				DomainID: generateUUID(t),
				Offset:   0,
				Limit:    100,
			},
			count: 0,
			err:   nil,
		},
		{
			desc:   "list alarms for user with no role assignments returns 0",
			userID: otherUserID,
			pm: alarms.PageMetadata{
				Offset: 0,
				Limit:  100,
			},
			count: 0,
			err:   nil,
		},
		{
			desc:   "list alarms for admin user with role on all rules returns all alarms",
			userID: adminUserID,
			pm: alarms.PageMetadata{
				Offset: 0,
				Limit:  100,
			},
			count: 10,
			err:   nil,
		},
		{
			desc:   "list user alarms ordered by created_at ascending",
			userID: userID,
			pm: alarms.PageMetadata{
				Offset: 0,
				Limit:  100,
				Order:  "created_at",
				Dir:    "asc",
			},
			count: 6,
			err:   nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			page, err := repo.ListUserAlarms(context.Background(), tc.userID, tc.pm)
			if tc.err != nil {
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
				return
			}
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
			assert.Equal(t, tc.count, len(page.Alarms), fmt.Sprintf("%s: expected %d alarms, got %d", tc.desc, tc.count, len(page.Alarms)))
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
		ID:          generateUUID(t),
		RuleID:      generateUUID(t),
		DomainID:    generateUUID(t),
		ChannelID:   generateUUID(t),
		ClientID:    generateUUID(t),
		Measurement: namegen.Generate(),
		Value:       namegen.Generate(),
		Unit:        namegen.Generate(),
		Threshold:   namegen.Generate(),
		Cause:       namegen.Generate(),
		Status:      0,
		AssigneeID:  generateUUID(t),
		CreatedAt:   time.Now().UTC(),
		Metadata: map[string]any{
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
			id:   generateUUID(t),
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
			assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		})
	}
}

func generateUUID(t *testing.T) string {
	ulid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	return ulid
}
