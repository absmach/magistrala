// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cache_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/things/cache"
	"github.com/stretchr/testify/assert"
)

const (
	testKey  = "testKey"
	testID   = "testID"
	testKey2 = "testKey2"
	testID2  = "testID2"
)

func TestSave(t *testing.T) {
	redisClient.FlushAll(context.Background())
	tscache := cache.NewCache(redisClient, 1*time.Minute)
	ctx := context.Background()

	cases := []struct {
		desc string
		key  string
		id   string
		err  error
	}{
		{
			desc: "Save thing to cache",
			key:  testKey,
			id:   testID,
			err:  nil,
		},
		{
			desc: "Save already cached thing to cache",
			key:  testKey,
			id:   testID,
			err:  nil,
		},
		{
			desc: "Save another thing to cache",
			key:  testKey2,
			id:   testID2,
			err:  nil,
		},
		{
			desc: "Save thing with long key ",
			key:  strings.Repeat("a", 513*1024*1024),
			id:   testID,
			err:  errors.ErrCreateEntity,
		},
		{
			desc: "Save thing with long id ",
			key:  testKey,
			id:   strings.Repeat("a", 513*1024*1024),
			err:  errors.ErrCreateEntity,
		},
		{
			desc: "Save thing with empty key",
			key:  "",
			id:   testID,
			err:  errors.ErrCreateEntity,
		},
		{
			desc: "Save thing with empty id",
			key:  testKey,
			id:   "",
			err:  errors.ErrCreateEntity,
		},
		{
			desc: "Save thing with empty key and id",
			key:  "",
			id:   "",
			err:  errors.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		err := tscache.Save(ctx, tc.key, tc.id)
		if err == nil {
			id, _ := tscache.ID(ctx, tc.key)
			assert.Equal(t, tc.id, id, fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.id, id))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestID(t *testing.T) {
	redisClient.FlushAll(context.Background())
	tscache := cache.NewCache(redisClient, 1*time.Minute)
	ctx := context.Background()

	err := tscache.Save(ctx, testKey, testID)
	assert.Nil(t, err, fmt.Sprintf("Unexpected error while trying to save: %s", err))

	cases := []struct {
		desc string
		key  string
		id   string
		err  error
	}{
		{
			desc: "Get thing ID from cache",
			key:  testKey,
			id:   testID,
			err:  nil,
		},
		{
			desc: "Get thing ID from cache for non existing thing",
			key:  "nonExistingKey",
			id:   "",
			err:  errors.ErrNotFound,
		},
		{
			desc: "Get thing ID from cache for empty key",
			key:  "",
			id:   "",
			err:  errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		id, err := tscache.ID(ctx, tc.key)
		if err == nil {
			assert.Equal(t, tc.id, id, fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.id, id))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemove(t *testing.T) {
	redisClient.FlushAll(context.Background())
	tscache := cache.NewCache(redisClient, 1*time.Minute)
	ctx := context.Background()

	err := tscache.Save(ctx, testKey, testID)
	assert.Nil(t, err, fmt.Sprintf("Unexpected error while trying to save: %s", err))

	cases := []struct {
		desc string
		key  string
		err  error
	}{
		{
			desc: "Remove existing thing from cache",
			key:  testID,
			err:  nil,
		},
		{
			desc: "Remove non existing thing from cache",
			key:  testID2,
			err:  nil,
		},
		{
			desc: "Remove thing with empty ID from cache",
			key:  "",
			err:  nil,
		},
		{
			desc: "Remove thing with long id from cache",
			key:  strings.Repeat("a", 513*1024*1024),
			err:  errors.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		err := tscache.Remove(ctx, tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
