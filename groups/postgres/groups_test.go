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
	"github.com/absmach/supermq/groups"
	"github.com/absmach/supermq/groups/postgres"
	"github.com/absmach/supermq/internal/nullable"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/roles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	namegen            = namegenerator.NewGenerator()
	invalidID          = strings.Repeat("a", 37)
	validTimestamp     = time.Now().UTC().Truncate(time.Millisecond)
	description        = strings.Repeat("a", 64)
	desc               = nullable.New(description)
	invalidDescription = strings.Repeat("a", 1025)
	invalidDesc        = nullable.New(invalidDescription)

	validGroup = groups.Group{
		ID:          testsutil.GenerateUUID(&testing.T{}),
		Domain:      testsutil.GenerateUUID(&testing.T{}),
		Name:        namegen.Generate(),
		Tags:        []string{"tag1", "tag2"},
		Description: desc,
		Metadata:    map[string]any{"key": "value"},
		CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
		Status:      groups.EnabledStatus,
	}
	directAccess     = "direct"
	ascDir           = "asc"
	descDir          = "desc"
	availableActions = []string{
		"update",
		"read",
		"membership",
		"delete",
		"subgroup_create",
		"subgroup_client_create",
		"subgroup_channel_create",
		"subgroup_update",
		"subgroup_read",
		"subgroup_membership",
		"subgroup_delete",
		"subgroup_set_child",
		"subgroup_set_parent",
		"subgroup_manage_role",
		"subgroup_add_role_users",
		"subgroup_remove_role_users",
		"subgroup_view_role_users",
	}
	errGroupExists = errors.NewRequestError("group id already exists")
)

func TestSave(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM groups")
		require.Nil(t, err, fmt.Sprintf("clean groups unexpected error: %s", err))
	})

	validGroupRes := validGroup
	validGroupRes.Path = validGroup.ID
	validGroupRes.Level = 1

	repo := postgres.New(database)

	parentGroup := validGroup
	parentGroup.ID = testsutil.GenerateUUID(t)
	parentGroup.Name = namegen.Generate()

	pgroup, err := repo.Save(context.Background(), parentGroup)
	require.Nil(t, err, fmt.Sprintf("save group unexpected error: %s", err))

	validChildGroup := validGroup
	validChildGroup.ID = testsutil.GenerateUUID(t)
	validChildGroup.Name = namegen.Generate()
	validChildGroup.Parent = pgroup.ID
	validChildGroupRes := validChildGroup
	validChildGroupRes.Path = fmt.Sprintf("%s.%s", pgroup.Path, validChildGroupRes.ID)
	validChildGroupRes.Level = 2
	duplicateGroupID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc  string
		group groups.Group
		resp  groups.Group
		err   error
	}{
		{
			desc:  "add new group successfully",
			group: validGroup,
			resp:  validGroupRes,
			err:   nil,
		},
		{
			desc:  "add duplicate group",
			group: validGroup,
			err:   errGroupExists,
		},
		{
			desc:  "add group with parent",
			group: validChildGroup,
			resp:  validChildGroupRes,
			err:   nil,
		},
		{
			desc: "add group with invalid ID",
			group: groups.Group{
				ID:          invalidID,
				Domain:      testsutil.GenerateUUID(t),
				Name:        namegen.Generate(),
				Description: desc,
				Metadata:    map[string]any{"key": "value"},
				CreatedAt:   validTimestamp,
				Status:      groups.EnabledStatus,
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "add group with invalid domain",
			group: groups.Group{
				ID:          testsutil.GenerateUUID(t),
				Domain:      invalidID,
				Name:        namegen.Generate(),
				Description: desc,
				Metadata:    map[string]any{"key": "value"},
				CreatedAt:   validTimestamp,
				Status:      groups.EnabledStatus,
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "add group with invalid parent",
			group: groups.Group{
				ID:          testsutil.GenerateUUID(t),
				Parent:      testsutil.GenerateUUID(t),
				Name:        namegen.Generate(),
				Description: desc,
				Metadata:    map[string]any{"key": "value"},
				CreatedAt:   validTimestamp,
				Status:      groups.EnabledStatus,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "add group with invalid name",
			group: groups.Group{
				ID:          testsutil.GenerateUUID(t),
				Domain:      testsutil.GenerateUUID(t),
				Name:        strings.Repeat("a", 1025),
				Description: desc,
				Metadata:    map[string]any{"key": "value"},
				CreatedAt:   validTimestamp,
				Status:      groups.EnabledStatus,
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "add group with invalid description",
			group: groups.Group{
				ID:          testsutil.GenerateUUID(t),
				Domain:      testsutil.GenerateUUID(t),
				Name:        namegen.Generate(),
				Description: invalidDesc,
				Metadata:    map[string]any{"key": "value"},
				CreatedAt:   validTimestamp,
				Status:      groups.EnabledStatus,
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "add group with invalid metadata",
			group: groups.Group{
				ID:          testsutil.GenerateUUID(t),
				Domain:      testsutil.GenerateUUID(t),
				Name:        namegen.Generate(),
				Description: desc,
				Metadata: map[string]any{
					"key": make(chan int),
				},
				CreatedAt: validTimestamp,
				Status:    groups.EnabledStatus,
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "add group with invalid domain",
			group: groups.Group{
				ID:          testsutil.GenerateUUID(t),
				Name:        namegen.Generate(),
				Domain:      invalidID,
				Description: desc,
				Metadata:    map[string]any{"key": "value"},
				CreatedAt:   validTimestamp,
				Status:      groups.EnabledStatus,
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "add group with duplicate name",
			group: groups.Group{
				ID:          duplicateGroupID,
				Domain:      validGroup.Domain,
				Name:        validGroup.Name,
				Description: desc,
				Metadata:    map[string]any{"key": "different_value"},
				CreatedAt:   validTimestamp,
				Status:      groups.EnabledStatus,
				Path:        duplicateGroupID,
				Level:       1,
			},
			resp: groups.Group{
				ID:          duplicateGroupID,
				Domain:      validGroup.Domain,
				Name:        validGroup.Name,
				Description: desc,
				Metadata:    map[string]any{"key": "different_value"},
				CreatedAt:   validTimestamp,
				Status:      groups.EnabledStatus,
				Path:        duplicateGroupID,
				Level:       1,
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			group, err := repo.Save(context.Background(), tc.group)
			assert.Equal(t, tc.resp, group, fmt.Sprintf("%s: expected %v got %+v\n", tc.desc, tc.resp, group))
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM groups")
		require.Nil(t, err, fmt.Sprintf("clean groups unexpected error: %s", err))
	})

	repo := postgres.New(database)

	group, err := repo.Save(context.Background(), validGroup)
	require.Nil(t, err, fmt.Sprintf("save group unexpected error: %s", err))

	cases := []struct {
		desc   string
		update string
		group  groups.Group
		err    error
	}{
		{
			desc:   "update group successfully",
			update: "all",
			group: groups.Group{
				ID:          group.ID,
				Name:        namegen.Generate(),
				Description: desc,
				Metadata:    map[string]any{"key": "value"},
				UpdatedAt:   validTimestamp,
				UpdatedBy:   testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc:   "update group name",
			update: "name",
			group: groups.Group{
				ID:        group.ID,
				Name:      namegen.Generate(),
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc:   "update group description",
			update: "description",
			group: groups.Group{
				ID:          group.ID,
				Description: desc,
				UpdatedAt:   validTimestamp,
				UpdatedBy:   testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc:   "update group metadata",
			update: "metadata",
			group: groups.Group{
				ID:        group.ID,
				Metadata:  map[string]any{"key1": "value1"},
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc:   "update group with invalid ID",
			update: "all",
			group: groups.Group{
				ID:          testsutil.GenerateUUID(t),
				Name:        namegen.Generate(),
				Description: desc,
				Metadata:    map[string]any{"key": "value"},
				UpdatedAt:   validTimestamp,
				UpdatedBy:   testsutil.GenerateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "update group with empty ID",
			update: "all",
			group: groups.Group{
				Name:        namegen.Generate(),
				Description: desc,
				Metadata:    map[string]any{"key": "value"},
				UpdatedAt:   validTimestamp,
				UpdatedBy:   testsutil.GenerateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			group, err := repo.Update(context.Background(), tc.group)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.group.ID, group.ID, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group.ID, group.ID))
				assert.Equal(t, tc.group.UpdatedAt, group.UpdatedAt, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group.UpdatedAt, group.UpdatedAt))
				assert.Equal(t, tc.group.UpdatedBy, group.UpdatedBy, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group.UpdatedBy, group.UpdatedBy))
				switch tc.update {
				case "all":
					assert.Equal(t, tc.group.Name, group.Name, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group.Name, group.Name))
					assert.Equal(t, tc.group.Description, group.Description, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group.Description, group.Description))
					assert.Equal(t, tc.group.Metadata, group.Metadata, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group.Metadata, group.Metadata))
				case "name":
					assert.Equal(t, tc.group.Name, group.Name, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group.Name, group.Name))
				case "description":
					assert.Equal(t, tc.group.Description, group.Description, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group.Description, group.Description))
				case "metadata":
					assert.Equal(t, tc.group.Metadata, group.Metadata, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group.Metadata, group.Metadata))
				}
			}
		})
	}
}

func TestUpdateTags(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM groups")
		require.Nil(t, err, fmt.Sprintf("clean groups unexpected error: %s", err))
	})

	repo := postgres.New(database)

	_, err := repo.Save(context.Background(), validGroup)
	require.Nil(t, err, fmt.Sprintf("save group unexpected error: %s", err))

	cases := []struct {
		desc  string
		group groups.Group
		err   error
	}{
		{
			desc: "update group tags",
			group: groups.Group{
				ID:        validGroup.ID,
				Tags:      []string{"tag3", "tag4"},
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc: "update group with invalid ID",
			group: groups.Group{
				ID:        testsutil.GenerateUUID(t),
				Tags:      []string{"tag3", "tag4"},
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "update group with empty ID",
			group: groups.Group{
				Tags:      []string{"tag3", "tag4"},
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			group, err := repo.UpdateTags(context.Background(), tc.group)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.group.ID, group.ID, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group.ID, group.ID))
				assert.Equal(t, tc.group.UpdatedAt, group.UpdatedAt, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group.UpdatedAt, group.UpdatedAt))
				assert.Equal(t, tc.group.UpdatedBy, group.UpdatedBy, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group.UpdatedBy, group.UpdatedBy))
				assert.Equal(t, tc.group.Tags, group.Tags, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group.Tags, group.Tags))
			}
		})
	}
}

func TestChangeStatus(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM groups")
		require.Nil(t, err, fmt.Sprintf("clean groups unexpected error: %s", err))
	})

	repo := postgres.New(database)

	group, err := repo.Save(context.Background(), validGroup)
	require.Nil(t, err, fmt.Sprintf("save group unexpected error: %s", err))

	cases := []struct {
		desc  string
		group groups.Group
		err   error
	}{
		{
			desc: "change status group successfully",
			group: groups.Group{
				ID:        group.ID,
				Status:    groups.DisabledStatus,
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc: "change status group with invalid ID",
			group: groups.Group{
				ID:        testsutil.GenerateUUID(t),
				Status:    groups.DisabledStatus,
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "change status group with empty ID",
			group: groups.Group{
				Status:    groups.DisabledStatus,
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			group, err := repo.ChangeStatus(context.Background(), tc.group)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.group.ID, group.ID, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group.ID, group.ID))
				assert.Equal(t, tc.group.UpdatedAt, group.UpdatedAt, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group.UpdatedAt, group.UpdatedAt))
				assert.Equal(t, tc.group.UpdatedBy, group.UpdatedBy, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group.UpdatedBy, group.UpdatedBy))
				assert.Equal(t, tc.group.Status, group.Status, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group.Status, group.Status))
			}
		})
	}
}

func TestRetrieveByID(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM groups")
		require.Nil(t, err, fmt.Sprintf("clean groups unexpected error: %s", err))
	})

	repo := postgres.New(database)

	validGroupRes := validGroup
	validGroupRes.Path = validGroup.ID

	group, err := repo.Save(context.Background(), validGroup)
	require.Nil(t, err, fmt.Sprintf("save group unexpected error: %s", err))

	cases := []struct {
		desc  string
		id    string
		group groups.Group
		resp  groups.Group
		err   error
	}{
		{
			desc:  "retrieve group by id successfully",
			id:    group.ID,
			group: validGroup,
			resp:  validGroupRes,
			err:   nil,
		},
		{
			desc:  "retrieve group by id with invalid ID",
			id:    invalidID,
			group: groups.Group{},
			err:   repoerr.ErrNotFound,
		},
		{
			desc:  "retrieve group by id with empty ID",
			id:    "",
			group: groups.Group{},
			err:   repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			group, err := repo.RetrieveByID(context.Background(), tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Nil(t, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
				assert.Equal(t, tc.resp, group, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group, group))
			}
		})
	}
}

func TestRetrieveByIDAndUser(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM groups")
		require.Nil(t, err, fmt.Sprintf("clean groups unexpected error: %s", err))
	})

	repo := postgres.New(database)

	domainID := testsutil.GenerateUUID(t)
	userID := testsutil.GenerateUUID(t)
	num := 10
	items := []groups.Group{}
	for i := 0; i < num; i++ {
		name := namegen.Generate()
		group := groups.Group{
			ID:          testsutil.GenerateUUID(t),
			Domain:      domainID,
			Name:        name,
			Description: desc,
			Metadata:    map[string]any{"name": name},
			CreatedAt:   validTimestamp,
			Status:      groups.EnabledStatus,
		}
		grp, err := repo.Save(context.Background(), group)
		require.Nil(t, err, fmt.Sprintf("create group unexpected error: %s", err))
		newRolesProvision := []roles.RoleProvision{
			{
				Role: roles.Role{
					ID:        testsutil.GenerateUUID(t) + "_" + grp.ID,
					Name:      "admin",
					EntityID:  grp.ID,
					CreatedAt: validTimestamp,
					CreatedBy: userID,
				},
				OptionalActions: availableActions,
				OptionalMembers: []string{userID},
			},
		}
		_, err = repo.AddRoles(context.Background(), newRolesProvision)
		require.Nil(t, err, fmt.Sprintf("add roles unexpected error: %s", err))
		ngrp := grp
		ngrp.RoleID = newRolesProvision[0].Role.ID
		ngrp.RoleName = newRolesProvision[0].Role.Name
		ngrp.AccessType = directAccess
		items = append(items, ngrp)
	}

	cases := []struct {
		desc     string
		groupID  string
		userID   string
		domainID string
		resp     groups.Group
		err      error
	}{
		{
			desc:     "retrieve group by id and user successfully",
			groupID:  items[0].ID,
			userID:   userID,
			domainID: domainID,
			resp:     items[0],
			err:      nil,
		},
		{
			desc:     "retrieve group by id and user successfully",
			groupID:  items[5].ID,
			userID:   userID,
			domainID: domainID,
			resp:     items[5],
			err:      nil,
		},
		{
			desc:     "retrieve group by id and user with invalid group ID",
			groupID:  invalidID,
			userID:   userID,
			domainID: domainID,
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve group by id and user with empty group ID",
			groupID:  "",
			userID:   userID,
			domainID: domainID,
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve group by id and user with invalid user ID",
			groupID:  items[0].ID,
			userID:   testsutil.GenerateUUID(t),
			domainID: domainID,
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve group by id and user with empty user ID",
			groupID:  items[0].ID,
			userID:   "",
			domainID: domainID,
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve group by id and user with invalid domain ID",
			groupID:  items[0].ID,
			userID:   userID,
			domainID: invalidID,
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve group by id and user with empty domain ID",
			groupID:  items[0].ID,
			userID:   userID,
			domainID: "",
			err:      repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			group, err := repo.RetrieveByIDAndUser(context.Background(), tc.domainID, tc.userID, tc.groupID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				group.Actions = nil
				group.Level = 1
				group.AccessProviderRoleActions = nil
				assert.Equal(t, tc.resp, group, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, group))
			}
		})
	}
}

func TestRetrieveAll(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM groups")
		require.Nil(t, err, fmt.Sprintf("clean groups unexpected error: %s", err))
	})

	repo := postgres.New(database)
	num := 200
	baseTime := time.Now().UTC().Truncate(time.Millisecond)

	var items []groups.Group
	parentID := ""
	for i := 0; i < num; i++ {
		name := namegen.Generate()
		group := groups.Group{
			ID:          testsutil.GenerateUUID(t),
			Domain:      testsutil.GenerateUUID(t),
			Parent:      parentID,
			Name:        name,
			Description: desc,
			Metadata:    map[string]any{"name": name},
			CreatedAt:   baseTime.Add(time.Duration(i) * time.Millisecond),
			UpdatedAt:   baseTime.Add(time.Duration(i) * time.Millisecond),
			Status:      groups.EnabledStatus,
			Tags:        []string{"tag1", "tag2"},
		}
		if i%99 == 0 {
			group.Tags = []string{"tag1", "tag3"}
		}
		_, err := repo.Save(context.Background(), group)
		require.Nil(t, err, fmt.Sprintf("create group unexpected error: %s", err))
		items = append(items, group)
		if i%20 == 0 {
			parentID = group.ID
		}
	}

	reversedGroups := []groups.Group{}
	for i := len(items) - 1; i >= 0; i-- {
		reversedGroups = append(reversedGroups, items[i])
	}

	cases := []struct {
		desc     string
		page     groups.Page
		response groups.Page
		err      error
	}{
		{
			desc: "retrieve groups successfully",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  10,
					Order:  "created_at",
					Dir:    ascDir,
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  uint64(num),
					Offset: 0,
					Limit:  10,
				},
				Groups: items[:10],
			},
			err: nil,
		},
		{
			desc: "retrieve groups with offset",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 10,
					Limit:  10,
					Order:  "created_at",
					Dir:    ascDir,
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  uint64(num),
					Offset: 10,
					Limit:  10,
				},
				Groups: items[10:20],
			},
			err: nil,
		},
		{
			desc: "retrieve groups with limit",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  50,

					Order: "created_at",
					Dir:   ascDir,
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  uint64(num),
					Offset: 0,
					Limit:  50,
				},
				Groups: items[:50],
			},
			err: nil,
		},
		{
			desc: "retrieve groups with offset and limit",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 50,
					Limit:  50,
					Order:  "created_at",
					Dir:    ascDir,
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  uint64(num),
					Offset: 50,
					Limit:  50,
				},
				Groups: items[50:100],
			},
			err: nil,
		},
		{
			desc: "retrieve groups with offset out of range",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 1000,
					Limit:  50,
					Order:  "created_at",
					Dir:    ascDir,
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  uint64(num),
					Offset: 1000,
					Limit:  50,
				},
				Groups: []groups.Group(nil),
			},
			err: nil,
		},
		{
			desc: "retrieve groups with offset and limit out of range",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 170,
					Limit:  50,
					Order:  "created_at",
					Dir:    ascDir,
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  uint64(num),
					Offset: 170,
					Limit:  50,
				},
				Groups: items[170:200],
			},
			err: nil,
		},
		{
			desc: "retrieve groups with limit out of range",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  1000,
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  uint64(num),
					Offset: 0,
					Limit:  1000,
				},
				Groups: items,
			},
			err: nil,
		},
		{
			desc: "retrieve groups with empty page",
			page: groups.Page{},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  uint64(num),
					Offset: 0,
					Limit:  0,
				},
				Groups: []groups.Group(nil),
			},
			err: nil,
		},
		{
			desc: "retrieve groups with name",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  10,
					Name:   items[0].Name,
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Groups: []groups.Group{items[0]},
			},
			err: nil,
		},
		{
			desc: "retrieve groups with domain",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset:   0,
					Limit:    10,
					DomainID: items[0].Domain,
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Groups: []groups.Group{items[0]},
			},
			err: nil,
		},
		{
			desc: "retrieve groups with metadata",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset:   0,
					Limit:    10,
					Metadata: items[0].Metadata,
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Groups: []groups.Group{items[0]},
			},
			err: nil,
		},
		{
			desc: "retrieve groups with invalid metadata",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  10,
					Metadata: map[string]any{
						"key": make(chan int),
					},
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
				Groups: []groups.Group(nil),
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "retrieve groups with id",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  10,
					ID:     items[0].ID,
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Groups: []groups.Group{items[0]},
			},
			err: nil,
		},
		{
			desc: "retrieve groups with wrong id",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  10,
					ID:     "wrong",
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
				Groups: []groups.Group(nil),
			},
			err: nil,
		},
		{
			desc: "retrieve groups with order by name ascending",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  10,
					Order:  "name",
					Dir:    ascDir,
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  uint64(num),
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "retrieve groups with order by name descending",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  10,
					Order:  "name",
					Dir:    descDir,
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  uint64(num),
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "retrieve groups with order by created_at ascending",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  10,
					Order:  "created_at",
					Dir:    ascDir,
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  uint64(num),
					Offset: 0,
					Limit:  10,
				},
				Groups: items[:10],
			},
			err: nil,
		},
		{
			desc: "retrieve groups with order by created_at descending",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  10,
					Order:  "created_at",
					Dir:    descDir,
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  uint64(num),
					Offset: 0,
					Limit:  10,
				},
				Groups: reversedGroups[:10],
			},
			err: nil,
		},
		{
			desc: "retrieve groups with order by updated_at ascending",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  10,
					Order:  "updated_at",
					Dir:    ascDir,
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  uint64(num),
					Offset: 0,
					Limit:  10,
				},
				Groups: items[:10],
			},
			err: nil,
		},
		{
			desc: "retrieve groups with order by updated_at descending",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  10,
					Order:  "updated_at",
					Dir:    descDir,
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  uint64(num),
					Offset: 0,
					Limit:  10,
				},
				Groups: reversedGroups[:10],
			},
			err: nil,
		},
		{
			desc: "retrieve groups with single tag",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  uint64(num),
					Tags:   groups.TagsQuery{Elements: []string{"tag1"}, Operator: groups.OrOp},
					Status: groups.AllStatus,
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  200,
					Offset: 0,
					Limit:  uint64(num),
				},
				Groups: items,
			},
			err: nil,
		},
		{
			desc: "retrieve group with multiple tags and OR operator",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  uint64(num),
					Tags:   groups.TagsQuery{Elements: []string{"tag2", "tag3"}, Operator: groups.OrOp},
					Status: groups.AllStatus,
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  200,
					Offset: 0,
					Limit:  uint64(num),
				},
				Groups: items,
			},
		},
		{
			desc: "retrieve group with multiple tags and AND operator",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  uint64(num),
					Tags:   groups.TagsQuery{Elements: []string{"tag1", "tag3"}, Operator: groups.AndOp},
					Status: groups.AllStatus,
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  3,
					Offset: 0,
					Limit:  uint64(num),
				},
				Groups: []groups.Group{items[0], items[99], items[198]},
			},
		},
		{
			desc: "retrieve group with invalid tags",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  uint64(num),
					Tags:   groups.TagsQuery{Elements: []string{namegen.Generate(), namegen.Generate()}, Operator: groups.OrOp},
					Status: groups.AllStatus,
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  0,
					Offset: 0,
					Limit:  uint64(num),
				},
				Groups: []groups.Group(nil),
			},
		},
		{
			desc: "retrieve groups with created_from",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset:      0,
					Limit:       200,
					Order:       "created_at",
					Dir:         ascDir,
					CreatedFrom: baseTime.Add(100 * time.Millisecond),
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  100,
					Offset: 0,
					Limit:  200,
				},
				Groups: items[100:],
			},
			err: nil,
		},
		{
			desc: "retrieve groups with created_to",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset:    0,
					Limit:     200,
					Order:     "created_at",
					Dir:       ascDir,
					CreatedTo: baseTime.Add(99 * time.Millisecond),
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  100,
					Offset: 0,
					Limit:  200,
				},
				Groups: items[:100],
			},
			err: nil,
		},
		{
			desc: "retrieve groups with both created_from and created_to",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset:      0,
					Limit:       200,
					Order:       "created_at",
					Dir:         ascDir,
					CreatedFrom: baseTime.Add(50 * time.Millisecond),
					CreatedTo:   baseTime.Add(149 * time.Millisecond),
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  100,
					Offset: 0,
					Limit:  200,
				},
				Groups: items[50:150],
			},
			err: nil,
		},
		{
			desc: "retrieve groups with created_from returning no results",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset:      0,
					Limit:       10,
					Order:       "created_at",
					Dir:         ascDir,
					CreatedFrom: baseTime.Add(1000 * time.Millisecond),
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
				Groups: []groups.Group(nil),
			},
			err: nil,
		},
		{
			desc: "retrieve groups with created_to returning no results",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset:    0,
					Limit:     10,
					Order:     "created_at",
					Dir:       ascDir,
					CreatedTo: baseTime.Add(-1 * time.Millisecond),
				},
			},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
				Groups: []groups.Group(nil),
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			switch groups, err := repo.RetrieveAll(context.Background(), tc.page.PageMeta); {
			case err == nil:
				assert.Nil(t, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
				assert.Equal(t, tc.response.Total, groups.Total, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Total, groups.Total))
				assert.Equal(t, tc.response.Limit, groups.Limit, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Limit, groups.Limit))
				assert.Equal(t, tc.response.Offset, groups.Offset, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Offset, groups.Offset))
				got := stripGroupDetails(groups.Groups)
				if len(tc.response.Groups) > 0 {
					resp := stripGroupDetails(tc.response.Groups)
					assert.ElementsMatch(t, resp, got, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, resp, got))
				}
				verifyGroupsOrdering(t, groups.Groups, tc.page.PageMeta.Order, tc.page.PageMeta.Dir)
			default:
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			}
		})
	}
}

func TestRetrieveByIDs(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM groups")
		require.Nil(t, err, fmt.Sprintf("clean groups unexpected error: %s", err))
	})

	repo := postgres.New(database)
	num := 200

	var items []groups.Group
	parentID := ""
	for i := 0; i < num; i++ {
		name := namegen.Generate()
		group := groups.Group{
			ID:          testsutil.GenerateUUID(t),
			Domain:      testsutil.GenerateUUID(t),
			Parent:      parentID,
			Name:        name,
			Description: desc,
			Metadata:    map[string]any{"name": name},
			CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
			Status:      groups.EnabledStatus,
		}
		_, err := repo.Save(context.Background(), group)
		require.Nil(t, err, fmt.Sprintf("create invitation unexpected error: %s", err))
		items = append(items, group)
		if i%20 == 0 {
			parentID = group.ID
		}
	}

	cases := []struct {
		desc     string
		page     groups.Page
		ids      []string
		response groups.Page
		err      error
	}{
		{
			desc: "retrieve groups successfully",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  10,
				},
			},
			ids: getIDs(items[0:3]),
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  3,
					Offset: 0,
					Limit:  10,
				},
				Groups: items[0:3],
			},
			err: nil,
		},
		{
			desc: "retrieve groups with empty ids",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  10,
				},
			},
			ids: []string{},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  10,
				},
				Groups: []groups.Group(nil),
			},
			err: nil,
		},
		{
			desc: "retrieve groups with empty ids but with domain",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset:   0,
					Limit:    10,
					DomainID: items[0].Domain,
				},
			},
			ids: []string{},
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Groups: []groups.Group{items[0]},
			},
			err: nil,
		},
		{
			desc: "retrieve groups with offset",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 10,
					Limit:  10,
				},
			},
			ids: getIDs(items[0:20]),
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  20,
					Offset: 10,
					Limit:  10,
				},
				Groups: items[10:20],
			},
			err: nil,
		},
		{
			desc: "retrieve groups with offset out of range",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 1000,
					Limit:  50,
				},
			},
			ids: getIDs(items[0:20]),
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  20,
					Offset: 1000,
					Limit:  50,
				},
				Groups: []groups.Group(nil),
			},
			err: nil,
		},
		{
			desc: "retrieve groups with offset and limit out of range",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 15,
					Limit:  10,
				},
			},
			ids: getIDs(items[0:20]),
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  20,
					Offset: 15,
					Limit:  10,
				},
				Groups: items[15:20],
			},
			err: nil,
		},
		{
			desc: "retrieve groups with limit out of range",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  1000,
				},
			},
			ids: getIDs(items[0:20]),
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  20,
					Offset: 0,
					Limit:  1000,
				},
				Groups: items[:20],
			},
			err: nil,
		},
		{
			desc: "retrieve groups with name",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  10,
					Name:   items[0].Name,
				},
			},
			ids: getIDs(items[0:20]),
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Groups: []groups.Group{items[0]},
			},
			err: nil,
		},
		{
			desc: "retrieve groups with domain",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset:   0,
					Limit:    10,
					DomainID: items[0].Domain,
				},
			},
			ids: getIDs(items[0:20]),
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Groups: []groups.Group{items[0]},
			},
			err: nil,
		},
		{
			desc: "retrieve groups with metadata",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset:   0,
					Limit:    10,
					Metadata: items[0].Metadata,
				},
			},
			ids: getIDs(items[0:20]),
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Groups: []groups.Group{items[0]},
			},
			err: nil,
		},
		{
			desc: "retrieve groups with invalid metadata",
			page: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  10,
					Metadata: map[string]any{
						"key": make(chan int),
					},
				},
			},
			ids: getIDs(items[0:20]),
			response: groups.Page{
				PageMeta: groups.PageMeta{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
				Groups: []groups.Group(nil),
			},
			err: errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			groups, err := repo.RetrieveByIDs(context.Background(), tc.page.PageMeta, tc.ids...)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.response.Total, groups.Total, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Total, groups.Total))
				assert.Equal(t, tc.response.Limit, groups.Limit, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Limit, groups.Limit))
				assert.Equal(t, tc.response.Offset, groups.Offset, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Offset, groups.Offset))
				got := stripGroupDetails(groups.Groups)
				resp := stripGroupDetails(tc.response.Groups)
				assert.ElementsMatch(t, resp, got, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, resp, got))
			}
		})
	}
}

func TestDelete(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM groups")
		require.Nil(t, err, fmt.Sprintf("clean groups unexpected error: %s", err))
	})

	repo := postgres.New(database)

	group, err := repo.Save(context.Background(), validGroup)
	require.Nil(t, err, fmt.Sprintf("save group unexpected error: %s", err))

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "delete group successfully",
			id:   group.ID,
			err:  nil,
		},
		{
			desc: "delete group with invalid ID",
			id:   invalidID,
			err:  repoerr.ErrNotFound,
		},
		{
			desc: "delete group with empty ID",
			id:   "",
			err:  repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.Delete(context.Background(), tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestAssignParentGroup(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM groups")
		require.Nil(t, err, fmt.Sprintf("clean groups unexpected error: %s", err))
	})

	repo := postgres.New(database)

	num := 10

	var items []groups.Group
	for i := 0; i < num; i++ {
		name := namegen.Generate()
		group := groups.Group{
			ID:          testsutil.GenerateUUID(t),
			Domain:      testsutil.GenerateUUID(t),
			Name:        name,
			Description: desc,
			Metadata:    map[string]any{"name": name},
			CreatedAt:   validTimestamp,
			Status:      groups.EnabledStatus,
		}
		_, err := repo.Save(context.Background(), group)
		require.Nil(t, err, fmt.Sprintf("create invitation unexpected error: %s", err))
		items = append(items, group)
	}

	cases := []struct {
		desc string
		id   string
		ids  []string
		err  error
	}{
		{
			desc: "assign parent group successfully",
			id:   items[0].ID,
			ids:  []string{items[1].ID, items[2].ID, items[3].ID, items[4].ID, items[5].ID},
			err:  nil,
		},
		{
			desc: "assign parent group with invalid ID",
			id:   testsutil.GenerateUUID(t),
			ids:  []string{items[1].ID, items[2].ID, items[3].ID, items[4].ID, items[5].ID},
			err:  repoerr.ErrUpdateEntity,
		},
		{
			desc: "assign parent group with empty ID",
			id:   "",
			ids:  []string{items[1].ID, items[2].ID, items[3].ID, items[4].ID, items[5].ID},
			err:  repoerr.ErrUpdateEntity,
		},
		{
			desc: "assign parent group with invalid group IDs",
			id:   items[0].ID,
			ids:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t), testsutil.GenerateUUID(t), testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			err:  nil,
		},
		{
			desc: "assign parent group with empty group IDs",
			id:   items[0].ID,
			ids:  []string{},
			err:  nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.AssignParentGroup(context.Background(), tc.id, tc.ids...)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestUnassignParentGroup(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM groups")
		require.Nil(t, err, fmt.Sprintf("clean groups unexpected error: %s", err))
	})

	repo := postgres.New(database)

	num := 10

	var items []groups.Group
	parentID := ""
	for i := 0; i < num; i++ {
		name := namegen.Generate()
		group := groups.Group{
			ID:          testsutil.GenerateUUID(t),
			Domain:      testsutil.GenerateUUID(t),
			Parent:      parentID,
			Name:        name,
			Description: desc,
			Metadata:    map[string]any{"name": name},
			CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
			Status:      groups.EnabledStatus,
		}
		_, err := repo.Save(context.Background(), group)
		require.Nil(t, err, fmt.Sprintf("create invitation unexpected error: %s", err))
		items = append(items, group)
		if i == 0 {
			parentID = group.ID
		}
	}

	cases := []struct {
		desc string
		id   string
		ids  []string
		err  error
	}{
		{
			desc: "un-assign parent group successfully",
			id:   items[0].ID,
			ids:  []string{items[1].ID, items[2].ID, items[3].ID, items[4].ID, items[5].ID},
			err:  nil,
		},
		{
			desc: "un-assign parent group with invalid ID",
			id:   testsutil.GenerateUUID(t),
			ids:  []string{items[1].ID, items[2].ID, items[3].ID, items[4].ID, items[5].ID},
			err:  repoerr.ErrUpdateEntity,
		},
		{
			desc: "un-assign parent group with empty ID",
			id:   "",
			ids:  []string{items[1].ID, items[2].ID, items[3].ID, items[4].ID, items[5].ID},
			err:  repoerr.ErrUpdateEntity,
		},
		{
			desc: "un-assign parent group with invalid group IDs",
			id:   items[0].ID,
			ids:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t), testsutil.GenerateUUID(t), testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			err:  nil,
		},
		{
			desc: "un-assign parent group with empty group IDs",
			id:   items[0].ID,
			ids:  []string{},
			err:  nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.UnassignParentGroup(context.Background(), tc.id, tc.ids...)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestUnassignAllChildrenGroups(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM groups")
		require.Nil(t, err, fmt.Sprintf("clean groups unexpected error: %s", err))
	})

	repo := postgres.New(database)

	num := 10

	var items []groups.Group
	parentID := ""
	for i := 0; i < num; i++ {
		name := namegen.Generate()
		group := groups.Group{
			ID:          testsutil.GenerateUUID(t),
			Domain:      testsutil.GenerateUUID(t),
			Parent:      parentID,
			Name:        name,
			Description: desc,
			Metadata:    map[string]any{"name": name},
			CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
			Status:      groups.EnabledStatus,
		}
		_, err := repo.Save(context.Background(), group)
		require.Nil(t, err, fmt.Sprintf("create invitation unexpected error: %s", err))
		items = append(items, group)
		if i == 0 {
			parentID = group.ID
		}
	}

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "un-assign all children groups successfully",
			id:   items[0].ID,
			err:  nil,
		},
		{
			desc: "un-assign all children groups with invalid ID",
			id:   testsutil.GenerateUUID(t),
			err:  repoerr.ErrNotFound,
		},
		{
			desc: "un-assign all children groups with empty ID",
			id:   "",
			err:  repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.UnassignAllChildrenGroups(context.Background(), tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestRetrieveHierarchy(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM groups")
		require.Nil(t, err, fmt.Sprintf("clean groups unexpected error: %s", err))
	})

	repo := postgres.New(database)

	userID := testsutil.GenerateUUID(t)
	domainID := testsutil.GenerateUUID(t)
	num := 10

	var items []groups.Group
	parentID := ""
	for i := 0; i < num; i++ {
		name := namegen.Generate()
		group := groups.Group{
			ID:          testsutil.GenerateUUID(t),
			Domain:      domainID,
			Parent:      parentID,
			Name:        name,
			Description: desc,
			Metadata:    map[string]any{"name": name},
			CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
			Status:      groups.EnabledStatus,
		}
		_, err := repo.Save(context.Background(), group)
		require.Nil(t, err, fmt.Sprintf("create group unexpected error: %s", err))
		newRolesProvision := []roles.RoleProvision{
			{
				Role: roles.Role{
					ID:        testsutil.GenerateUUID(t) + "_" + group.ID,
					Name:      "admin",
					EntityID:  group.ID,
					CreatedAt: validTimestamp,
					CreatedBy: userID,
				},
				OptionalActions: availableActions,
				OptionalMembers: []string{userID},
			},
		}
		_, err = repo.AddRoles(context.Background(), newRolesProvision)
		require.Nil(t, err, fmt.Sprintf("add roles unexpected error: %s", err))
		items = append(items, group)
		if i == 0 {
			parentID = group.ID
		}
	}

	cases := []struct {
		desc     string
		groupID  string
		userID   string
		domainID string
		hm       groups.HierarchyPageMeta
		resp     groups.HierarchyPage
		err      error
	}{
		{
			desc:     "retrieve ancestors successfully",
			groupID:  items[1].ID,
			userID:   userID,
			domainID: domainID,
			hm: groups.HierarchyPageMeta{
				Level:     1,
				Direction: +1,
				Tree:      false,
			},
			resp: groups.HierarchyPage{
				Groups: []groups.Group{items[0], items[1]},
				HierarchyPageMeta: groups.HierarchyPageMeta{
					Level:     1,
					Direction: +1,
					Tree:      false,
				},
			},
			err: nil,
		},
		{
			desc:     "retrieve descendants successfully",
			groupID:  items[0].ID,
			userID:   userID,
			domainID: domainID,
			hm: groups.HierarchyPageMeta{
				Level:     1,
				Direction: -1,
				Tree:      false,
			},
			resp: groups.HierarchyPage{
				Groups: items,
				HierarchyPageMeta: groups.HierarchyPageMeta{
					Level:     1,
					Direction: -1,
					Tree:      false,
				},
			},
			err: nil,
		},
		{
			desc:     "retrieve hierarchy with invalid ID",
			groupID:  testsutil.GenerateUUID(t),
			userID:   userID,
			domainID: domainID,
			err:      nil,
		},
		{
			desc:     "retrieve hierarchy with empty ID",
			groupID:  "",
			userID:   userID,
			domainID: domainID,
			err:      nil,
		},
		{
			desc:     "retrieve hierarchy with invalid domain ID",
			groupID:  items[0].ID,
			userID:   userID,
			domainID: testsutil.GenerateUUID(t),
			hm: groups.HierarchyPageMeta{
				Level:     1,
				Direction: -1,
				Tree:      false,
			},
			resp: groups.HierarchyPage{
				Groups: []groups.Group(nil),
				HierarchyPageMeta: groups.HierarchyPageMeta{
					Level:     1,
					Direction: -1,
					Tree:      false,
				},
			},
			err: nil,
		},
		{
			desc:     "retrieve hierarchy with invalid user ID",
			groupID:  items[0].ID,
			userID:   testsutil.GenerateUUID(t),
			domainID: domainID,
			hm: groups.HierarchyPageMeta{
				Level:     1,
				Direction: -1,
				Tree:      false,
			},
			resp: groups.HierarchyPage{
				Groups: []groups.Group(nil),
				HierarchyPageMeta: groups.HierarchyPageMeta{
					Level:     1,
					Direction: -1,
					Tree:      false,
				},
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			gpPage, err := repo.RetrieveHierarchy(context.Background(), tc.domainID, tc.userID, tc.groupID, tc.hm)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				got := stripGroupDetails(gpPage.Groups)
				resp := stripGroupDetails(tc.resp.Groups)
				assert.ElementsMatch(t, resp, got, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, resp, got))
			}
		})
	}
}

func TestRetrieveAllParentGroups(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM groups")
		require.Nil(t, err, fmt.Sprintf("clean groups unexpected error: %s", err))
	})

	repo := postgres.New(database)

	parentID := ""
	domainID := testsutil.GenerateUUID(t)
	userID := testsutil.GenerateUUID(t)
	num := 10
	halfindex := num/2 + 1
	items := []groups.Group{}
	for i := 0; i < num; i++ {
		name := namegen.Generate()
		group := groups.Group{
			ID:          testsutil.GenerateUUID(t),
			Domain:      domainID,
			Name:        name,
			Parent:      parentID,
			Description: desc,
			Metadata:    map[string]any{"name": name},
			CreatedAt:   validTimestamp,
			Status:      groups.EnabledStatus,
		}
		grp, err := repo.Save(context.Background(), group)
		require.Nil(t, err, fmt.Sprintf("create group unexpected error: %s", err))
		parentID = grp.ID
		newRolesProvision := []roles.RoleProvision{
			{
				Role: roles.Role{
					ID:        testsutil.GenerateUUID(t) + "_" + grp.ID,
					Name:      "admin",
					EntityID:  grp.ID,
					CreatedAt: validTimestamp,
					CreatedBy: userID,
				},
				OptionalActions: availableActions,
				OptionalMembers: []string{userID},
			},
		}
		_, err = repo.AddRoles(context.Background(), newRolesProvision)
		require.Nil(t, err, fmt.Sprintf("add roles unexpected error: %s", err))
		ngrp := grp
		ngrp.RoleID = newRolesProvision[0].Role.ID
		ngrp.RoleName = newRolesProvision[0].Role.Name
		ngrp.AccessType = directAccess
		items = append(items, ngrp)
	}

	cases := []struct {
		desc     string
		id       string
		domainID string
		userID   string
		pageMeta groups.PageMeta
		resp     groups.Page
		err      error
	}{
		{
			desc:     "retrieve all parent groups successfully",
			id:       items[num-1].ID,
			domainID: domainID,
			userID:   userID,
			pageMeta: groups.PageMeta{
				Offset: 0,
				Limit:  20,
			},
			resp: groups.Page{
				PageMeta: groups.PageMeta{
					Total: uint64(num),
				},
				Groups: items,
			},
			err: nil,
		},
		{
			desc:     "retrieve half of all parent groups successfully",
			id:       items[num/2].ID,
			domainID: domainID,
			userID:   userID,
			pageMeta: groups.PageMeta{
				Offset: 0,
				Limit:  20,
			},
			resp: groups.Page{
				PageMeta: groups.PageMeta{
					Total: uint64(halfindex),
				},
				Groups: items[:halfindex],
			},
			err: nil,
		},
		{
			desc:     "retrieve all parent groups with invalid group ID",
			id:       testsutil.GenerateUUID(t),
			domainID: domainID,
			userID:   userID,
			pageMeta: groups.PageMeta{
				Offset: 0,
				Limit:  20,
			},
			resp: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 0,
				},
				Groups: []groups.Group(nil),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve all parent groups with empty group ID",
			id:       "",
			domainID: domainID,
			userID:   userID,
			pageMeta: groups.PageMeta{
				Offset: 0,
				Limit:  20,
			},
			resp: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 0,
				},
				Groups: []groups.Group(nil),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve all parent groups with invalid domain ID",
			id:       items[num-1].ID,
			domainID: testsutil.GenerateUUID(t),
			userID:   userID,
			pageMeta: groups.PageMeta{
				Offset: 0,
				Limit:  20,
			},
			resp: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 0,
				},
				Groups: []groups.Group(nil),
			},
			err: nil,
		},
		{
			desc:     "retrieve all parent groups with invalid user ID",
			id:       items[num-1].ID,
			domainID: domainID,
			userID:   testsutil.GenerateUUID(t),
			pageMeta: groups.PageMeta{
				Offset: 0,
				Limit:  20,
			},
			resp: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 0,
				},
				Groups: []groups.Group(nil),
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			groups, err := repo.RetrieveAllParentGroups(context.Background(), tc.domainID, tc.userID, tc.id, tc.pageMeta)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.resp.Total, groups.Total, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.resp.Total, groups.Total))
				got := stripGroupDetails(groups.Groups)
				resp := stripGroupDetails(tc.resp.Groups)
				assert.ElementsMatch(t, resp, got, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, resp, got))
			}
		})
	}
}

func TestRetrieveChildrenGroups(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM groups")
		require.Nil(t, err, fmt.Sprintf("clean groups unexpected error: %s", err))
	})

	repo := postgres.New(database)

	parentID := ""
	domainID := testsutil.GenerateUUID(t)
	userID := testsutil.GenerateUUID(t)
	num := 10
	items := []groups.Group{}
	for i := 0; i < num; i++ {
		name := namegen.Generate()
		group := groups.Group{
			ID:          testsutil.GenerateUUID(t),
			Domain:      domainID,
			Name:        name,
			Parent:      parentID,
			Description: desc,
			Metadata:    map[string]any{"name": name},
			CreatedAt:   validTimestamp,
			Status:      groups.EnabledStatus,
		}
		grp, err := repo.Save(context.Background(), group)
		require.Nil(t, err, fmt.Sprintf("create group unexpected error: %s", err))
		parentID = grp.ID
		newRolesProvision := []roles.RoleProvision{
			{
				Role: roles.Role{
					ID:        testsutil.GenerateUUID(t) + "_" + grp.ID,
					Name:      "admin",
					EntityID:  grp.ID,
					CreatedAt: validTimestamp,
					CreatedBy: userID,
				},
				OptionalActions: availableActions,
				OptionalMembers: []string{userID},
			},
		}
		_, err = repo.AddRoles(context.Background(), newRolesProvision)
		require.Nil(t, err, fmt.Sprintf("add roles unexpected error: %s", err))
		ngrp := grp
		ngrp.RoleID = newRolesProvision[0].Role.ID
		ngrp.RoleName = newRolesProvision[0].Role.Name
		ngrp.AccessType = directAccess
		items = append(items, ngrp)
	}

	cases := []struct {
		desc       string
		id         string
		domainID   string
		userID     string
		startLevel int64
		endLevel   int64
		pageMeta   groups.PageMeta
		resp       groups.Page
		err        error
	}{
		{
			desc:       "retrieve children groups from parent group level successfully",
			id:         items[0].ID,
			domainID:   domainID,
			userID:     userID,
			startLevel: 0,
			endLevel:   -1,
			pageMeta: groups.PageMeta{
				Offset: 0,
				Limit:  20,
			},
			resp: groups.Page{
				PageMeta: groups.PageMeta{
					Total: uint64(num),
				},
				Groups: items,
			},
			err: nil,
		},
		{
			desc:       "Retrieve specific level of children groups from parent group level",
			id:         items[0].ID,
			domainID:   domainID,
			userID:     userID,
			startLevel: 1,
			endLevel:   1,
			pageMeta: groups.PageMeta{
				Offset: 0,
				Limit:  20,
			},
			resp: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 1,
				},
				Groups: []groups.Group{items[1]},
			},
			err: nil,
		},
		{
			desc: "Retrieve all children groups from specific level from parent group level",
			id:   items[0].ID,
			pageMeta: groups.PageMeta{
				Offset: 0,
				Limit:  20,
			},
			domainID:   domainID,
			userID:     userID,
			startLevel: 2,
			endLevel:   -1,
			resp: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 8,
				},
				Groups: items[2:],
			},
			err: nil,
		},
		{
			desc: "Retrieve all children groups from specific level to specific level from parent group level",
			id:   items[0].ID,
			pageMeta: groups.PageMeta{
				Offset: 0,
				Limit:  20,
			},
			domainID:   domainID,
			userID:     userID,
			startLevel: 1,
			endLevel:   2,
			resp: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 2,
				},
				Groups: items[1:3],
			},
			err: nil,
		},
		{
			desc:       "Retrieve all children groups with invalid group ID",
			id:         testsutil.GenerateUUID(t),
			domainID:   domainID,
			userID:     userID,
			startLevel: 0,
			endLevel:   -1,
			pageMeta: groups.PageMeta{
				Offset: 0,
				Limit:  20,
			},
			resp: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 0,
				},
				Groups: []groups.Group(nil),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:       "Retrieve all children groups with empty group ID",
			id:         "",
			domainID:   domainID,
			userID:     userID,
			startLevel: 0,
			endLevel:   -1,
			pageMeta: groups.PageMeta{
				Offset: 0,
				Limit:  20,
			},
			resp: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 0,
				},
				Groups: []groups.Group(nil),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:       "Retrieve all children groups with invalid domain ID",
			id:         items[0].ID,
			domainID:   testsutil.GenerateUUID(t),
			userID:     userID,
			startLevel: 0,
			endLevel:   -1,
			pageMeta: groups.PageMeta{
				Offset: 0,
				Limit:  20,
			},
			resp: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 0,
				},
				Groups: []groups.Group(nil),
			},
			err: nil,
		},
		{
			desc:       "Retrieve all children groups with invalid user ID",
			id:         items[0].ID,
			domainID:   domainID,
			userID:     testsutil.GenerateUUID(t),
			startLevel: 0,
			endLevel:   -1,
			pageMeta: groups.PageMeta{
				Offset: 0,
				Limit:  20,
			},
			resp: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 0,
				},
				Groups: []groups.Group(nil),
			},
			err: nil,
		},
		{
			desc:       "Retrieve all children groups with invalid start level",
			id:         items[0].ID,
			domainID:   domainID,
			userID:     userID,
			startLevel: -1,
			endLevel:   -1,
			pageMeta: groups.PageMeta{
				Offset: 0,
				Limit:  20,
			},
			resp: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 0,
				},
				Groups: []groups.Group(nil),
			},
			err: repoerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			groups, err := repo.RetrieveChildrenGroups(context.Background(), tc.domainID, tc.userID, tc.id, tc.startLevel, tc.endLevel, tc.pageMeta)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.resp.Total, groups.Total, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.resp.Total, groups.Total))
				got := stripGroupDetails(groups.Groups)
				resp := stripGroupDetails(tc.resp.Groups)
				assert.ElementsMatch(t, resp, got, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, resp, got))
			}
		})
	}
}

func getIDs(groups []groups.Group) []string {
	var ids []string
	for _, group := range groups {
		ids = append(ids, group.ID)
	}

	return ids
}

func stripGroupDetails(groups []groups.Group) []groups.Group {
	for i := range groups {
		groups[i].Level = 0
		groups[i].Path = ""
		groups[i].CreatedAt = validTimestamp
		groups[i].UpdatedAt = validTimestamp
		groups[i].Actions = nil
		groups[i].AccessProviderRoleActions = nil
	}

	return groups
}

func verifyGroupsOrdering(t *testing.T, groups []groups.Group, order, dir string) {
	if order == "" || len(groups) <= 1 {
		return
	}

	for i := 0; i < len(groups)-1; i++ {
		switch order {
		case "name":
			if dir == ascDir {
				assert.LessOrEqual(t, groups[i].Name, groups[i+1].Name, fmt.Sprintf("Groups not ordered by name ascending at index %d: %s > %s", i, groups[i].Name, groups[i+1].Name))
				continue
			}
			assert.GreaterOrEqual(t, groups[i].Name, groups[i+1].Name, fmt.Sprintf("Groups not ordered by name descending at index %d: %s < %s", i, groups[i].Name, groups[i+1].Name))
		case "created_at":
			if dir == ascDir {
				assert.False(t, groups[i].CreatedAt.After(groups[i+1].CreatedAt), fmt.Sprintf("Groups not ordered by created_at ascending at index %d: %v > %v", i, groups[i].CreatedAt, groups[i+1].CreatedAt))
				continue
			}
			assert.False(t, groups[i].CreatedAt.Before(groups[i+1].CreatedAt), fmt.Sprintf("Groups not ordered by created_at descending at index %d: %v < %v", i, groups[i].CreatedAt, groups[i+1].CreatedAt))
		case "updated_at":
			if dir == ascDir {
				assert.False(t, groups[i].UpdatedAt.After(groups[i+1].UpdatedAt), fmt.Sprintf("Groups not ordered by updated_at ascending at index %d: %v > %v", i, groups[i].UpdatedAt, groups[i+1].UpdatedAt))
				continue
			}
			assert.False(t, groups[i].UpdatedAt.Before(groups[i+1].UpdatedAt), fmt.Sprintf("Groups not ordered by updated_at descending at index %d: %v < %v", i, groups[i].UpdatedAt, groups[i+1].UpdatedAt))
		}
	}
}
