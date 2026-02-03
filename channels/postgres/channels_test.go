// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0x6flab/namegenerator"
	"github.com/absmach/supermq/channels"
	"github.com/absmach/supermq/channels/postgres"
	"github.com/absmach/supermq/domains"
	dpostgres "github.com/absmach/supermq/domains/postgres"
	"github.com/absmach/supermq/groups"
	gpostgres "github.com/absmach/supermq/groups/postgres"
	"github.com/absmach/supermq/internal/nullable"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/connections"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/roles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	namegen      = namegenerator.NewGenerator()
	invalidID    = strings.Repeat("a", 37)
	validChannel = channels.Channel{
		ID:              testsutil.GenerateUUID(&testing.T{}),
		Domain:          testsutil.GenerateUUID(&testing.T{}),
		ParentGroup:     testsutil.GenerateUUID(&testing.T{}),
		Name:            namegen.Generate(),
		Route:           testsutil.GenerateUUID(&testing.T{}),
		Tags:            []string{"tag1", "tag2"},
		Metadata:        map[string]any{"key": "value"},
		CreatedAt:       time.Now().UTC().Truncate(time.Microsecond),
		Status:          channels.EnabledStatus,
		ConnectionTypes: []connections.ConnType{},
	}
	validConnection = channels.Connection{
		ClientID:  testsutil.GenerateUUID(&testing.T{}),
		ChannelID: validChannel.ID,
		DomainID:  validChannel.Domain,
		Type:      connections.Publish,
	}
	validTimestamp    = time.Now().UTC().Truncate(time.Millisecond)
	directAccess      = "direct"
	directGroupAccess = "direct_group"
	domainAccess      = "domain"
	defOrder          = "created_at"
	ascDir            = "asc"
	descDir           = "desc"
	availableActions  = []string{
		"delete",
		"membership",
		"read",
		"update",
	}
	domainAvailableActions = []string{
		"channel_add_role_users",
		"channel_connect_to_client",
		"channel_create",
		"channel_delete",
		"channel_manage_role",
		"channel_read",
		"channel_remove_role_users",
		"channel_set_parent_group",
		"channel_update",
		"channel_view_role_users",
	}
	groupAvailableActions = []string{
		"channel_add_role_users",
		"channel_connect_to_client",
		"channel_create",
		"channel_delete",
		"channel_manage_role",
		"channel_read",
		"channel_remove_role_users",
		"channel_set_parent_group",
		"channel_update",
		"channel_view_role_users",
		"subgroup_channel_add_role_users",
		"subgroup_channel_connect_to_client",
		"subgroup_channel_create",
		"subgroup_channel_delete",
		"subgroup_channel_manage_role",
		"subgroup_channel_read",
		"subgroup_channel_remove_role_users",
		"subgroup_channel_set_parent_group",
		"subgroup_channel_update",
		"subgroup_channel_view_role_users",
		"subgroup_manage_role",
		"subgroup_membership",
		"subgroup_read",
		"subgroup_remove_role_users",
		"subgroup_set_child",
		"subgroup_set_parent",
		"subgroup_update",
	}
	errChannelExists = errors.New("channel id already exists")
)

func TestSave(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM channels")
		require.Nil(t, err, fmt.Sprintf("clean channels unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	duplicateChannelID := testsutil.GenerateUUID(t)

	duplicateRoute := testsutil.GenerateUUID(t)
	duplicateDomain := testsutil.GenerateUUID(t)

	duplicateChannel := channels.Channel{
		ID:     testsutil.GenerateUUID(t),
		Domain: duplicateDomain,
		Name:   namegen.Generate(),
		Route:  duplicateRoute,
	}

	_, err := repo.Save(context.Background(), duplicateChannel)
	require.Nil(t, err, fmt.Sprintf("save channel unexpected error: %s", err))

	cases := []struct {
		desc    string
		channel channels.Channel
		resp    []channels.Channel
		err     error
	}{
		{
			desc:    "add new channel successfully",
			channel: validChannel,
			resp:    []channels.Channel{validChannel},
			err:     nil,
		},
		{
			desc:    "add duplicate channel",
			channel: validChannel,
			resp:    []channels.Channel{},
			err:     errChannelExists,
		},
		{
			desc: "add channel with invalid ID",
			channel: channels.Channel{
				ID:        invalidID,
				Domain:    testsutil.GenerateUUID(t),
				Name:      namegen.Generate(),
				Metadata:  map[string]any{"key": "value"},
				CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
				Status:    channels.EnabledStatus,
			},
			resp: []channels.Channel{},
			err:  repoerr.ErrCreateEntity,
		},
		{
			desc: "add channel with invalid domain",
			channel: channels.Channel{
				ID:        testsutil.GenerateUUID(t),
				Domain:    invalidID,
				Name:      namegen.Generate(),
				Metadata:  map[string]any{"key": "value"},
				CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
				Status:    channels.EnabledStatus,
			},
			resp: []channels.Channel{},
			err:  repoerr.ErrCreateEntity,
		},
		{
			desc: "add channel with invalid name",
			channel: channels.Channel{
				ID:        testsutil.GenerateUUID(t),
				Domain:    testsutil.GenerateUUID(t),
				Name:      strings.Repeat("a", 1025),
				Metadata:  map[string]any{"key": "value"},
				CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
				Status:    channels.EnabledStatus,
			},
			resp: []channels.Channel{},
			err:  repoerr.ErrCreateEntity,
		},
		{
			desc: "add channel with invalid metadata",
			channel: channels.Channel{
				ID:     testsutil.GenerateUUID(t),
				Domain: testsutil.GenerateUUID(t),
				Name:   namegen.Generate(),
				Metadata: map[string]any{
					"key": make(chan int),
				},
				CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
				Status:    channels.EnabledStatus,
			},
			resp: []channels.Channel{},
			err:  repoerr.ErrCreateEntity,
		},
		{
			desc: "add channel with duplicate name",
			channel: channels.Channel{
				ID:        duplicateChannelID,
				Domain:    validChannel.Domain,
				Name:      validChannel.Name,
				Metadata:  map[string]any{"key": "different_value"},
				CreatedAt: validTimestamp,
				Status:    channels.EnabledStatus,
			},
			resp: []channels.Channel{
				{
					ID:              duplicateChannelID,
					Domain:          validChannel.Domain,
					Name:            validChannel.Name,
					Metadata:        map[string]any{"key": "different_value"},
					CreatedAt:       validTimestamp,
					Status:          channels.EnabledStatus,
					ConnectionTypes: []connections.ConnType{},
				},
			},
			err: nil,
		},
		{
			desc: "add channel with duplicate route",
			channel: channels.Channel{
				ID:     testsutil.GenerateUUID(t),
				Domain: duplicateDomain,
				Name:   namegen.Generate(),
				Route:  duplicateRoute,
			},
			resp: []channels.Channel{},
			err:  errors.ErrRouteNotAvailable,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			channels, err := repo.Save(context.Background(), tc.channel)
			assert.Equal(t, tc.resp, channels, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, channels))
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM channels")
		require.Nil(t, err, fmt.Sprintf("clean channels unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	_, err := repo.Save(context.Background(), validChannel)
	require.Nil(t, err, fmt.Sprintf("save channel unexpected error: %s", err))

	cases := []struct {
		desc    string
		update  string
		channel channels.Channel
		err     error
	}{
		{
			desc:   "update channel successfully",
			update: "all",
			channel: channels.Channel{
				ID:        validChannel.ID,
				Name:      namegen.Generate(),
				Route:     testsutil.GenerateUUID(t),
				Metadata:  map[string]any{"key": "value"},
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc:   "update channel name",
			update: "name",
			channel: channels.Channel{
				ID:        validChannel.ID,
				Name:      namegen.Generate(),
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc:   "update channel metadata",
			update: "metadata",
			channel: channels.Channel{
				ID:        validChannel.ID,
				Metadata:  map[string]any{"key1": "value1"},
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc:   "update channel with invalid ID",
			update: "all",
			channel: channels.Channel{
				ID:        testsutil.GenerateUUID(t),
				Name:      namegen.Generate(),
				Metadata:  map[string]any{"key": "value"},
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "update channel with empty ID",
			update: "all",
			channel: channels.Channel{
				Name:      namegen.Generate(),
				Metadata:  map[string]any{"key": "value"},
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			channel, err := repo.Update(context.Background(), tc.channel)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.channel.ID, channel.ID, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.channel.ID, channel.ID))
				assert.Equal(t, tc.channel.UpdatedAt, channel.UpdatedAt, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.channel.UpdatedAt, channel.UpdatedAt))
				assert.Equal(t, tc.channel.UpdatedBy, channel.UpdatedBy, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.channel.UpdatedBy, channel.UpdatedBy))
				switch tc.update {
				case "all":
					assert.Equal(t, tc.channel.Name, channel.Name, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.channel.Name, channel.Name))
					assert.Equal(t, tc.channel.Metadata, channel.Metadata, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.channel.Metadata, channel.Metadata))
				case "name":
					assert.Equal(t, tc.channel.Name, channel.Name, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.channel.Name, channel.Name))
				case "metadata":
					assert.Equal(t, tc.channel.Metadata, channel.Metadata, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.channel.Metadata, channel.Metadata))
				}
			}
		})
	}
}

func TestUpdateTags(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM channels")
		require.Nil(t, err, fmt.Sprintf("clean channels unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	_, err := repo.Save(context.Background(), validChannel)
	require.Nil(t, err, fmt.Sprintf("save channel unexpected error: %s", err))

	cases := []struct {
		desc    string
		channel channels.Channel
		err     error
	}{
		{
			desc: "update channel tags",
			channel: channels.Channel{
				ID:        validChannel.ID,
				Tags:      []string{"tag3", "tag4"},
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc: "update channel with invalid ID",
			channel: channels.Channel{
				ID:        testsutil.GenerateUUID(t),
				Tags:      []string{"tag3", "tag4"},
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "update channel with empty ID",
			channel: channels.Channel{
				Tags:      []string{"tag3", "tag4"},
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			channel, err := repo.UpdateTags(context.Background(), tc.channel)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.channel.ID, channel.ID, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.channel.ID, channel.ID))
				assert.Equal(t, tc.channel.UpdatedAt, channel.UpdatedAt, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.channel.UpdatedAt, channel.UpdatedAt))
				assert.Equal(t, tc.channel.UpdatedBy, channel.UpdatedBy, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.channel.UpdatedBy, channel.UpdatedBy))
				assert.Equal(t, tc.channel.Tags, channel.Tags, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.channel.Tags, channel.Tags))
			}
		})
	}
}

func TestChangeStatus(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM channels")
		require.Nil(t, err, fmt.Sprintf("clean channels unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	disabledChannel := validChannel
	disabledChannel.ID = testsutil.GenerateUUID(t)
	disabledChannel.Name = namegen.Generate()
	disabledChannel.Route = testsutil.GenerateUUID(t)
	disabledChannel.Status = channels.DisabledStatus

	_, err := repo.Save(context.Background(), validChannel, disabledChannel)
	require.Nil(t, err, fmt.Sprintf("save channel unexpected error: %s", err))

	cases := []struct {
		desc    string
		channel channels.Channel
		err     error
	}{
		{
			desc: "disable channel successfully",
			channel: channels.Channel{
				ID:        validChannel.ID,
				Status:    channels.DisabledStatus,
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc: "enable channel successfully",
			channel: channels.Channel{
				ID:        disabledChannel.ID,
				Status:    channels.EnabledStatus,
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc: "change status channel with invalid ID",
			channel: channels.Channel{
				ID:        testsutil.GenerateUUID(t),
				Status:    channels.DisabledStatus,
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "change status channel with empty ID",
			channel: channels.Channel{
				Status:    channels.DisabledStatus,
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			channel, err := repo.ChangeStatus(context.Background(), tc.channel)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.channel.ID, channel.ID, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.channel.ID, channel.ID))
				assert.Equal(t, tc.channel.UpdatedAt, channel.UpdatedAt, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.channel.UpdatedAt, channel.UpdatedAt))
				assert.Equal(t, tc.channel.UpdatedBy, channel.UpdatedBy, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.channel.UpdatedBy, channel.UpdatedBy))
				assert.Equal(t, tc.channel.Status, channel.Status, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.channel.Status, channel.Status))
			}
		})
	}
}

func TestRetrieveByID(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM channels")
		require.Nil(t, err, fmt.Sprintf("clean channels unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	_, err := repo.Save(context.Background(), validChannel)
	require.Nil(t, err, fmt.Sprintf("save channel unexpected error: %s", err))

	cases := []struct {
		desc string
		id   string
		resp channels.Channel
		err  error
	}{
		{
			desc: "retrieve channel by id successfully",
			id:   validChannel.ID,
			resp: validChannel,
			err:  nil,
		},
		{
			desc: "retrieve channel by id with invalid ID",
			id:   invalidID,
			err:  repoerr.ErrNotFound,
		},
		{
			desc: "retrieve channel by id with empty ID",
			id:   "",
			err:  repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			channel, err := repo.RetrieveByID(context.Background(), tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Nil(t, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
				assert.Equal(t, tc.resp, channel, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, channel))
			}
		})
	}
}

func TestRetrieveByRoute(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM channels")
		require.Nil(t, err, fmt.Sprintf("clean channels unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	_, err := repo.Save(context.Background(), validChannel)
	require.Nil(t, err, fmt.Sprintf("save channel unexpected error: %s", err))

	cases := []struct {
		desc     string
		route    string
		domainID string
		resp     channels.Channel
		err      error
	}{
		{
			desc:     "retrieve channel by route successfully",
			route:    validChannel.Route,
			domainID: validChannel.Domain,
			resp:     validChannel,
			err:      nil,
		},
		{
			desc:     "retrieve channel by id with invalid route",
			route:    "invalid-route",
			domainID: validChannel.Domain,
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve channel by id with empty route",
			route:    "",
			domainID: validChannel.Domain,
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve channel by id with invalid domain",
			route:    validChannel.Route,
			domainID: "invalid-domain",
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve channel by id with empty domain",
			route:    validChannel.Route,
			domainID: "",
			err:      repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			channel, err := repo.RetrieveByRoute(context.Background(), tc.route, tc.domainID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Nil(t, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
				assert.Equal(t, tc.resp, channel, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, channel))
			}
		})
	}
}

func TestRetrieveAll(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM channels")
		require.Nil(t, err, fmt.Sprintf("clean channels unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)
	num := 200

	var items []channels.Channel
	parentID := ""
	baseTime := time.Now().UTC().Truncate(time.Millisecond)
	for i := 0; i < num; i++ {
		name := namegen.Generate()
		channel := channels.Channel{
			ID:              testsutil.GenerateUUID(t),
			Domain:          testsutil.GenerateUUID(t),
			ParentGroup:     parentID,
			Name:            name,
			Route:           testsutil.GenerateUUID(t),
			Metadata:        map[string]any{"name": name},
			CreatedAt:       baseTime.Add(time.Duration(i) * time.Millisecond),
			UpdatedAt:       baseTime.Add(time.Duration(i) * time.Millisecond),
			Status:          channels.EnabledStatus,
			ConnectionTypes: []connections.ConnType{},
			Tags:            []string{"tag1", "tag2"},
		}
		if i%99 == 0 {
			channel.Tags = []string{"tag1", "tag3"}
		}
		_, err := repo.Save(context.Background(), channel)
		require.Nil(t, err, fmt.Sprintf("create channel unexpected error: %s", err))
		items = append(items, channel)
		if i%20 == 0 {
			parentID = channel.ID
		}
	}

	reversedChannels := []channels.Channel{}
	for i := len(items) - 1; i >= 0; i-- {
		reversedChannels = append(reversedChannels, items[i])
	}

	cases := []struct {
		desc     string
		page     channels.ChannelsPage
		response channels.ChannelsPage
		err      error
	}{
		{
			desc: "retrieve channels successfully",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset: 0,
					Limit:  10,
					Order:  defOrder,
					Dir:    ascDir,
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  uint64(num),
					Offset: 0,
					Limit:  10,
				},
				Channels: items[:10],
			},
			err: nil,
		},
		{
			desc: "retrieve channels with offset",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset: 10,
					Limit:  10,
					Order:  defOrder,
					Dir:    ascDir,
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  uint64(num),
					Offset: 10,
					Limit:  10,
				},
				Channels: items[10:20],
			},
			err: nil,
		},
		{
			desc: "retrieve channels with limit",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset: 0,
					Limit:  50,
					Order:  defOrder,
					Dir:    ascDir,
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  uint64(num),
					Offset: 0,
					Limit:  50,
				},
				Channels: items[:50],
			},
			err: nil,
		},
		{
			desc: "retrieve channels with offset and limit",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset: 50,
					Limit:  50,
					Order:  defOrder,
					Dir:    ascDir,
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  uint64(num),
					Offset: 50,
					Limit:  50,
				},
				Channels: items[50:100],
			},
			err: nil,
		},
		{
			desc: "retrieve channels with offset out of range",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset: 1000,
					Limit:  50,
					Order:  defOrder,
					Dir:    descDir,
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  uint64(num),
					Offset: 1000,
					Limit:  50,
				},
				Channels: []channels.Channel(nil),
			},
			err: nil,
		},
		{
			desc: "retrieve channels with offset and limit out of range",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset: 170,
					Limit:  50,
					Order:  defOrder,
					Dir:    ascDir,
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  uint64(num),
					Offset: 170,
					Limit:  50,
				},
				Channels: items[170:200],
			},
			err: nil,
		},
		{
			desc: "retrieve channels with limit out of range",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset: 0,
					Limit:  1000,
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  uint64(num),
					Offset: 0,
					Limit:  1000,
				},
				Channels: items,
			},
			err: nil,
		},
		{
			desc: "retrieve channels with empty page",
			page: channels.ChannelsPage{},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  uint64(num),
					Offset: 0,
					Limit:  0,
				},
				Channels: []channels.Channel(nil),
			},
			err: nil,
		},
		{
			desc: "retrieve channels with name",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset: 0,
					Limit:  10,
					Name:   items[0].Name,
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Channels: []channels.Channel{items[0]},
			},
			err: nil,
		},
		{
			desc: "retrieve channels with IDs filter",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset: 0,
					Limit:  10,
					IDs:    []string{items[0].ID, items[1].ID, items[2].ID},
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  3,
					Offset: 0,
					Limit:  10,
				},
				Channels: []channels.Channel{items[0], items[1], items[2]},
			},
			err: nil,
		},
		{
			desc: "retrieve channels with non-existing IDs",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset: 0,
					Limit:  10,
					IDs:    []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
				Channels: []channels.Channel(nil),
			},
			err: nil,
		},
		{
			desc: "retrieve channels with domain",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset: 0,
					Limit:  10,
					Domain: items[0].Domain,
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Channels: []channels.Channel{items[0]},
			},
			err: nil,
		},
		{
			desc: "retrieve channels with metadata",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset:   0,
					Limit:    10,
					Metadata: items[0].Metadata,
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Channels: []channels.Channel{items[0]},
			},
			err: nil,
		},
		{
			desc: "retrieve channels with invalid metadata",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset: 0,
					Limit:  10,
					Metadata: map[string]any{
						"key": make(chan int),
					},
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
				Channels: []channels.Channel(nil),
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "retrieve channels with id",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset: 0,
					Limit:  10,
					ID:     items[0].ID,
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Channels: []channels.Channel{items[0]},
			},
			err: nil,
		},
		{
			desc: "retrieve channels with wrong id",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset: 0,
					Limit:  10,
					ID:     "wrong",
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
				Channels: []channels.Channel(nil),
			},
			err: nil,
		},
		{
			desc: "retrieve channels with single tag",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset: 0,
					Limit:  uint64(num),
					Tags:   channels.TagsQuery{Elements: []string{"tag1"}, Operator: channels.OrOp},
					Status: channels.AllStatus,
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  200,
					Offset: 0,
					Limit:  uint64(num),
				},
				Channels: items,
			},
		},
		{
			desc: "retrieve channel with multiple tags and OR operator",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset: 0,
					Limit:  uint64(num),
					Tags:   channels.TagsQuery{Elements: []string{"tag2", "tag3"}, Operator: channels.OrOp},
					Status: channels.AllStatus,
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  200,
					Offset: 0,
					Limit:  uint64(num),
				},
				Channels: items,
			},
		},
		{
			desc: "retrieve channel with multiple tags and AND operator",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset: 0,
					Limit:  uint64(num),
					Tags:   channels.TagsQuery{Elements: []string{"tag1", "tag3"}, Operator: channels.AndOp},
					Status: channels.AllStatus,
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  3,
					Offset: 0,
					Limit:  uint64(num),
				},
				Channels: []channels.Channel{items[0], items[99], items[198]},
			},
		},
		{
			desc: "retrieve channel with invalid tags",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset: 0,
					Limit:  uint64(num),
					Tags:   channels.TagsQuery{Elements: []string{namegen.Generate(), namegen.Generate()}, Operator: channels.OrOp},
					Status: channels.AllStatus,
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  0,
					Offset: 0,
					Limit:  uint64(num),
				},
				Channels: []channels.Channel(nil),
			},
		},
		{
			desc: "retrieve channels with order by name ascending",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset: 0,
					Limit:  10,
					Order:  "name",
					Dir:    ascDir,
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  uint64(num),
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "retrieve channels with order by name descending",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset: 0,
					Limit:  10,
					Order:  "name",
					Dir:    descDir,
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  uint64(num),
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "retrieve channels with order by created_at ascending",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset: 0,
					Limit:  10,
					Order:  defOrder,
					Dir:    ascDir,
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  uint64(num),
					Offset: 0,
					Limit:  10,
				},
				Channels: items[:10],
			},
			err: nil,
		},
		{
			desc: "retrieve channels with order by created_at descending",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset: 0,
					Limit:  10,
					Order:  defOrder,
					Dir:    descDir,
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  uint64(num),
					Offset: 0,
					Limit:  10,
				},
				Channels: reversedChannels[:10],
			},
			err: nil,
		},
		{
			desc: "retrieve channels with order by updated_at ascending",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset: 0,
					Limit:  10,
					Order:  "updated_at",
					Dir:    ascDir,
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  uint64(num),
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "retrieve channels with order by updated_at descending",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset: 0,
					Limit:  10,
					Order:  "updated_at",
					Dir:    descDir,
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  uint64(num),
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "retrieve channels with created_from",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset:      0,
					Limit:       200,
					Order:       "created_at",
					Dir:         ascDir,
					CreatedFrom: baseTime.Add(100 * time.Millisecond),
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  100,
					Offset: 0,
					Limit:  200,
				},
				Channels: items[100:],
			},
			err: nil,
		},
		{
			desc: "retrieve channels with created_to",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset:    0,
					Limit:     200,
					Order:     "created_at",
					Dir:       ascDir,
					CreatedTo: baseTime.Add(99 * time.Millisecond),
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  100,
					Offset: 0,
					Limit:  200,
				},
				Channels: items[:100],
			},
			err: nil,
		},
		{
			desc: "retrieve channels with both created_from and created_to",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset:      0,
					Limit:       200,
					Order:       "created_at",
					Dir:         ascDir,
					CreatedFrom: baseTime.Add(50 * time.Millisecond),
					CreatedTo:   baseTime.Add(149 * time.Millisecond),
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  100,
					Offset: 0,
					Limit:  200,
				},
				Channels: items[50:150],
			},
			err: nil,
		},
		{
			desc: "retrieve channels with created_from returning no results",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset:      0,
					Limit:       10,
					Order:       "created_at",
					Dir:         ascDir,
					CreatedFrom: baseTime.Add(1000 * time.Millisecond),
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
				Channels: []channels.Channel{},
			},
			err: nil,
		},
		{
			desc: "retrieve channels with created_to returning no results",
			page: channels.ChannelsPage{
				Page: channels.Page{
					Offset:    0,
					Limit:     10,
					Order:     "created_at",
					Dir:       ascDir,
					CreatedTo: baseTime.Add(-1 * time.Millisecond),
				},
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
				Channels: []channels.Channel{},
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			switch channels, err := repo.RetrieveAll(context.Background(), tc.page.Page); {
			case err == nil:
				assert.Nil(t, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
				assert.Equal(t, tc.response.Total, channels.Total, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Total, channels.Total))
				assert.Equal(t, tc.response.Limit, channels.Limit, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Limit, channels.Limit))
				assert.Equal(t, tc.response.Offset, channels.Offset, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Offset, channels.Offset))
				if len(tc.response.Channels) > 0 {
					got := updateTimestamp(channels.Channels)
					resp := updateTimestamp(tc.response.Channels)
					assert.ElementsMatch(t, resp, got, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, resp, got))
				}
				verifyChannelsOrdering(t, channels.Channels, tc.page.Page.Order, tc.page.Page.Dir)
			default:
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			}
		})
	}
}

func TestRemove(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM channels")
		require.Nil(t, err, fmt.Sprintf("clean channels unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	_, err := repo.Save(context.Background(), validChannel)
	require.Nil(t, err, fmt.Sprintf("save channel unexpected error: %s", err))

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "remove channel successfully",
			id:   validChannel.ID,
			err:  nil,
		},
		{
			desc: "remove channel with invalid ID",
			id:   invalidID,
			err:  repoerr.ErrNotFound,
		},
		{
			desc: "remove channel with empty ID",
			id:   "",
			err:  repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.Remove(context.Background(), tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestSetParentGroup(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM channels")
		require.Nil(t, err, fmt.Sprintf("clean channels unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	_, err := repo.Save(context.Background(), validChannel)
	require.Nil(t, err, fmt.Sprintf("save channel unexpected error: %s", err))

	cases := []struct {
		desc          string
		id            string
		parentGroupID string
		err           error
	}{
		{
			desc:          "set parent group successfully",
			id:            validChannel.ID,
			parentGroupID: testsutil.GenerateUUID(t),
			err:           nil,
		},
		{
			desc:          "set parent group with invalid ID",
			id:            invalidID,
			parentGroupID: testsutil.GenerateUUID(t),
			err:           repoerr.ErrNotFound,
		},
		{
			desc:          "set parent group with empty ID",
			id:            "",
			parentGroupID: testsutil.GenerateUUID(t),
			err:           repoerr.ErrNotFound,
		},
		{
			desc:          "set parent group with invalid parent group ID",
			id:            validChannel.ID,
			parentGroupID: invalidID,
			err:           repoerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.SetParentGroup(context.Background(), channels.Channel{
				ID:          tc.id,
				ParentGroup: tc.parentGroupID,
			})
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				resp, err := repo.RetrieveByID(context.Background(), tc.id)
				require.Nil(t, err, fmt.Sprintf("retrieve channel unexpected error: %s", err))
				assert.Equal(t, tc.id, resp.ID, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.id, resp.ID))
				assert.Equal(t, tc.parentGroupID, resp.ParentGroup, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.parentGroupID, resp.ParentGroup))
			}
		})
	}
}

func TestRemoveParentGroup(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM channels")
		require.Nil(t, err, fmt.Sprintf("clean channels unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	_, err := repo.Save(context.Background(), validChannel)
	require.Nil(t, err, fmt.Sprintf("save channel unexpected error: %s", err))

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "remove parent group successfully",
			id:   validChannel.ID,
			err:  nil,
		},
		{
			desc: "remove parent group with invalid ID",
			id:   invalidID,
			err:  repoerr.ErrNotFound,
		},
		{
			desc: "remove parent group with empty ID",
			id:   "",
			err:  repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.RemoveParentGroup(context.Background(), channels.Channel{
				ID: tc.id,
			})
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				resp, err := repo.RetrieveByID(context.Background(), tc.id)
				require.Nil(t, err, fmt.Sprintf("retrieve channel unexpected error: %s", err))
				assert.Equal(t, tc.id, resp.ID, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.id, resp.ID))
				assert.Equal(t, "", resp.ParentGroup, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, "", resp.ParentGroup))
			}
		})
	}
}

func TestAddConnection(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM connections")
		require.Nil(t, err, fmt.Sprintf("clean connections unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM channels")
		require.Nil(t, err, fmt.Sprintf("clean channels unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	_, err := repo.Save(context.Background(), validChannel)
	require.Nil(t, err, fmt.Sprintf("save channel unexpected error: %s", err))

	cases := []struct {
		desc       string
		connection channels.Connection
		err        error
	}{
		{
			desc:       "add connection successfully",
			connection: validConnection,
			err:        nil,
		},
		{
			desc: "add connection with non-existent channel",
			connection: channels.Connection{
				ClientID:  testsutil.GenerateUUID(t),
				ChannelID: testsutil.GenerateUUID(t),
				DomainID:  validChannel.Domain,
				Type:      connections.Publish,
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "add connection with non-existent domain",
			connection: channels.Connection{
				ClientID:  testsutil.GenerateUUID(t),
				ChannelID: validChannel.ID,
				DomainID:  testsutil.GenerateUUID(t),
				Type:      connections.Publish,
			},
			err: repoerr.ErrCreateEntity,
		},

		{
			desc: "add connection with invalid client ID",
			connection: channels.Connection{
				ClientID:  invalidID,
				ChannelID: testsutil.GenerateUUID(t),
				DomainID:  testsutil.GenerateUUID(t),
				Type:      connections.Publish,
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "add connection with invalid channel ID",
			connection: channels.Connection{
				ClientID:  testsutil.GenerateUUID(t),
				ChannelID: invalidID,
				DomainID:  testsutil.GenerateUUID(t),
				Type:      connections.Publish,
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "add connection with invalid domain ID",
			connection: channels.Connection{
				ClientID:  testsutil.GenerateUUID(t),
				ChannelID: testsutil.GenerateUUID(t),
				DomainID:  invalidID,
				Type:      connections.Publish,
			},
			err: repoerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.AddConnections(context.Background(), []channels.Connection{tc.connection})
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestRemoveConnection(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM connections")
		require.Nil(t, err, fmt.Sprintf("clean connections unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM channels")
		require.Nil(t, err, fmt.Sprintf("clean channels unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	_, err := repo.Save(context.Background(), validChannel)
	require.Nil(t, err, fmt.Sprintf("save channel unexpected error: %s", err))

	err = repo.AddConnections(context.Background(), []channels.Connection{validConnection})
	require.Nil(t, err, fmt.Sprintf("add connection unexpected error: %s", err))

	conn1 := channels.Connection{
		ClientID:  testsutil.GenerateUUID(t),
		ChannelID: validChannel.ID,
		DomainID:  validChannel.Domain,
		Type:      connections.Publish,
	}
	conn2 := channels.Connection{
		ClientID:  testsutil.GenerateUUID(t),
		ChannelID: validChannel.ID,
		DomainID:  validChannel.Domain,
		Type:      connections.Subscribe,
	}
	err = repo.AddConnections(context.Background(), []channels.Connection{conn1, conn2})
	require.Nil(t, err, fmt.Sprintf("add connections unexpected error: %s", err))

	cases := []struct {
		desc        string
		connections []channels.Connection
		err         error
	}{
		{
			desc:        "remove connection successfully",
			connections: []channels.Connection{validConnection},
			err:         nil,
		},
		{
			desc: "remove connection with non-existent channel",
			connections: []channels.Connection{
				{
					ClientID:  testsutil.GenerateUUID(t),
					ChannelID: testsutil.GenerateUUID(t),
					DomainID:  validChannel.Domain,
					Type:      connections.Publish,
				},
			},
			err: nil,
		},
		{
			desc: "remove connection with non-existent domain",
			connections: []channels.Connection{
				{
					ClientID:  testsutil.GenerateUUID(t),
					ChannelID: validChannel.ID,
					DomainID:  testsutil.GenerateUUID(t),
					Type:      connections.Publish,
				},
			},
			err: nil,
		},
		{
			desc: "remove connection with non-existent client",
			connections: []channels.Connection{
				{
					ClientID:  testsutil.GenerateUUID(t),
					ChannelID: validChannel.ID,
					DomainID:  validChannel.Domain,
					Type:      connections.Publish,
				},
			},
			err: nil,
		},
		{
			desc: "remove connection with invalid type",
			connections: []channels.Connection{
				{
					ClientID:  validConnection.ClientID,
					ChannelID: validConnection.ChannelID,
					DomainID:  validConnection.DomainID,
					Type:      connections.Invalid,
				},
			},
			err: nil,
		},
		{
			desc:        "remove multiple connections",
			connections: []channels.Connection{conn1, conn2},
			err:         nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.RemoveConnections(context.Background(), tc.connections)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestCheckConnection(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM connections")
		require.Nil(t, err, fmt.Sprintf("clean connections unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM channels")
		require.Nil(t, err, fmt.Sprintf("clean channels unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	_, err := repo.Save(context.Background(), validChannel)
	require.Nil(t, err, fmt.Sprintf("save channel unexpected error: %s", err))

	err = repo.AddConnections(context.Background(), []channels.Connection{validConnection})
	require.Nil(t, err, fmt.Sprintf("add connection unexpected error: %s", err))

	cases := []struct {
		desc       string
		connection channels.Connection
		err        error
	}{
		{
			desc:       "check connection successfully",
			connection: validConnection,
			err:        nil,
		},
		{
			desc: "check connection with non-existent channel",
			connection: channels.Connection{
				ClientID:  testsutil.GenerateUUID(t),
				ChannelID: testsutil.GenerateUUID(t),
				DomainID:  validChannel.Domain,
				Type:      connections.Publish,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "check connection with non-existent domain",
			connection: channels.Connection{
				ClientID:  testsutil.GenerateUUID(t),
				ChannelID: validChannel.ID,
				DomainID:  testsutil.GenerateUUID(t),
				Type:      connections.Publish,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "check connection with non-existent client",
			connection: channels.Connection{
				ClientID:  testsutil.GenerateUUID(t),
				ChannelID: validChannel.ID,
				DomainID:  validChannel.Domain,
				Type:      connections.Publish,
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.CheckConnection(context.Background(), tc.connection)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestClientAuthorize(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM connections")
		require.Nil(t, err, fmt.Sprintf("clean connections unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM channels")
		require.Nil(t, err, fmt.Sprintf("clean channels unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	_, err := repo.Save(context.Background(), validChannel)
	require.Nil(t, err, fmt.Sprintf("save channel unexpected error: %s", err))

	err = repo.AddConnections(context.Background(), []channels.Connection{validConnection})
	require.Nil(t, err, fmt.Sprintf("add connection unexpected error: %s", err))

	cases := []struct {
		desc       string
		connection channels.Connection
		err        error
	}{
		{
			desc:       "authorize successfully",
			connection: validConnection,
			err:        nil,
		},
		{
			desc: "authorize with  non-existent channel",
			connection: channels.Connection{
				ClientID:  testsutil.GenerateUUID(t),
				ChannelID: testsutil.GenerateUUID(t),
				DomainID:  validChannel.Domain,
				Type:      connections.Publish,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "authorize with non-existent client",
			connection: channels.Connection{
				ClientID:  testsutil.GenerateUUID(t),
				ChannelID: validChannel.ID,
				DomainID:  validChannel.Domain,
				Type:      connections.Publish,
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.ClientAuthorize(context.Background(), tc.connection)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestChannelConnectionsCount(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM connections")
		require.Nil(t, err, fmt.Sprintf("clean connections unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM channels")
		require.Nil(t, err, fmt.Sprintf("clean channels unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	rConnections := []channels.Connection{}
	for i := 0; i < 10; i++ {
		connection := channels.Connection{
			ClientID:  testsutil.GenerateUUID(t),
			ChannelID: validChannel.ID,
			DomainID:  validChannel.Domain,
			Type:      connections.Publish,
		}
		rConnections = append(rConnections, connection)
	}

	_, err := repo.Save(context.Background(), validChannel)
	require.Nil(t, err, fmt.Sprintf("save channel unexpected error: %s", err))

	err = repo.AddConnections(context.Background(), rConnections)
	require.Nil(t, err, fmt.Sprintf("add connection unexpected error: %s", err))

	cases := []struct {
		desc      string
		channelID string
		count     uint64
		err       error
	}{
		{
			desc:      "get channel connections count successfully",
			channelID: validChannel.ID,
			count:     10,
			err:       nil,
		},
		{
			desc:      "get channel connections count with non-existent channel",
			channelID: testsutil.GenerateUUID(t),
			count:     0,
			err:       nil,
		},
		{
			desc:      "get channel connections count with empty channel ID",
			channelID: "",
			count:     0,
			err:       nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			count, err := repo.ChannelConnectionsCount(context.Background(), tc.channelID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.count, count, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.count, count))
		})
	}
}

func TestDoesChannelHaveConnections(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM connections")
		require.Nil(t, err, fmt.Sprintf("clean connections unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM channels")
		require.Nil(t, err, fmt.Sprintf("clean channels unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	_, err := repo.Save(context.Background(), validChannel)
	require.Nil(t, err, fmt.Sprintf("save channel unexpected error: %s", err))

	err = repo.AddConnections(context.Background(), []channels.Connection{validConnection})
	require.Nil(t, err, fmt.Sprintf("add connection unexpected error: %s", err))

	cases := []struct {
		desc      string
		channelID string
		has       bool
		err       error
	}{
		{
			desc:      "check if channel has connections successfully",
			channelID: validChannel.ID,
			has:       true,
			err:       nil,
		},
		{
			desc:      "check if channel has connections with non-existent channel",
			channelID: testsutil.GenerateUUID(t),
			has:       false,
			err:       nil,
		},
		{
			desc:      "check if channel has connections with empty channel ID",
			channelID: "",
			has:       false,
			err:       nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			has, err := repo.DoesChannelHaveConnections(context.Background(), tc.channelID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.has, has, fmt.Sprintf("%s: expected %t got %t\n", tc.desc, tc.has, has))
		})
	}
}

func TestRemoveClientConnections(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM connections")
		require.Nil(t, err, fmt.Sprintf("clean connections unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM channels")
		require.Nil(t, err, fmt.Sprintf("clean channels unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	_, err := repo.Save(context.Background(), validChannel)
	require.Nil(t, err, fmt.Sprintf("save channel unexpected error: %s", err))

	err = repo.AddConnections(context.Background(), []channels.Connection{validConnection})
	require.Nil(t, err, fmt.Sprintf("add connection unexpected error: %s", err))

	cases := []struct {
		desc     string
		clientID string
		err      error
	}{
		{
			desc:     "remove client connections successfully",
			clientID: validConnection.ClientID,
			err:      nil,
		},
		{
			desc:     "remove client connections with non-existent client",
			clientID: testsutil.GenerateUUID(t),
			err:      nil,
		},
		{
			desc:     "remove client connections with empty client ID",
			clientID: "",
			err:      nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.RemoveClientConnections(context.Background(), tc.clientID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestRemoveChannelConnections(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM connections")
		require.Nil(t, err, fmt.Sprintf("clean connections unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM channels")
		require.Nil(t, err, fmt.Sprintf("clean channels unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	_, err := repo.Save(context.Background(), validChannel)
	require.Nil(t, err, fmt.Sprintf("save channel unexpected error: %s", err))

	err = repo.AddConnections(context.Background(), []channels.Connection{validConnection})
	require.Nil(t, err, fmt.Sprintf("add connection unexpected error: %s", err))

	cases := []struct {
		desc      string
		channelID string
		err       error
	}{
		{
			desc:      "remove channel connections successfully",
			channelID: validConnection.ChannelID,
			err:       nil,
		},
		{
			desc:      "remove channel connections with non-existent channel",
			channelID: testsutil.GenerateUUID(t),
			err:       nil,
		},
		{
			desc:      "remove channel connections with empty channel ID",
			channelID: "",
			err:       nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.RemoveChannelConnections(context.Background(), tc.channelID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestRetrieveParentGroupChannels(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM channels")
		require.Nil(t, err, fmt.Sprintf("clean channels unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	var items []channels.Channel
	parentID := testsutil.GenerateUUID(t)
	for i := 0; i < 10; i++ {
		name := namegen.Generate()
		channel := channels.Channel{
			ID:              testsutil.GenerateUUID(t),
			Domain:          testsutil.GenerateUUID(t),
			ParentGroup:     parentID,
			Name:            name,
			Metadata:        map[string]any{"name": name},
			CreatedAt:       time.Now().UTC().Truncate(time.Microsecond),
			Status:          channels.EnabledStatus,
			ConnectionTypes: []connections.ConnType{},
		}
		items = append(items, channel)
	}

	_, err := repo.Save(context.Background(), items...)
	require.Nil(t, err, fmt.Sprintf("create channel unexpected error: %s", err))

	cases := []struct {
		desc          string
		parentGroupID string
		resp          []channels.Channel
		err           error
	}{
		{
			desc:          "retrieve parent group channels successfully",
			parentGroupID: parentID,
			resp:          items[:10],
			err:           nil,
		},
		{
			desc:          "retrieve parent group channels with non-existent channel",
			parentGroupID: testsutil.GenerateUUID(t),
			resp:          []channels.Channel(nil),
			err:           nil,
		},
		{
			desc:          "retrieve parent group channels with empty channel ID",
			parentGroupID: "",
			resp:          []channels.Channel(nil),
			err:           nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			channels, err := repo.RetrieveParentGroupChannels(context.Background(), tc.parentGroupID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				got := updateTimestamp(channels)
				resp := updateTimestamp(tc.resp)
				assert.Equal(t, len(tc.resp), len(channels), fmt.Sprintf("%s: expected %d got %d\n", tc.desc, len(tc.resp), len(channels)))
				assert.ElementsMatch(t, resp, got, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, resp, got))
			}
		})
	}
}

func TestUnsetParentGroupFromChannels(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM channels")
		require.Nil(t, err, fmt.Sprintf("clean channels unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	var items []channels.Channel
	parentID := testsutil.GenerateUUID(t)
	for i := 0; i < 10; i++ {
		name := namegen.Generate()
		channel := channels.Channel{
			ID:          testsutil.GenerateUUID(t),
			Domain:      testsutil.GenerateUUID(t),
			ParentGroup: parentID,
			Name:        name,
			Metadata:    map[string]any{"name": name},
			CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
			Status:      channels.EnabledStatus,
		}
		items = append(items, channel)
	}

	_, err := repo.Save(context.Background(), items...)
	require.Nil(t, err, fmt.Sprintf("create channel unexpected error: %s", err))

	cases := []struct {
		desc          string
		parentGroupID string
		err           error
	}{
		{
			desc:          "unset parent group from channels successfully",
			parentGroupID: parentID,
			err:           nil,
		},
		{
			desc:          "unset parent group from channels with non-existent id",
			parentGroupID: testsutil.GenerateUUID(t),
			err:           nil,
		},
		{
			desc:          "unset parent group from channels with empty channel ID",
			parentGroupID: "",
			err:           nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.UnsetParentGroupFromChannels(context.Background(), tc.parentGroupID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestRetrieveByIDWithRoles(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM channels")
		require.Nil(t, err, fmt.Sprintf("clean channels unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	nChannels := uint64(10)

	domainID := testsutil.GenerateUUID(t)
	userID := testsutil.GenerateUUID(t)
	expectedChannels := []channels.Channel{}
	for range nChannels {
		channel := channels.Channel{
			ID:     testsutil.GenerateUUID(t),
			Domain: domainID,
			Name:   namegen.Generate(),
			Route:  testsutil.GenerateUUID(t),
			Tags:   namegen.GenerateMultiple(5),
			Metadata: map[string]any{
				"department": namegen.Generate(),
			},
			Status:    channels.EnabledStatus,
			CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		}
		_, err := repo.Save(context.Background(), channel)
		require.Nil(t, err, fmt.Sprintf("add new channel: expected nil got %s\n", err))
		newRolesProvision := []roles.RoleProvision{
			{
				Role: roles.Role{
					ID:        testsutil.GenerateUUID(t) + "_" + channel.ID,
					Name:      "admin",
					EntityID:  channel.ID,
					CreatedAt: validTimestamp,
					CreatedBy: userID,
				},
				OptionalActions: availableActions,
				OptionalMembers: []string{userID},
			},
		}
		npr, err := repo.AddRoles(context.Background(), newRolesProvision)
		require.Nil(t, err, fmt.Sprintf("add roles unexpected error: %s", err))
		expectedChannel := channel
		expectedChannel.ConnectionTypes = []connections.ConnType{}
		expectedChannel.Roles = []roles.MemberRoleActions{
			{
				RoleID:     npr[0].Role.ID,
				RoleName:   npr[0].Role.Name,
				Actions:    npr[0].OptionalActions,
				AccessType: directAccess,
			},
		}
		expectedChannels = append(expectedChannels, expectedChannel)
	}

	cases := []struct {
		desc      string
		channelID string
		userID    string
		response  channels.Channel
		err       error
	}{
		{
			desc:      "retrieve channel with role successfully",
			channelID: expectedChannels[0].ID,
			userID:    userID,
			response:  expectedChannels[0],
			err:       nil,
		},
		{
			desc:      "retrieve another channel with role successfully",
			channelID: expectedChannels[1].ID,
			userID:    userID,
			response:  expectedChannels[1],
			err:       nil,
		},
		{
			desc:      "retrieve channel with invalid channel id",
			channelID: testsutil.GenerateUUID(t),
			userID:    userID,
			response:  channels.Channel{},
			err:       repoerr.ErrNotFound,
		},
		{
			desc:      "retrieve channel with empty channel id",
			channelID: "",
			userID:    userID,
			response:  channels.Channel{},
			err:       repoerr.ErrNotFound,
		},
		{
			desc:      "retrieve channel with invalid user id",
			channelID: expectedChannels[0].ID,
			userID:    testsutil.GenerateUUID(t),
			response:  channels.Channel{},
			err:       repoerr.ErrNotFound,
		},
		{
			desc:      "retrieve channel with empty user id",
			channelID: expectedChannels[0].ID,
			userID:    "",
			response:  channels.Channel{},
			err:       repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			channel, err := repo.RetrieveByIDWithRoles(context.Background(), tc.channelID, tc.userID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected %s to contain %s\n", err, tc.err))
			if err == nil {
				assert.Equal(t, tc.response, channel, fmt.Sprintf("expected %v got %v\n", tc.response, channel))
			}
		})
	}
}

func TestRetrieveUserChannels(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM channels")
		require.Nil(t, err, fmt.Sprintf("clean channels unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM groups")
		require.Nil(t, err, fmt.Sprintf("clean groups unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM domains")
		require.Nil(t, err, fmt.Sprintf("clean domains unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	nChannels := uint64(10)

	emptyGroupParam := ""
	userID := testsutil.GenerateUUID(t)
	domainMemberID := testsutil.GenerateUUID(t)
	groupMemberID := testsutil.GenerateUUID(t)
	clientID := testsutil.GenerateUUID(t)
	domain := generateDomain(t, userID, domainMemberID)
	group := generateGroup(t, userID, groupMemberID, domain.ID)
	groupChannel := channels.Channel{}
	parentGroupChannel := channels.Channel{}
	connectedChannel := channels.Channel{}
	directChannels := []channels.Channel{}
	domainChannels := []channels.Channel{}
	for i := range nChannels {
		channel := channels.Channel{
			ID:     testsutil.GenerateUUID(t),
			Domain: domain.ID,
			Name:   namegen.Generate(),
			Route:  testsutil.GenerateUUID(t),
			Tags:   namegen.GenerateMultiple(5),
			Metadata: map[string]any{
				"department": namegen.Generate(),
			},
			Status:    channels.EnabledStatus,
			CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		}
		if i == 1 {
			channel.ParentGroup = group.ID
		}
		_, err := repo.Save(context.Background(), channel)
		require.Nil(t, err, fmt.Sprintf("add new channel: expected nil got %s\n", err))
		newRolesProvision := []roles.RoleProvision{
			{
				Role: roles.Role{
					ID:        testsutil.GenerateUUID(t) + "_" + channel.ID,
					Name:      "admin",
					EntityID:  channel.ID,
					CreatedAt: validTimestamp,
					CreatedBy: userID,
				},
				OptionalActions: availableActions,
				OptionalMembers: []string{userID},
			},
		}
		npr, err := repo.AddRoles(context.Background(), newRolesProvision)
		require.Nil(t, err, fmt.Sprintf("add roles unexpected error: %s", err))
		directChannel := channel
		directChannel.RoleID = npr[0].Role.ID
		directChannel.RoleName = npr[0].Role.Name
		directChannel.AccessType = directAccess
		directChannel.AccessProviderRoleActions = []string{}
		if i == 1 {
			directChannel.ParentGroupPath = group.ID
		}
		directChannels = append(directChannels, directChannel)
		if i == 1 {
			parentGroupChannel = directChannel
			parentGroupChannel.ParentGroupPath = group.ID
			channel.ParentGroupPath = group.ID
			groupChannel = channel
			groupChannel.AccessType = directGroupAccess
			groupChannel.AccessProviderId = group.ID
			groupChannel.AccessProviderRoleId = group.Roles[0].RoleID
			groupChannel.AccessProviderRoleName = group.Roles[0].RoleName
			groupChannel.AccessProviderRoleActions = groupAvailableActions
		}
		if i == 2 {
			conn := channels.Connection{
				ClientID:  clientID,
				ChannelID: channel.ID,
				DomainID:  channel.Domain,
				Type:      connections.Publish,
			}
			err = repo.AddConnections(context.Background(), []channels.Connection{conn})
			assert.Nil(t, err, fmt.Sprintf("add connection unexpected error: %s", err))
			connectedChannel = channel
			connectedChannel.RoleID = npr[0].Role.ID
			connectedChannel.RoleName = npr[0].Role.Name
			connectedChannel.AccessType = directAccess
			connectedChannel.AccessProviderRoleActions = []string{}
			connectedChannel.ConnectionTypes = []connections.ConnType{connections.Publish}
		}
		domainChannel := channel
		domainChannel.AccessType = domainAccess
		domainChannel.AccessProviderId = domain.ID
		domainChannel.AccessProviderRoleId = domain.Roles[0].RoleID
		domainChannel.AccessProviderRoleName = domain.Roles[0].RoleName
		domainChannel.AccessProviderRoleActions = domainAvailableActions
		domainChannels = append(domainChannels, domainChannel)
	}

	cases := []struct {
		desc     string
		domainID string
		userID   string
		pm       channels.Page
		response channels.ChannelsPage
		err      error
	}{
		{
			desc:     "retrieve channels with empty page",
			domainID: domain.ID,
			userID:   userID,
			pm:       channels.Page{},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  10,
					Offset: 0,
					Limit:  0,
				},
				Channels: []channels.Channel(nil),
			},
		},
		{
			desc:     "retrieve channels with offset and limit",
			domainID: domain.ID,
			userID:   userID,
			pm: channels.Page{
				Offset: 5,
				Limit:  10,
				Status: channels.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  nChannels,
					Offset: 5,
					Limit:  10,
				},
				Channels: directChannels[5:10],
			},
		},
		{
			desc:     "retrieve channels with member id of parent group with direct group access",
			domainID: domain.ID,
			userID:   groupMemberID,
			pm: channels.Page{
				Offset: 0,
				Limit:  10,
				Status: channels.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Channels: []channels.Channel{groupChannel},
			},
		},
		{
			desc:     "retrieve channels with member id of domain with domain access",
			domainID: domain.ID,
			userID:   domainMemberID,
			pm: channels.Page{
				Offset: 0,
				Limit:  10,
				Status: channels.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  10,
					Offset: 0,
					Limit:  10,
				},
				Channels: domainChannels,
			},
		},
		{
			desc:     "retrieve channels connected to a client",
			domainID: domain.ID,
			userID:   userID,
			pm: channels.Page{
				Offset: 0,
				Limit:  10,
				Client: clientID,
				Status: channels.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Channels: []channels.Channel{connectedChannel},
			},
		},
		{
			desc:     "retrieve channels with offset out of range and limit",
			domainID: domain.ID,
			userID:   userID,
			pm: channels.Page{
				Offset: 1000,
				Limit:  50,
				Status: channels.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  nChannels,
					Offset: 1000,
					Limit:  50,
				},
				Channels: []channels.Channel(nil),
			},
		},
		{
			desc:     "retrieve channels with metadata",
			domainID: domain.ID,
			userID:   userID,
			pm: channels.Page{
				Offset:   0,
				Limit:    nChannels,
				Metadata: directChannels[0].Metadata,
				Status:   channels.AllStatus,
				Order:    defOrder,
				Dir:      ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  1,
					Offset: 0,
					Limit:  nChannels,
				},
				Channels: []channels.Channel{directChannels[0]},
			},
		},
		{
			desc:     "retrieve channels with wrong metadata",
			domainID: domain.ID,
			userID:   userID,
			pm: channels.Page{
				Offset: 0,
				Limit:  nChannels,
				Metadata: map[string]any{
					"faculty": namegen.Generate(),
				},
				Status: channels.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  0,
					Offset: 0,
					Limit:  nChannels,
				},
				Channels: []channels.Channel(nil),
			},
		},
		{
			desc:     "retrieve channels with invalid metadata",
			domainID: domain.ID,
			userID:   userID,
			pm: channels.Page{
				Offset: 0,
				Limit:  nChannels,
				Metadata: map[string]any{
					"faculty": make(chan int),
				},
				Status: channels.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  uint64(nChannels),
					Offset: 0,
					Limit:  nChannels,
				},
				Channels: []channels.Channel(nil),
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc:     "retrieve channels with name",
			domainID: domain.ID,
			userID:   userID,
			pm: channels.Page{
				Offset: 0,
				Limit:  nChannels,
				Name:   directChannels[0].Name,
				Status: channels.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  1,
					Offset: 0,
					Limit:  nChannels,
				},
				Channels: []channels.Channel{directChannels[0]},
			},
		},
		{
			desc:     "retrieve channels with wrong name",
			domainID: domain.ID,
			userID:   userID,
			pm: channels.Page{
				Offset: 0,
				Limit:  nChannels,
				Name:   namegen.Generate(),
				Status: channels.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  0,
					Offset: 0,
					Limit:  nChannels,
				},
				Channels: []channels.Channel(nil),
			},
		},
		{
			desc:     "retrieve channels with tag",
			domainID: domain.ID,
			userID:   userID,
			pm: channels.Page{
				Offset: 0,
				Limit:  nChannels,
				Tags:   channels.TagsQuery{Elements: []string{directChannels[0].Tags[0]}, Operator: channels.OrOp},
				Status: channels.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  1,
					Offset: 0,
					Limit:  uint64(nChannels),
				},
				Channels: []channels.Channel{directChannels[0]},
			},
		},
		{
			desc:     "retrieve channels with wrong tags",
			domainID: domain.ID,
			userID:   userID,
			pm: channels.Page{
				Offset: 0,
				Limit:  nChannels,
				Tags:   channels.TagsQuery{Elements: []string{namegen.Generate()}, Operator: channels.OrOp},
				Status: channels.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  0,
					Offset: 0,
					Limit:  nChannels,
				},
				Channels: []channels.Channel(nil),
			},
		},
		{
			desc:     "retrieve channels with multiple parameters",
			domainID: domain.ID,
			userID:   userID,
			pm: channels.Page{
				Offset:   0,
				Limit:    nChannels,
				Metadata: directChannels[0].Metadata,
				Name:     directChannels[0].Name,
				Tags:     channels.TagsQuery{Elements: []string{directChannels[0].Tags[0]}, Operator: channels.OrOp},
				Status:   channels.AllStatus,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  1,
					Offset: 0,
					Limit:  nChannels,
				},
				Channels: []channels.Channel{directChannels[0]},
			},
		},
		{
			desc:     "retrieve channels with id",
			domainID: domain.ID,
			userID:   userID,
			pm: channels.Page{
				Offset: 0,
				Limit:  nChannels,
				ID:     directChannels[0].ID,
				Status: channels.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  1,
					Offset: 0,
					Limit:  nChannels,
				},
				Channels: []channels.Channel{directChannels[0]},
			},
		},
		{
			desc:     "retrieve channels with wrong id",
			domainID: domain.ID,
			userID:   userID,
			pm: channels.Page{
				Offset: 0,
				Limit:  nChannels,
				ID:     testsutil.GenerateUUID(t),
				Status: channels.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  0,
					Offset: 0,
					Limit:  nChannels,
				},
				Channels: []channels.Channel(nil),
			},
		},
		{
			desc:     "retrieve channels with wrong domain id",
			domainID: testsutil.GenerateUUID(t),
			userID:   userID,
			pm: channels.Page{
				Offset: 0,
				Limit:  nChannels,
				Status: channels.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  0,
					Offset: 0,
					Limit:  nChannels,
				},
				Channels: []channels.Channel(nil),
			},
		},
		{
			desc:     "retrieve channels with wrong user id",
			domainID: domain.ID,
			userID:   testsutil.GenerateUUID(t),
			pm: channels.Page{
				Offset: 0,
				Limit:  nChannels,
				Status: channels.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  0,
					Offset: 0,
					Limit:  nChannels,
				},
				Channels: []channels.Channel(nil),
			},
		},
		{
			desc:     "retrieve channels with parent group",
			domainID: domain.ID,
			userID:   userID,
			pm: channels.Page{
				Offset: 0,
				Limit:  nChannels,
				Group: nullable.Value[string]{
					Value: group.ID,
					Valid: true,
				},
				Status: channels.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  1,
					Offset: 0,
					Limit:  nChannels,
				},
				Channels: []channels.Channel{parentGroupChannel},
			},
			err: nil,
		},
		{
			desc:     "retrieve channels with no parent group",
			domainID: domain.ID,
			userID:   userID,
			pm: channels.Page{
				Offset: 0,
				Limit:  nChannels,
				Group: nullable.Value[string]{
					Value: emptyGroupParam,
					Valid: true,
				},
				Status: channels.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  0,
					Offset: 0,
					Limit:  nChannels,
				},
				Channels: []channels.Channel{},
			},
		},
		{
			desc:     "retrieve channels with access type",
			domainID: domain.ID,
			userID:   domainMemberID,
			pm: channels.Page{
				Offset:     0,
				Limit:      10,
				AccessType: domainAccess,
				Status:     channels.AllStatus,
				Order:      defOrder,
				Dir:        ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  10,
					Offset: 0,
					Limit:  10,
				},
				Channels: domainChannels,
			},
		},
		{
			desc:     "retrieve channels with wrong access type",
			domainID: domain.ID,
			userID:   userID,
			pm: channels.Page{
				Offset:     0,
				Limit:      nChannels,
				AccessType: domainAccess,
				Status:     channels.AllStatus,
				Order:      defOrder,
				Dir:        ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  0,
					Offset: 0,
					Limit:  nChannels,
				},
				Channels: []channels.Channel{},
			},
		},
		{
			desc:     "retrieve channels with role ID",
			domainID: domain.ID,
			userID:   userID,
			pm: channels.Page{
				Offset: 0,
				Limit:  nChannels,
				RoleID: directChannels[0].RoleID,
				Status: channels.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  1,
					Offset: 0,
					Limit:  nChannels,
				},
				Channels: []channels.Channel{directChannels[0]},
			},
		},
		{
			desc:     "retrieve channels with wrong role ID",
			domainID: domain.ID,
			userID:   userID,
			pm: channels.Page{
				Offset: 0,
				Limit:  nChannels,
				RoleID: testsutil.GenerateUUID(t),
				Status: channels.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  0,
					Offset: 0,
					Limit:  nChannels,
				},
				Channels: []channels.Channel(nil),
			},
		},
		{
			desc:     "retrieve channels with role name",
			domainID: domain.ID,
			userID:   userID,
			pm: channels.Page{
				Offset:   0,
				Limit:    1,
				RoleName: directChannels[0].RoleName,
				Status:   channels.AllStatus,
				Order:    defOrder,
				Dir:      ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  10,
					Offset: 0,
					Limit:  1,
				},
				Channels: directChannels[0:1],
			},
		},
		{
			desc:     "retrieve channels with wrong role name",
			domainID: domain.ID,
			userID:   userID,
			pm: channels.Page{
				Offset:   0,
				Limit:    nChannels,
				RoleName: namegen.Generate(),
				Status:   channels.AllStatus,
				Order:    defOrder,
				Dir:      ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  0,
					Offset: 0,
					Limit:  nChannels,
				},
				Channels: []channels.Channel(nil),
			},
		},
		{
			desc:     "retrieve channels with actions",
			domainID: domain.ID,
			userID:   userID,
			pm: channels.Page{
				Offset:  0,
				Limit:   nChannels,
				Actions: availableActions,
				Status:  channels.AllStatus,
				Order:   defOrder,
				Dir:     ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  10,
					Offset: 0,
					Limit:  nChannels,
				},
				Channels: directChannels,
			},
		},
		{
			desc:     "retrieve channels with non-matching actions",
			domainID: domain.ID,
			userID:   userID,
			pm: channels.Page{
				Offset:  0,
				Limit:   nChannels,
				Actions: []string{"non_existent_action"},
				Status:  channels.AllStatus,
				Order:   defOrder,
				Dir:     ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  0,
					Offset: 0,
					Limit:  nChannels,
				},
				Channels: []channels.Channel(nil),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			page, err := repo.RetrieveUserChannels(context.Background(), tc.domainID, tc.userID, tc.pm)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected %s to contain %s\n", err, tc.err))
			if err == nil {
				assert.Equal(t, tc.response.Total, page.Total)
				assert.Equal(t, tc.response.Offset, page.Offset)
				assert.Equal(t, tc.response.Limit, page.Limit)
				expected := stripChannelDetails(tc.response.Channels)
				got := stripChannelDetails(page.Channels)
				assert.ElementsMatch(t, expected, got, fmt.Sprintf("expected %+v got %+v\n", expected, got))
			}
		})
	}
}

func TestSearchChannels(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM channels")
		require.Nil(t, err, fmt.Sprintf("clean channels unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	name := namegen.Generate()

	nChannels := uint64(200)
	expectedChannels := []channels.Channel{}
	baseTime := time.Now().UTC().Truncate(time.Microsecond)
	for i := 0; i < int(nChannels); i++ {
		channelName := name + strconv.Itoa(i)
		channel := channels.Channel{
			ID:        testsutil.GenerateUUID(t),
			Name:      channelName,
			Route:     testsutil.GenerateUUID(t),
			Metadata:  map[string]any{},
			Status:    channels.EnabledStatus,
			CreatedAt: baseTime.Add(time.Duration(i) * time.Microsecond),
		}
		_, err := repo.Save(context.Background(), channel)
		require.Nil(t, err, fmt.Sprintf("save channel unexpected error: %s", err))

		expectedChannels = append(expectedChannels, channels.Channel{
			ID:        channel.ID,
			Name:      channel.Name,
			CreatedAt: channel.CreatedAt,
		})
	}

	page, err := repo.RetrieveAll(context.Background(), channels.Page{Offset: 0, Limit: nChannels})
	require.Nil(t, err, fmt.Sprintf("retrieve all channels unexpected error: %s", err))
	assert.Equal(t, nChannels, page.Total)

	cases := []struct {
		desc     string
		page     channels.Page
		response channels.ChannelsPage
		err      error
	}{
		{
			desc: "with empty page",
			page: channels.Page{},
			response: channels.ChannelsPage{
				Channels: []channels.Channel(nil),
				Page: channels.Page{
					Total:  nChannels,
					Offset: 0,
					Limit:  0,
				},
			},
			err: nil,
		},
		{
			desc: "with offset only",
			page: channels.Page{
				Offset: 50,
			},
			response: channels.ChannelsPage{
				Channels: []channels.Channel(nil),
				Page: channels.Page{
					Total:  nChannels,
					Offset: 50,
					Limit:  0,
				},
			},
			err: nil,
		},
		{
			desc: "with limit only",
			page: channels.Page{
				Limit: 10,
				Order: "name",
				Dir:   ascDir,
			},
			response: channels.ChannelsPage{
				Channels: expectedChannels[0:10],
				Page: channels.Page{
					Total:  nChannels,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "retrieve all channels",
			page: channels.Page{
				Offset: 0,
				Limit:  nChannels,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  nChannels,
					Offset: 0,
					Limit:  nChannels,
				},
				Channels: expectedChannels,
			},
		},
		{
			desc: "with offset and limit",
			page: channels.Page{
				Offset: 10,
				Limit:  10,
				Order:  "name",
				Dir:    ascDir,
			},
			response: channels.ChannelsPage{
				Channels: expectedChannels[10:20],
				Page: channels.Page{
					Total:  nChannels,
					Offset: 10,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with offset out of range and limit",
			page: channels.Page{
				Offset: 1000,
				Limit:  50,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  nChannels,
					Offset: 1000,
					Limit:  50,
				},
				Channels: []channels.Channel(nil),
			},
		},
		{
			desc: "with offset and limit out of range",
			page: channels.Page{
				Offset: 190,
				Limit:  50,
				Order:  "name",
				Dir:    ascDir,
			},
			response: channels.ChannelsPage{
				Page: channels.Page{
					Total:  nChannels,
					Offset: 190,
					Limit:  50,
				},
				Channels: expectedChannels[190:200],
			},
		},
		{
			desc: "with shorter name",
			page: channels.Page{
				Name:   expectedChannels[0].Name[:4],
				Offset: 0,
				Limit:  10,
				Order:  "name",
				Dir:    ascDir,
			},
			response: channels.ChannelsPage{
				Channels: findChannels(expectedChannels, expectedChannels[0].Name[:4], 0, 10),
				Page: channels.Page{
					Total:  nChannels,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with longer name",
			page: channels.Page{
				Name:   expectedChannels[0].Name,
				Offset: 0,
				Limit:  10,
			},
			response: channels.ChannelsPage{
				Channels: []channels.Channel{expectedChannels[0]},
				Page: channels.Page{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with name SQL injected",
			page: channels.Page{
				Name:   fmt.Sprintf("%s' OR '1'='1", expectedChannels[0].Name[:1]),
				Offset: 0,
				Limit:  10,
			},
			response: channels.ChannelsPage{
				Channels: []channels.Channel(nil),
				Page: channels.Page{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with unknown name",
			page: channels.Page{
				Name:   namegen.Generate(),
				Offset: 0,
				Limit:  10,
			},
			response: channels.ChannelsPage{
				Channels: []channels.Channel(nil),
				Page: channels.Page{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with unknown name SQL injected",
			page: channels.Page{
				Name:   fmt.Sprintf("%s' OR '1'='1", namegen.Generate()),
				Offset: 0,
				Limit:  10,
			},
			response: channels.ChannelsPage{
				Channels: []channels.Channel(nil),
				Page: channels.Page{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with name in asc order",
			page: channels.Page{
				Order:  "name",
				Dir:    ascDir,
				Name:   expectedChannels[0].Name[:1],
				Offset: 0,
				Limit:  10,
			},
			response: channels.ChannelsPage{},
			err:      nil,
		},
		{
			desc: "with name in desc order",
			page: channels.Page{
				Order:  "name",
				Dir:    descDir,
				Name:   expectedChannels[0].Name[:1],
				Offset: 0,
				Limit:  10,
			},
			response: channels.ChannelsPage{},
			err:      nil,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			switch response, err := repo.RetrieveAll(context.Background(), c.page); {
			case err == nil:
				if c.page.Order != "" && c.page.Dir != "" {
					c.response = response
				}
				assert.Nil(t, err)
				assert.Equal(t, c.response.Total, response.Total)
				assert.Equal(t, c.response.Limit, response.Limit)
				assert.Equal(t, c.response.Offset, response.Offset)
				expected := stripChannelDetails(c.response.Channels)
				got := stripChannelDetails(response.Channels)
				assert.ElementsMatch(t, expected, got)
			default:
				assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s\n", err, c.err))
			}
		})
	}
}

func updateTimestamp(channels []channels.Channel) []channels.Channel {
	for i := range channels {
		channels[i].CreatedAt = validTimestamp
	}

	return channels
}

func generateDomain(t *testing.T, userID, memberID string) domains.Domain {
	domain := domains.Domain{
		ID:        testsutil.GenerateUUID(t),
		Route:     namegen.Generate(),
		Status:    domains.EnabledStatus,
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		CreatedBy: userID,
	}

	drepo := dpostgres.NewRepository(database)
	_, err := drepo.SaveDomain(context.Background(), domain)
	require.Nil(t, err, fmt.Sprintf("add new domain: expected nil got %s\n", err))
	newRolesProvision := []roles.RoleProvision{
		{
			Role: roles.Role{
				ID:        testsutil.GenerateUUID(t) + "_" + domain.ID,
				Name:      "admin",
				EntityID:  domain.ID,
				CreatedAt: validTimestamp,
				CreatedBy: userID,
			},
			OptionalActions: domainAvailableActions,
			OptionalMembers: []string{userID, memberID},
		},
	}
	_, err = drepo.AddRoles(context.Background(), newRolesProvision)
	require.Nil(t, err, fmt.Sprintf("add new role: expected nil got %s\n", err))
	domain.Roles = []roles.MemberRoleActions{
		{
			RoleID:   newRolesProvision[0].Role.ID,
			RoleName: newRolesProvision[0].Role.Name,
			Actions:  newRolesProvision[0].OptionalActions,
		},
	}

	return domain
}

func generateGroup(t *testing.T, userID, memberID, domainID string) groups.Group {
	group := groups.Group{
		ID:        testsutil.GenerateUUID(t),
		Name:      namegen.Generate(),
		Domain:    domainID,
		Status:    groups.EnabledStatus,
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}

	grepo := gpostgres.New(database)
	_, err := grepo.Save(context.Background(), group)
	require.Nil(t, err, fmt.Sprintf("add new group: expected nil got %s\n", err))
	newRolesProvision := []roles.RoleProvision{
		{
			Role: roles.Role{
				ID:        testsutil.GenerateUUID(t) + "_" + group.ID,
				Name:      "admin",
				EntityID:  group.ID,
				CreatedAt: validTimestamp,
				CreatedBy: userID,
			},
			OptionalActions: groupAvailableActions,
			OptionalMembers: []string{userID, memberID},
		},
	}
	_, err = grepo.AddRoles(context.Background(), newRolesProvision)
	require.Nil(t, err, fmt.Sprintf("add new role: expected nil got %s\n", err))
	group.Roles = []roles.MemberRoleActions{
		{
			RoleID:   newRolesProvision[0].Role.ID,
			RoleName: newRolesProvision[0].Role.Name,
			Actions:  newRolesProvision[0].OptionalActions,
		},
	}

	return group
}

func stripChannelDetails(channels []channels.Channel) []channels.Channel {
	for i := range channels {
		channels[i].CreatedAt = validTimestamp
		channels[i].Actions = []string{}
		channels[i].Route = ""
		if channels[i].Metadata != nil && len(channels[i].Metadata) == 0 {
			channels[i].Metadata = nil
		}
		if channels[i].ConnectionTypes != nil && len(channels[i].ConnectionTypes) == 0 {
			channels[i].ConnectionTypes = nil
		}
		channels[i].AccessProviderRoleActions = []string{}
	}

	return channels
}

func findChannels(chs []channels.Channel, query string, offset, limit uint64) []channels.Channel {
	rchannels := []channels.Channel{}
	for _, channel := range chs {
		if strings.Contains(channel.Name, query) {
			rchannels = append(rchannels, channel)
		}
	}

	if offset > uint64(len(rchannels)) {
		return []channels.Channel{}
	}

	if limit > uint64(len(rchannels)) {
		return rchannels[offset:]
	}

	return rchannels[offset:limit]
}

func verifyChannelsOrdering(t *testing.T, chs []channels.Channel, order, dir string) {
	if order == "" || len(chs) <= 1 {
		return
	}

	switch order {
	case "name":
		for i := 1; i < len(chs); i++ {
			if dir == ascDir {
				assert.LessOrEqual(t, chs[i-1].Name, chs[i].Name)
				continue
			}
			assert.GreaterOrEqual(t, chs[i-1].Name, chs[i].Name)
		}
	case "created_at":
		for i := 1; i < len(chs); i++ {
			if dir == ascDir {
				assert.True(t, !chs[i-1].CreatedAt.After(chs[i].CreatedAt))
				continue
			}
			assert.True(t, !chs[i-1].CreatedAt.Before(chs[i].CreatedAt))
		}
	case "updated_at":
		for i := 1; i < len(chs); i++ {
			if dir == ascDir {
				assert.True(t, !chs[i-1].UpdatedAt.After(chs[i].UpdatedAt))
				continue
			}
			assert.True(t, !chs[i-1].UpdatedAt.Before(chs[i].UpdatedAt))
		}
	}
}
