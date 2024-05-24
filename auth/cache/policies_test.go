// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cache_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/auth/cache"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

var policy = auth.PolicyReq{
	SubjectType: auth.UserType,
	Subject:     testsutil.GenerateUUID(&testing.T{}),
	ObjectType:  auth.ThingType,
	Object:      testsutil.GenerateUUID(&testing.T{}),
	Permission:  auth.ViewPermission,
}

func setupRedisClient(t *testing.T) auth.Cache {
	opts, err := redis.ParseURL(redisURL)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on parsing redis URL: %s", err))
	redisClient := redis.NewClient(opts)
	return cache.NewPoliciesCache(redisClient, 10*time.Minute)
}

func TestSave(t *testing.T) {
	authCache := setupRedisClient(t)

	cases := []struct {
		desc   string
		policy auth.PolicyReq
		err    error
	}{
		{
			desc:   "Save policy",
			policy: policy,
			err:    nil,
		},
		{
			desc:   "Save already cached policy",
			policy: policy,
			err:    nil,
		},
		{
			desc: "Save another policy",
			policy: auth.PolicyReq{
				SubjectType: auth.UserType,
				Subject:     testsutil.GenerateUUID(&testing.T{}),
				ObjectType:  auth.ThingType,
				Object:      testsutil.GenerateUUID(&testing.T{}),
				Permission:  auth.ViewPermission,
			},
			err: nil,
		},
		{
			desc: "Save another policy with domain",
			policy: auth.PolicyReq{
				Domain:      testsutil.GenerateUUID(&testing.T{}),
				SubjectType: auth.UserType,
				Subject:     testsutil.GenerateUUID(&testing.T{}),
				ObjectType:  auth.ThingType,
				Object:      testsutil.GenerateUUID(&testing.T{}),
				Permission:  auth.ViewPermission,
			},
			err: nil,
		},
		{
			desc: "Save policy with long key",
			policy: auth.PolicyReq{
				SubjectType: strings.Repeat("a", 513*1024*1024),
				Subject:     strings.Repeat("a", 513*1024*1024),
				ObjectType:  strings.Repeat("a", 513*1024*1024),
				Object:      strings.Repeat("a", 513*1024*1024),
				Permission:  auth.ViewPermission,
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "Save policy with long value",
			policy: auth.PolicyReq{
				SubjectType: auth.UserType,
				Subject:     testsutil.GenerateUUID(&testing.T{}),
				ObjectType:  auth.ThingType,
				Object:      testsutil.GenerateUUID(&testing.T{}),
				Permission:  strings.Repeat("a", 513*1024*1024),
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "Save policy with empty key",
			policy: auth.PolicyReq{
				Permission: auth.ViewPermission,
			},
			err: nil,
		},
		{
			desc: "Save policy with empty subject",
			policy: auth.PolicyReq{
				ObjectType: auth.ThingType,
				Object:     testsutil.GenerateUUID(&testing.T{}),
				Permission: auth.ViewPermission,
			},
			err: nil,
		},
		{
			desc: "Save policy with empty object",
			policy: auth.PolicyReq{
				SubjectType: auth.UserType,
				Subject:     testsutil.GenerateUUID(&testing.T{}),
				Permission:  auth.ViewPermission,
			},
			err: nil,
		},
		{
			desc: "Save policy with empty value",
			policy: auth.PolicyReq{
				SubjectType: auth.UserType,
				Subject:     testsutil.GenerateUUID(&testing.T{}),
				ObjectType:  auth.ThingType,
				Object:      testsutil.GenerateUUID(&testing.T{}),
			},
			err: nil,
		},
		{
			desc:   "Save policy with empty key and id",
			policy: auth.PolicyReq{},
			err:    nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			key, val := tc.policy.KV()
			err := authCache.Save(context.Background(), key, val)
			if err == nil {
				ok := authCache.Contains(context.Background(), key, val)
				assert.True(t, ok)
			}
			assert.True(t, errors.Contains(err, tc.err))
		})
	}
}

func TestContains(t *testing.T) {
	authCache := setupRedisClient(t)

	key, val := policy.KV()
	err := authCache.Save(context.Background(), key, val)
	assert.Nil(t, err, fmt.Sprintf("Unexpected error while trying to save: %s", err))

	cases := []struct {
		desc   string
		policy auth.PolicyReq
		ok     bool
	}{
		{
			desc:   "Contains existing policy",
			policy: policy,
			ok:     true,
		},
		{
			desc: "Contains invalid policy",
			policy: auth.PolicyReq{
				SubjectType: policy.SubjectType,
				Subject:     policy.Subject,
				ObjectType:  policy.ObjectType,
				Object:      policy.Object,
				Permission:  auth.EditPermission,
			},
		},
		{
			desc: "Contains non existing policy",
			policy: auth.PolicyReq{
				SubjectType: auth.UserType,
				Subject:     testsutil.GenerateUUID(&testing.T{}),
				ObjectType:  auth.ThingType,
				Object:      testsutil.GenerateUUID(&testing.T{}),
				Permission:  auth.ViewPermission,
			},
		},
		{
			desc: "Contains non existing policy with domain",
			policy: auth.PolicyReq{
				Domain:      testsutil.GenerateUUID(&testing.T{}),
				SubjectType: auth.UserType,
				Subject:     testsutil.GenerateUUID(&testing.T{}),
				ObjectType:  auth.ThingType,
				Object:      testsutil.GenerateUUID(&testing.T{}),
				Permission:  auth.ViewPermission,
			},
		},
		{
			desc: "Contains policy with empty key",
			policy: auth.PolicyReq{
				Permission: auth.ViewPermission,
			},
		},
		{
			desc: "Contains policy with long key",
			policy: auth.PolicyReq{
				SubjectType: strings.Repeat("a", 513*1024*1024),
				Subject:     strings.Repeat("a", 513*1024*1024),
				ObjectType:  strings.Repeat("a", 513*1024*1024),
				Object:      strings.Repeat("a", 513*1024*1024),
				Permission:  auth.ViewPermission,
			},
		},
		{
			desc: "Contains policy with empty value",
			policy: auth.PolicyReq{
				SubjectType: auth.UserType,
				Subject:     testsutil.GenerateUUID(&testing.T{}),
				ObjectType:  auth.ThingType,
				Object:      testsutil.GenerateUUID(&testing.T{}),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			key, val := tc.policy.KV()
			ok := authCache.Contains(context.Background(), key, val)
			assert.Equal(t, tc.ok, ok)
		})
	}
}

func TestRemove(t *testing.T) {
	authCache := setupRedisClient(t)

	subject := policy.Subject
	object := policy.Object

	num := 200
	var policies []auth.PolicyReq
	for i := 0; i < num; i++ {
		policy.Subject = fmt.Sprintf("%s-%d", policy.Subject, i)
		policy.Object = fmt.Sprintf("%s-%d", policy.Object, i)
		key, val := policy.KV()
		err := authCache.Save(context.Background(), key, val)
		assert.Nil(t, err, fmt.Sprintf("Unexpected error while trying to save: %s", err))
		policies = append(policies, policy)
	}

	cases := []struct {
		desc     string
		multiple bool
		policy   auth.PolicyReq
		err      error
	}{
		{
			desc:   "Remove an existing policy from cache",
			policy: policies[0],
			err:    nil,
		},
		{
			desc:     "Remove multiple existing policies from cache with subject",
			multiple: true,
			policy: auth.PolicyReq{
				Subject: subject,
			},
			err: nil,
		},
		{
			desc:     "Remove multiple existing policies from cache with object",
			multiple: true,
			policy: auth.PolicyReq{
				Object: object,
			},
			err: nil,
		},
		{
			desc: "Remove non existing policy from cache",
			policy: auth.PolicyReq{
				SubjectType: auth.UserType,
				Subject:     testsutil.GenerateUUID(&testing.T{}),
				ObjectType:  auth.ThingType,
				Object:      testsutil.GenerateUUID(&testing.T{}),
				Permission:  auth.ViewPermission,
			},
			err: nil,
		},
		{
			desc: "Remove policy with empty key from cache",
			policy: auth.PolicyReq{
				Permission: auth.ViewPermission,
			},
			err: nil,
		},
		{
			desc: "Remove policy with long key from cache",
			policy: auth.PolicyReq{
				SubjectType: strings.Repeat("a", 513*1024*1024),
				Subject:     strings.Repeat("a", 513*1024*1024),
				ObjectType:  strings.Repeat("a", 513*1024*1024),
				Object:      strings.Repeat("a", 513*1024*1024),
				Permission:  auth.ViewPermission,
			},
			err: repoerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := authCache.Remove(context.Background(), tc.policy.KeyForRemoval())
			assert.True(t, errors.Contains(err, tc.err))
			if err == nil {
				key, val := tc.policy.KV()
				ok := authCache.Contains(context.Background(), key, val)
				assert.False(t, ok)
				if tc.multiple {
					switch {
					case tc.policy.Subject != "":
						for _, p := range policies {
							if strings.HasPrefix(p.Subject, subject) {
								key, val := p.KV()
								ok := authCache.Contains(context.Background(), key, val)
								assert.False(t, ok)
							}
						}
					case tc.policy.Object != "":
						for _, p := range policies {
							if strings.HasPrefix(p.Object, object) {
								key, val := p.KV()
								ok := authCache.Contains(context.Background(), key, val)
								assert.False(t, ok)
							}
						}
					}
				}
			}
		})
	}
}
