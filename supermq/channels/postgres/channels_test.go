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
	"github.com/absmach/supermq/channels"
	"github.com/absmach/supermq/channels/postgres"
	"github.com/absmach/supermq/clients"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/connections"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
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
		Tags:            []string{"tag1", "tag2"},
		Metadata:        map[string]interface{}{"key": "value"},
		CreatedAt:       time.Now().UTC().Truncate(time.Microsecond),
		Status:          clients.EnabledStatus,
		ConnectionTypes: []connections.ConnType{},
	}
	validConnection = channels.Connection{
		ClientID:  testsutil.GenerateUUID(&testing.T{}),
		ChannelID: validChannel.ID,
		DomainID:  validChannel.Domain,
		Type:      connections.Publish,
	}
	validTimestamp = time.Now().UTC().Truncate(time.Millisecond)
)

func TestSave(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM channels")
		require.Nil(t, err, fmt.Sprintf("clean channels unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

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
			err:     repoerr.ErrConflict,
		},
		{
			desc: "add channel with invalid ID",
			channel: channels.Channel{
				ID:        invalidID,
				Domain:    testsutil.GenerateUUID(t),
				Name:      namegen.Generate(),
				Metadata:  map[string]interface{}{"key": "value"},
				CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
				Status:    clients.EnabledStatus,
			},
			resp: []channels.Channel{},
			err:  repoerr.ErrMalformedEntity,
		},
		{
			desc: "add channel with invalid domain",
			channel: channels.Channel{
				ID:        testsutil.GenerateUUID(t),
				Domain:    invalidID,
				Name:      namegen.Generate(),
				Metadata:  map[string]interface{}{"key": "value"},
				CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
				Status:    clients.EnabledStatus,
			},
			resp: []channels.Channel{},
			err:  repoerr.ErrMalformedEntity,
		},
		{
			desc: "add channel with invalid name",
			channel: channels.Channel{
				ID:        testsutil.GenerateUUID(t),
				Domain:    testsutil.GenerateUUID(t),
				Name:      strings.Repeat("a", 1025),
				Metadata:  map[string]interface{}{"key": "value"},
				CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
				Status:    clients.EnabledStatus,
			},
			resp: []channels.Channel{},
			err:  repoerr.ErrMalformedEntity,
		},
		{
			desc: "add channel with invalid metadata",
			channel: channels.Channel{
				ID:     testsutil.GenerateUUID(t),
				Domain: testsutil.GenerateUUID(t),
				Name:   namegen.Generate(),
				Metadata: map[string]interface{}{
					"key": make(chan int),
				},
				CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
				Status:    clients.EnabledStatus,
			},
			resp: []channels.Channel{},
			err:  repoerr.ErrMalformedEntity,
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
				Metadata:  map[string]interface{}{"key": "value"},
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
				Metadata:  map[string]interface{}{"key1": "value1"},
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
				Metadata:  map[string]interface{}{"key": "value"},
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
				Metadata:  map[string]interface{}{"key": "value"},
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
	disabledChannel.Status = clients.DisabledStatus

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
				Status:    clients.DisabledStatus,
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc: "enable channel successfully",
			channel: channels.Channel{
				ID:        disabledChannel.ID,
				Status:    clients.EnabledStatus,
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc: "change status channel with invalid ID",
			channel: channels.Channel{
				ID:        testsutil.GenerateUUID(t),
				Status:    clients.DisabledStatus,
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "change status channel with empty ID",
			channel: channels.Channel{
				Status:    clients.DisabledStatus,
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

func TestRetrieveAll(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM channels")
		require.Nil(t, err, fmt.Sprintf("clean channels unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)
	num := 200

	var items []channels.Channel
	parentID := ""
	for i := 0; i < num; i++ {
		name := namegen.Generate()
		channel := channels.Channel{
			ID:              testsutil.GenerateUUID(t),
			Domain:          testsutil.GenerateUUID(t),
			ParentGroup:     parentID,
			Name:            name,
			Metadata:        map[string]interface{}{"name": name},
			CreatedAt:       time.Now().UTC().Truncate(time.Microsecond),
			Status:          clients.EnabledStatus,
			ConnectionTypes: []connections.ConnType{},
		}
		_, err := repo.Save(context.Background(), channel)
		require.Nil(t, err, fmt.Sprintf("create channel unexpected error: %s", err))
		items = append(items, channel)
		if i%20 == 0 {
			parentID = channel.ID
		}
	}

	cases := []struct {
		desc     string
		page     channels.Page
		response channels.Page
		err      error
	}{
		{
			desc: "retrieve channels successfully",
			page: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: 0,
					Limit:  10,
				},
			},
			response: channels.Page{
				PageMetadata: channels.PageMetadata{
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
			page: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: 10,
					Limit:  10,
				},
			},
			response: channels.Page{
				PageMetadata: channels.PageMetadata{
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
			page: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: 0,
					Limit:  50,
				},
			},
			response: channels.Page{
				PageMetadata: channels.PageMetadata{
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
			page: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: 50,
					Limit:  50,
				},
			},
			response: channels.Page{
				PageMetadata: channels.PageMetadata{
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
			page: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: 1000,
					Limit:  50,
				},
			},
			response: channels.Page{
				PageMetadata: channels.PageMetadata{
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
			page: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: 170,
					Limit:  50,
				},
			},
			response: channels.Page{
				PageMetadata: channels.PageMetadata{
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
			page: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: 0,
					Limit:  1000,
				},
			},
			response: channels.Page{
				PageMetadata: channels.PageMetadata{
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
			page: channels.Page{},
			response: channels.Page{
				PageMetadata: channels.PageMetadata{
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
			page: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: 0,
					Limit:  10,
					Name:   items[0].Name,
				},
			},
			response: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Channels: []channels.Channel{items[0]},
			},
			err: nil,
		},
		{
			desc: "retrieve channels with domain",
			page: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: 0,
					Limit:  10,
					Domain: items[0].Domain,
				},
			},
			response: channels.Page{
				PageMetadata: channels.PageMetadata{
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
			page: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset:   0,
					Limit:    10,
					Metadata: items[0].Metadata,
				},
			},
			response: channels.Page{
				PageMetadata: channels.PageMetadata{
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
			page: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: 0,
					Limit:  10,
					Metadata: map[string]interface{}{
						"key": make(chan int),
					},
				},
			},
			response: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
				Channels: []channels.Channel(nil),
			},
			err: errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			switch channels, err := repo.RetrieveAll(context.Background(), tc.page.PageMetadata); {
			case err == nil:
				assert.Nil(t, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
				assert.Equal(t, tc.response.Total, channels.Total, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Total, channels.Total))
				assert.Equal(t, tc.response.Limit, channels.Limit, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Limit, channels.Limit))
				assert.Equal(t, tc.response.Offset, channels.Offset, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Offset, channels.Offset))
				got := updateTimestamp(channels.Channels)
				resp := updateTimestamp(tc.response.Channels)
				assert.ElementsMatch(t, resp, got, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, resp, got))
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
			err:           repoerr.ErrMalformedEntity,
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
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "add connection with invalid channel ID",
			connection: channels.Connection{
				ClientID:  testsutil.GenerateUUID(t),
				ChannelID: invalidID,
				DomainID:  testsutil.GenerateUUID(t),
				Type:      connections.Publish,
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "add connection with invalid domain ID",
			connection: channels.Connection{
				ClientID:  testsutil.GenerateUUID(t),
				ChannelID: testsutil.GenerateUUID(t),
				DomainID:  invalidID,
				Type:      connections.Publish,
			},
			err: repoerr.ErrMalformedEntity,
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

	cases := []struct {
		desc       string
		connection channels.Connection
		err        error
	}{
		{
			desc:       "remove connection successfully",
			connection: validConnection,
			err:        nil,
		},
		{
			desc: "remove connection with non-existent channel",
			connection: channels.Connection{
				ClientID:  testsutil.GenerateUUID(t),
				ChannelID: testsutil.GenerateUUID(t),
				DomainID:  validChannel.Domain,
				Type:      connections.Publish,
			},
			err: nil,
		},
		{
			desc: "remove connection with non-existent domain",
			connection: channels.Connection{
				ClientID:  testsutil.GenerateUUID(t),
				ChannelID: validChannel.ID,
				DomainID:  testsutil.GenerateUUID(t),
				Type:      connections.Publish,
			},
			err: nil,
		},
		{
			desc: "remove connection with non-existent client",
			connection: channels.Connection{
				ClientID:  testsutil.GenerateUUID(t),
				ChannelID: validChannel.ID,
				DomainID:  validChannel.Domain,
				Type:      connections.Publish,
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.RemoveConnections(context.Background(), []channels.Connection{tc.connection})
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
			Metadata:        map[string]interface{}{"name": name},
			CreatedAt:       time.Now().UTC().Truncate(time.Microsecond),
			Status:          clients.EnabledStatus,
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
			Metadata:    map[string]interface{}{"name": name},
			CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
			Status:      clients.EnabledStatus,
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

func updateTimestamp(channels []channels.Channel) []channels.Channel {
	for i := range channels {
		channels[i].CreatedAt = validTimestamp
	}

	return channels
}
