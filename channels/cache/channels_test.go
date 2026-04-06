// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cache_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/absmach/magistrala/channels"
	"github.com/absmach/magistrala/channels/cache"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

var (
	testRoute   = "test-route"
	nonExistent = "non-existing"
)

func setupChannelsClient(t *testing.T) channels.Cache {
	opts, err := redis.ParseURL(redisURL)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on parsing redis URL: %s", err))
	redisClient := redis.NewClient(opts)

	return cache.NewChannelsCache(redisClient, 10*time.Minute)
}

func TestSave(t *testing.T) {
	cc := setupChannelsClient(t)

	route := testRoute
	domainID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc         string
		domainID     string
		channelID    string
		channelRoute string
		err          error
	}{
		{
			desc:         "Save successfully",
			domainID:     domainID,
			channelID:    testsutil.GenerateUUID(t),
			channelRoute: route,
			err:          nil,
		},
		{
			desc:         "Save with empty domain ID",
			domainID:     "",
			channelID:    testsutil.GenerateUUID(t),
			channelRoute: route,
			err:          cache.ErrEmptyDomainID,
		},
		{
			desc:         "Save with empty channel ID",
			domainID:     domainID,
			channelID:    "",
			channelRoute: route,
			err:          cache.ErrEmptyChannelID,
		},
		{
			desc:         "Save with empty channel route",
			domainID:     domainID,
			channelID:    testsutil.GenerateUUID(t),
			channelRoute: "",
			err:          cache.ErrEmptyChannelRoute,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := cc.Save(context.Background(), tc.channelRoute, tc.domainID, tc.channelID)
			assert.True(t, errors.Contains(err, tc.err))
		})
	}
}

func TestID(t *testing.T) {
	cc := setupChannelsClient(t)

	domainID := testsutil.GenerateUUID(t)
	route := testRoute
	id := testsutil.GenerateUUID(t)

	err := cc.Save(context.Background(), route, domainID, id)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on saving channel ID: %s", err))

	cases := []struct {
		desc         string
		domainID     string
		channelRoute string
		channelID    string
		err          error
	}{
		{
			desc:         "Retrieve existing channel",
			domainID:     domainID,
			channelRoute: route,
			channelID:    id,
			err:          nil,
		},
		{
			desc:         "Retrieve non-existing channel",
			domainID:     domainID,
			channelRoute: nonExistent,
			channelID:    "",
			err:          repoerr.ErrNotFound,
		},
		{
			desc:         "Retrieve with empty domain ID",
			domainID:     "",
			channelRoute: route,
			channelID:    "",
			err:          cache.ErrEmptyDomainID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			id, err := cc.ID(context.Background(), tc.channelRoute, tc.domainID)
			assert.Equal(t, tc.channelID, id, fmt.Sprintf("expected channel ID '%s' got '%s'", tc.channelID, id))
			assert.True(t, errors.Contains(err, tc.err))
		})
	}
}

func TestRemove(t *testing.T) {
	cc := setupChannelsClient(t)

	domainID := testsutil.GenerateUUID(t)
	route := testRoute
	id := testsutil.GenerateUUID(t)

	err := cc.Save(context.Background(), domainID, route, id)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on saving channel ID: %s", err))

	cases := []struct {
		desc         string
		domainID     string
		channelRoute string
		err          error
	}{
		{
			desc:         "Remove existing channel",
			domainID:     domainID,
			channelRoute: route,
			err:          nil,
		},
		{
			desc:         "Remove non-existing channel",
			domainID:     domainID,
			channelRoute: nonExistent,
			err:          nil,
		},
		{
			desc:         "Remove with empty domain ID",
			domainID:     "",
			channelRoute: route,
			err:          cache.ErrEmptyDomainID,
		},
		{
			desc:         "Remove with empty channel route",
			domainID:     domainID,
			channelRoute: "",
			err:          cache.ErrEmptyChannelRoute,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := cc.Remove(context.Background(), tc.channelRoute, tc.domainID)
			assert.True(t, errors.Contains(err, tc.err))

			if tc.err == nil {
				id, err := cc.ID(context.Background(), tc.channelRoute, tc.domainID)
				assert.Equal(t, "", id, fmt.Sprintf("expected channel ID to be empty after removal, got '%s'", id))
				assert.True(t, errors.Contains(err, repoerr.ErrNotFound))
			}
		})
	}
}
