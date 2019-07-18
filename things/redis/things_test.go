//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package redis_test

import (
	"context"
	"fmt"
	"testing"

	r "github.com/go-redis/redis"
	"github.com/mainflux/mainflux/things/redis"
	"github.com/mainflux/mainflux/things/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThingSave(t *testing.T) {
	thingCache := redis.NewThingCache(redisClient)
	key, err := uuid.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	id := "123"
	id2 := "124"

	err = thingCache.Save(context.Background(), key, id2)
	require.Nil(t, err, fmt.Sprintf("Save thing to cache: expected nil got %s", err))

	cases := []struct {
		desc string
		ID   string
		key  string
		err  error
	}{
		{
			desc: "Save thing to cache",
			ID:   id,
			key:  key,
			err:  nil,
		},
		{
			desc: "Save already cached thing to cache",
			ID:   id2,
			key:  key,
			err:  nil,
		},
	}

	for _, tc := range cases {
		err := thingCache.Save(context.Background(), tc.key, tc.ID)
		assert.Nil(t, err, fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))

	}
}

func TestThingID(t *testing.T) {
	thingCache := redis.NewThingCache(redisClient)

	key, err := uuid.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	id := "123"
	err = thingCache.Save(context.Background(), key, id)
	require.Nil(t, err, fmt.Sprintf("Save thing to cache: expected nil got %s", err))

	cases := map[string]struct {
		ID  string
		key string
		err error
	}{
		"Get ID by existing thing-key": {
			ID:  id,
			key: key,
			err: nil,
		},
		"Get ID by non-existing thing-key": {
			ID:  "",
			key: wrongValue,
			err: r.Nil,
		},
	}

	for desc, tc := range cases {
		cacheID, err := thingCache.ID(context.Background(), tc.key)
		assert.Equal(t, tc.ID, cacheID, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.ID, cacheID))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestThingRemove(t *testing.T) {
	thingCache := redis.NewThingCache(redisClient)

	key, err := uuid.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	id := "123"
	id2 := "321"
	thingCache.Save(context.Background(), key, id)

	cases := []struct {
		desc string
		ID   string
		err  error
	}{
		{
			desc: "Remove existing thing from cache",
			ID:   id,
			err:  nil,
		},
		{
			desc: "Remove non-existing thing from cache",
			ID:   id2,
			err:  r.Nil,
		},
	}

	for _, tc := range cases {
		err := thingCache.Remove(context.Background(), tc.ID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}

}
