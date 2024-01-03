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
	"github.com/absmach/magistrala/internal/groups/postgres"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	mggroups "github.com/absmach/magistrala/pkg/groups"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	namegen    = namegenerator.NewNameGenerator()
	invalidID  = strings.Repeat("a", 37)
	validGroup = mggroups.Group{
		ID:          testsutil.GenerateUUID(&testing.T{}),
		Owner:       testsutil.GenerateUUID(&testing.T{}),
		Name:        namegen.Generate(),
		Description: strings.Repeat("a", 64),
		Metadata:    map[string]interface{}{"key": "value"},
		CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
		Status:      clients.EnabledStatus,
	}
)

func TestSave(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM groups")
		require.Nil(t, err, fmt.Sprintf("clean groups unexpected error: %s", err))
	})

	repo := postgres.New(database)

	cases := []struct {
		desc  string
		group mggroups.Group
		err   error
	}{
		{
			desc:  "add new group successfully",
			group: validGroup,
			err:   nil,
		},
		{
			desc:  "add duplicate group",
			group: validGroup,
			err:   repoerr.ErrConflict,
		},
		{
			desc: "add group with invalid ID",
			group: mggroups.Group{
				ID:          invalidID,
				Owner:       testsutil.GenerateUUID(t),
				Name:        namegen.Generate(),
				Description: strings.Repeat("a", 64),
				Metadata:    map[string]interface{}{"key": "value"},
				CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
				Status:      clients.EnabledStatus,
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "add group with invalid owner",
			group: mggroups.Group{
				ID:          testsutil.GenerateUUID(t),
				Owner:       invalidID,
				Name:        namegen.Generate(),
				Description: strings.Repeat("a", 64),
				Metadata:    map[string]interface{}{"key": "value"},
				CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
				Status:      clients.EnabledStatus,
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "add group with invalid parent",
			group: mggroups.Group{
				ID:          testsutil.GenerateUUID(t),
				Parent:      invalidID,
				Name:        namegen.Generate(),
				Description: strings.Repeat("a", 64),
				Metadata:    map[string]interface{}{"key": "value"},
				CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
				Status:      clients.EnabledStatus,
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "add group with invalid name",
			group: mggroups.Group{
				ID:          testsutil.GenerateUUID(t),
				Owner:       testsutil.GenerateUUID(t),
				Name:        strings.Repeat("a", 1025),
				Description: strings.Repeat("a", 64),
				Metadata:    map[string]interface{}{"key": "value"},
				CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
				Status:      clients.EnabledStatus,
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "add group with invalid description",
			group: mggroups.Group{
				ID:          testsutil.GenerateUUID(t),
				Owner:       testsutil.GenerateUUID(t),
				Name:        namegen.Generate(),
				Description: strings.Repeat("a", 1025),
				Metadata:    map[string]interface{}{"key": "value"},
				CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
				Status:      clients.EnabledStatus,
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "add group with invalid metadata",
			group: mggroups.Group{
				ID:          testsutil.GenerateUUID(t),
				Owner:       testsutil.GenerateUUID(t),
				Name:        namegen.Generate(),
				Description: strings.Repeat("a", 64),
				Metadata: map[string]interface{}{
					"key": make(chan int),
				},
				CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
				Status:    clients.EnabledStatus,
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "add group with empty owner",
			group: mggroups.Group{
				ID:          testsutil.GenerateUUID(t),
				Name:        namegen.Generate(),
				Description: strings.Repeat("a", 64),
				Metadata:    map[string]interface{}{"key": "value"},
				CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
				Status:      clients.EnabledStatus,
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "add group with empty name",
			group: mggroups.Group{
				ID:          testsutil.GenerateUUID(t),
				Owner:       testsutil.GenerateUUID(t),
				Description: strings.Repeat("a", 64),
				Metadata:    map[string]interface{}{"key": "value"},
				CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
				Status:      clients.EnabledStatus,
			},
			err: repoerr.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		switch group, err := repo.Save(context.Background(), tc.group); {
		case err == nil:
			assert.Nil(t, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.group, group, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group, group))
		default:
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		}
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
		desc  string
		group mggroups.Group
		err   error
	}{
		{
			desc: "update group successfully",
			group: mggroups.Group{
				ID:          group.ID,
				Name:        namegen.Generate(),
				Description: strings.Repeat("a", 64),
				Metadata:    map[string]interface{}{"key": "value"},
				UpdatedAt:   time.Now().UTC().Truncate(time.Microsecond),
				UpdatedBy:   testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc: "update group name",
			group: mggroups.Group{
				ID:        group.ID,
				Name:      namegen.Generate(),
				UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc: "update group description",
			group: mggroups.Group{
				ID:          group.ID,
				Description: strings.Repeat("a", 64),
				UpdatedAt:   time.Now().UTC().Truncate(time.Microsecond),
				UpdatedBy:   testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc: "update group metadata",
			group: mggroups.Group{
				ID:        group.ID,
				Metadata:  map[string]interface{}{"key": "value"},
				UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc: "update group with invalid ID",
			group: mggroups.Group{
				ID:          testsutil.GenerateUUID(t),
				Name:        namegen.Generate(),
				Description: strings.Repeat("a", 64),
				Metadata:    map[string]interface{}{"key": "value"},
				UpdatedAt:   time.Now().UTC().Truncate(time.Microsecond),
				UpdatedBy:   testsutil.GenerateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "update group with empty ID",
			group: mggroups.Group{
				Name:        namegen.Generate(),
				Description: strings.Repeat("a", 64),
				Metadata:    map[string]interface{}{"key": "value"},
				UpdatedAt:   time.Now().UTC().Truncate(time.Microsecond),
				UpdatedBy:   testsutil.GenerateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		switch group, err := repo.Update(context.Background(), tc.group); {
		case err == nil:
			assert.Nil(t, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.group.ID, group.ID, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group.ID, group.ID))
			assert.Equal(t, tc.group.UpdatedAt, group.UpdatedAt, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group.UpdatedAt, group.UpdatedAt))
			assert.Equal(t, tc.group.UpdatedBy, group.UpdatedBy, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group.UpdatedBy, group.UpdatedBy))
		default:
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		}
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
		group mggroups.Group
		err   error
	}{
		{
			desc: "change status group successfully",
			group: mggroups.Group{
				ID:        group.ID,
				Status:    clients.DisabledStatus,
				UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc: "change status group with invalid ID",
			group: mggroups.Group{
				ID:        testsutil.GenerateUUID(t),
				Status:    clients.DisabledStatus,
				UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "change status group with empty ID",
			group: mggroups.Group{
				Status:    clients.DisabledStatus,
				UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		switch group, err := repo.ChangeStatus(context.Background(), tc.group); {
		case err == nil:
			assert.Nil(t, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.group.ID, group.ID, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group.ID, group.ID))
			assert.Equal(t, tc.group.UpdatedAt, group.UpdatedAt, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group.UpdatedAt, group.UpdatedAt))
			assert.Equal(t, tc.group.UpdatedBy, group.UpdatedBy, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group.UpdatedBy, group.UpdatedBy))
		default:
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		}
	}
}

func TestRetrieveByID(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM groups")
		require.Nil(t, err, fmt.Sprintf("clean groups unexpected error: %s", err))
	})

	repo := postgres.New(database)

	group, err := repo.Save(context.Background(), validGroup)
	require.Nil(t, err, fmt.Sprintf("save group unexpected error: %s", err))

	cases := []struct {
		desc  string
		id    string
		group mggroups.Group
		err   error
	}{
		{
			desc:  "retrieve group by id successfully",
			id:    group.ID,
			group: validGroup,
			err:   nil,
		},
		{
			desc:  "retrieve group by id with invalid ID",
			id:    invalidID,
			group: mggroups.Group{},
			err:   repoerr.ErrNotFound,
		},
		{
			desc:  "retrieve group by id with empty ID",
			id:    "",
			group: mggroups.Group{},
			err:   repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		switch group, err := repo.RetrieveByID(context.Background(), tc.id); {
		case err == nil:
			assert.Nil(t, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.group, group, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group, group))
		default:
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		}
	}
}

func TestRetrieveAll(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM groups")
		require.Nil(t, err, fmt.Sprintf("clean groups unexpected error: %s", err))
	})

	repo := postgres.New(database)
	num := 200

	var items []mggroups.Group
	parentID := ""
	for i := 0; i < num; i++ {
		name := namegen.Generate()
		group := mggroups.Group{
			ID:          testsutil.GenerateUUID(t),
			Owner:       testsutil.GenerateUUID(t),
			Parent:      parentID,
			Name:        name,
			Description: strings.Repeat("a", 64),
			Metadata:    map[string]interface{}{"name": name},
			CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
			Status:      clients.EnabledStatus,
		}
		_, err := repo.Save(context.Background(), group)
		require.Nil(t, err, fmt.Sprintf("create invitation unexpected error: %s", err))
		items = append(items, group)
		parentID = group.ID
	}

	cases := []struct {
		desc     string
		page     mggroups.Page
		response mggroups.Page
		err      error
	}{
		{
			desc: "retrieve groups successfully",
			page: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset: 0,
					Limit:  10,
				},
			},
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
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
			page: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset: 10,
					Limit:  10,
				},
			},
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
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
			page: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset: 0,
					Limit:  50,
				},
			},
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
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
			page: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset: 50,
					Limit:  50,
				},
			},
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
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
			page: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset: 1000,
					Limit:  50,
				},
			},
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Total:  uint64(num),
					Offset: 1000,
					Limit:  50,
				},
				Groups: []mggroups.Group(nil),
			},
			err: nil,
		},
		{
			desc: "retrieve groups with offset and limit out of range",
			page: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset: 170,
					Limit:  50,
				},
			},
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
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
			page: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset: 0,
					Limit:  1000,
				},
			},
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
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
			page: mggroups.Page{},
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Total:  uint64(num),
					Offset: 0,
					Limit:  0,
				},
				Groups: []mggroups.Group(nil),
			},
			err: nil,
		},
		{
			desc: "retrieve groups with name",
			page: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset: 0,
					Limit:  10,
					Name:   items[0].Name,
				},
			},
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Groups: []mggroups.Group{items[0]},
			},
			err: nil,
		},
		{
			desc: "retrieve groups with owner",
			page: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset:  0,
					Limit:   10,
					OwnerID: items[0].Owner,
				},
			},
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Groups: []mggroups.Group{items[0]},
			},
			err: nil,
		},
		{
			desc: "retrieve groups with metadata",
			page: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset:   0,
					Limit:    10,
					Metadata: items[0].Metadata,
				},
			},
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Groups: []mggroups.Group{items[0]},
			},
			err: nil,
		},
		{
			desc: "retrieve groups with invalid metadata",
			page: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset: 0,
					Limit:  10,
					Metadata: map[string]interface{}{
						"key": make(chan int),
					},
				},
			},
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
				Groups: []mggroups.Group(nil),
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "retrieve parent groups",
			page: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset: 0,
					Limit:  uint64(num),
				},
				ID:        items[5].ID,
				Direction: 1,
			},
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Total:  uint64(num),
					Offset: 0,
					Limit:  uint64(num),
				},
				Groups: items[:6],
			},
			err: nil,
		},
		{
			desc: "retrieve children groups",
			page: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset: 0,
					Limit:  uint64(num),
				},
				ID:        items[150].ID,
				Direction: -1,
			},
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Total:  uint64(num),
					Offset: 0,
					Limit:  uint64(num),
				},
				Groups: items[150:],
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		switch groups, err := repo.RetrieveAll(context.Background(), tc.page); {
		case err == nil:
			assert.Nil(t, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.response.Total, groups.Total, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Total, groups.Total))
			assert.Equal(t, tc.response.Limit, groups.Limit, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Limit, groups.Limit))
			assert.Equal(t, tc.response.Offset, groups.Offset, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Offset, groups.Offset))
			for i := range tc.response.Groups {
				tc.response.Groups[i].Level = groups.Groups[i].Level
				tc.response.Groups[i].Path = groups.Groups[i].Path
			}
			assert.ElementsMatch(t, groups.Groups, tc.response.Groups, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, tc.response.Groups, groups.Groups))
		default:
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		}
	}
}

func TestRetrieveByIDs(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM groups")
		require.Nil(t, err, fmt.Sprintf("clean groups unexpected error: %s", err))
	})

	repo := postgres.New(database)
	num := 200

	var items []mggroups.Group
	parentID := ""
	for i := 0; i < num; i++ {
		name := namegen.Generate()
		group := mggroups.Group{
			ID:          testsutil.GenerateUUID(t),
			Owner:       testsutil.GenerateUUID(t),
			Parent:      parentID,
			Name:        name,
			Description: strings.Repeat("a", 64),
			Metadata:    map[string]interface{}{"name": name},
			CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
			Status:      clients.EnabledStatus,
		}
		_, err := repo.Save(context.Background(), group)
		require.Nil(t, err, fmt.Sprintf("create invitation unexpected error: %s", err))
		items = append(items, group)
		parentID = group.ID
	}

	cases := []struct {
		desc     string
		page     mggroups.Page
		ids      []string
		response mggroups.Page
		err      error
	}{
		{
			desc: "retrieve groups successfully",
			page: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset: 0,
					Limit:  10,
				},
			},
			ids: getIDs(items[0:3]),
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
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
			page: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset: 0,
					Limit:  10,
				},
			},
			ids: []string{},
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset: 0,
					Limit:  10,
				},
				Groups: []mggroups.Group(nil),
			},
			err: nil,
		},
		{
			desc: "retrieve groups with empty ids but with owner",
			page: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset:  0,
					Limit:   10,
					OwnerID: items[0].Owner,
				},
			},
			ids: []string{},
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Groups: []mggroups.Group{items[0]},
			},
			err: nil,
		},
		{
			desc: "retrieve groups with offset",
			page: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset: 10,
					Limit:  10,
				},
			},
			ids: getIDs(items[0:20]),
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
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
			page: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset: 1000,
					Limit:  50,
				},
			},
			ids: getIDs(items[0:20]),
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Total:  20,
					Offset: 1000,
					Limit:  50,
				},
				Groups: []mggroups.Group(nil),
			},
			err: nil,
		},
		{
			desc: "retrieve groups with offset and limit out of range",
			page: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset: 15,
					Limit:  10,
				},
			},
			ids: getIDs(items[0:20]),
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
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
			page: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset: 0,
					Limit:  1000,
				},
			},
			ids: getIDs(items[0:20]),
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
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
			page: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset: 0,
					Limit:  10,
					Name:   items[0].Name,
				},
			},
			ids: getIDs(items[0:20]),
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Groups: []mggroups.Group{items[0]},
			},
			err: nil,
		},
		{
			desc: "retrieve groups with owner",
			page: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset:  0,
					Limit:   10,
					OwnerID: items[0].Owner,
				},
			},
			ids: getIDs(items[0:20]),
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Groups: []mggroups.Group{items[0]},
			},
			err: nil,
		},
		{
			desc: "retrieve groups with metadata",
			page: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset:   0,
					Limit:    10,
					Metadata: items[0].Metadata,
				},
			},
			ids: getIDs(items[0:20]),
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Groups: []mggroups.Group{items[0]},
			},
			err: nil,
		},
		{
			desc: "retrieve groups with invalid metadata",
			page: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset: 0,
					Limit:  10,
					Metadata: map[string]interface{}{
						"key": make(chan int),
					},
				},
			},
			ids: getIDs(items[0:20]),
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
				Groups: []mggroups.Group(nil),
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "retrieve parent groups",
			page: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset: 0,
					Limit:  uint64(num),
				},
				ID:        items[5].ID,
				Direction: 1,
			},
			ids: getIDs(items[0:20]),
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Total:  20,
					Offset: 0,
					Limit:  uint64(num),
				},
				Groups: items[:6],
			},
			err: nil,
		},
		{
			desc: "retrieve children groups",
			page: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Offset: 0,
					Limit:  uint64(num),
				},
				ID:        items[15].ID,
				Direction: -1,
			},
			ids: getIDs(items[0:20]),
			response: mggroups.Page{
				PageMeta: mggroups.PageMeta{
					Total:  20,
					Offset: 0,
					Limit:  uint64(num),
				},
				Groups: items[15:20],
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		switch groups, err := repo.RetrieveByIDs(context.Background(), tc.page, tc.ids...); {
		case err == nil:
			assert.Nil(t, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.response.Total, groups.Total, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Total, groups.Total))
			assert.Equal(t, tc.response.Limit, groups.Limit, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Limit, groups.Limit))
			assert.Equal(t, tc.response.Offset, groups.Offset, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Offset, groups.Offset))
			for i := range tc.response.Groups {
				tc.response.Groups[i].Level = groups.Groups[i].Level
				tc.response.Groups[i].Path = groups.Groups[i].Path
			}
			assert.ElementsMatch(t, groups.Groups, tc.response.Groups, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, tc.response.Groups, groups.Groups))
		default:
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		}
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
		switch err := repo.Delete(context.Background(), tc.id); {
		case err == nil:
			assert.Nil(t, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		default:
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		}
	}
}

func TestAssignParentGroup(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM groups")
		require.Nil(t, err, fmt.Sprintf("clean groups unexpected error: %s", err))
	})

	repo := postgres.New(database)

	num := 10

	var items []mggroups.Group
	parentID := ""
	for i := 0; i < num; i++ {
		name := namegen.Generate()
		group := mggroups.Group{
			ID:          testsutil.GenerateUUID(t),
			Owner:       testsutil.GenerateUUID(t),
			Parent:      parentID,
			Name:        name,
			Description: strings.Repeat("a", 64),
			Metadata:    map[string]interface{}{"name": name},
			CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
			Status:      clients.EnabledStatus,
		}
		_, err := repo.Save(context.Background(), group)
		require.Nil(t, err, fmt.Sprintf("create invitation unexpected error: %s", err))
		items = append(items, group)
		parentID = group.ID
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
			err:  repoerr.ErrCreateEntity,
		},
		{
			desc: "assign parent group with empty ID",
			id:   "",
			ids:  []string{items[1].ID, items[2].ID, items[3].ID, items[4].ID, items[5].ID},
			err:  repoerr.ErrCreateEntity,
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
		switch err := repo.AssignParentGroup(context.Background(), tc.id, tc.ids...); {
		case err == nil:
			assert.Nil(t, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		default:
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		}
	}
}

func TestUnassignParentGroup(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM groups")
		require.Nil(t, err, fmt.Sprintf("clean groups unexpected error: %s", err))
	})

	repo := postgres.New(database)

	num := 10

	var items []mggroups.Group
	parentID := ""
	for i := 0; i < num; i++ {
		name := namegen.Generate()
		group := mggroups.Group{
			ID:          testsutil.GenerateUUID(t),
			Owner:       testsutil.GenerateUUID(t),
			Parent:      parentID,
			Name:        name,
			Description: strings.Repeat("a", 64),
			Metadata:    map[string]interface{}{"name": name},
			CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
			Status:      clients.EnabledStatus,
		}
		_, err := repo.Save(context.Background(), group)
		require.Nil(t, err, fmt.Sprintf("create invitation unexpected error: %s", err))
		items = append(items, group)
		parentID = group.ID
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
			err:  repoerr.ErrCreateEntity,
		},
		{
			desc: "un-assign parent group with empty ID",
			id:   "",
			ids:  []string{items[1].ID, items[2].ID, items[3].ID, items[4].ID, items[5].ID},
			err:  repoerr.ErrCreateEntity,
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
		switch err := repo.UnassignParentGroup(context.Background(), tc.id, tc.ids...); {
		case err == nil:
			assert.Nil(t, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		default:
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		}
	}
}

func getIDs(groups []mggroups.Group) []string {
	var ids []string
	for _, group := range groups {
		ids = append(ids, group.ID)
	}

	return ids
}
