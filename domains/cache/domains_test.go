// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cache_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/absmach/magistrala/domains"
	"github.com/absmach/magistrala/domains/cache"
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

func setupDomainsClient(t *testing.T) domains.Cache {
	opts, err := redis.ParseURL(redisURL)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on parsing redis URL: %s", err))
	redisClient := redis.NewClient(opts)

	return cache.NewDomainsCache(redisClient, 10*time.Minute)
}

func TestSaveStatus(t *testing.T) {
	dc := setupDomainsClient(t)

	domainID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc     string
		domainID string
		status   domains.Status
		err      error
	}{
		{
			desc:     "Save with enabled status",
			domainID: domainID,
			status:   domains.EnabledStatus,
			err:      nil,
		},
		{
			desc:     "Save with disabled status",
			domainID: testsutil.GenerateUUID(t),
			status:   domains.DisabledStatus,
			err:      nil,
		},
		{
			desc:     "Save with frozen status",
			domainID: testsutil.GenerateUUID(t),
			status:   domains.FreezeStatus,
			err:      nil,
		},
		{
			desc:     "Save with empty domain ID",
			domainID: "",
			status:   domains.EnabledStatus,
			err:      repoerr.ErrCreateEntity,
		},
		{
			desc:     "Save with all status",
			domainID: testsutil.GenerateUUID(t),
			status:   domains.AllStatus,
			err:      nil,
		},
		{
			desc:     "Save with invalid status",
			domainID: testsutil.GenerateUUID(t),
			status:   domains.Status(6),
			err:      repoerr.ErrCreateEntity,
		},
		{
			desc:     "Save the same record",
			domainID: domainID,
			status:   domains.EnabledStatus,
			err:      nil,
		},
		{
			desc:     "Save client with long id ",
			domainID: strings.Repeat("a", 513*1024*1024),
			status:   domains.EnabledStatus,
			err:      repoerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := dc.SaveStatus(context.Background(), tc.domainID, tc.status)
			assert.True(t, errors.Contains(err, tc.err))
		})
	}
}

func TestSaveID(t *testing.T) {
	dc := setupDomainsClient(t)

	route := testRoute
	domainID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc     string
		route    string
		domainID string
		err      error
	}{
		{
			desc:     "Save domain ID with valid route",
			route:    route,
			domainID: domainID,
			err:      nil,
		},
		{
			desc:     "Save domain ID with empty route",
			route:    "",
			domainID: domainID,
			err:      repoerr.ErrCreateEntity,
		},
		{
			desc:     "Save domain ID with empty domain ID",
			route:    route,
			domainID: "",
			err:      repoerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := dc.SaveID(context.Background(), tc.route, tc.domainID)
			assert.True(t, errors.Contains(err, tc.err))
		})
	}
}

func TestStatus(t *testing.T) {
	dc := setupDomainsClient(t)

	enabledDomainID := testsutil.GenerateUUID(t)
	err := dc.SaveStatus(context.Background(), enabledDomainID, domains.EnabledStatus)
	assert.Nil(t, err, fmt.Sprintf("Unexpected error while trying to save: %s", err))

	disabledDomainID := testsutil.GenerateUUID(t)
	err = dc.SaveStatus(context.Background(), disabledDomainID, domains.DisabledStatus)
	assert.Nil(t, err, fmt.Sprintf("Unexpected error while trying to save: %s", err))

	frozenDomainID := testsutil.GenerateUUID(t)
	err = dc.SaveStatus(context.Background(), frozenDomainID, domains.FreezeStatus)
	assert.Nil(t, err, fmt.Sprintf("Unexpected error while trying to save: %s", err))

	allDomainID := testsutil.GenerateUUID(t)
	err = dc.SaveStatus(context.Background(), allDomainID, domains.AllStatus)
	assert.Nil(t, err, fmt.Sprintf("Unexpected error while trying to save: %s", err))

	cases := []struct {
		desc     string
		domainID string
		status   domains.Status
		err      error
	}{
		{
			desc:     "Get domain status from cache for enabled domain",
			domainID: enabledDomainID,
			status:   domains.EnabledStatus,
			err:      nil,
		},
		{
			desc:     "Get domain status from cache for disabled domain",
			domainID: disabledDomainID,
			status:   domains.DisabledStatus,
			err:      nil,
		},
		{
			desc:     "Get domain status from cache for frozen domain",
			domainID: frozenDomainID,
			status:   domains.FreezeStatus,
			err:      nil,
		},
		{
			desc:     "Get domain status from cache for all domain",
			domainID: allDomainID,
			status:   domains.AllStatus,
			err:      nil,
		},
		{
			desc:     "Get domain status from cache for non existing domain",
			domainID: testsutil.GenerateUUID(t),
			status:   domains.AllStatus,
			err:      repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			status, err := dc.Status(context.Background(), tc.domainID)
			assert.True(t, errors.Contains(err, tc.err))
			assert.Equal(t, tc.status, status)
		})
	}
}

func TestID(t *testing.T) {
	dc := setupDomainsClient(t)

	route := testRoute
	domainID := testsutil.GenerateUUID(t)
	err := dc.SaveID(context.Background(), route, domainID)
	assert.Nil(t, err, fmt.Sprintf("Unexpected error while trying to save: %s", err))

	cases := []struct {
		desc     string
		route    string
		domainID string
		err      error
	}{
		{
			desc:     "Get domain ID from cache for valid route",
			route:    route,
			domainID: domainID,
			err:      nil,
		},
		{
			desc:     "Get domain ID from cache for non existing route",
			route:    nonExistent,
			domainID: "",
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "Get domain ID from cache with empty route",
			route:    "",
			domainID: "",
			err:      repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			id, err := dc.ID(context.Background(), tc.route)
			assert.True(t, errors.Contains(err, tc.err))
			assert.Equal(t, tc.domainID, id)
		})
	}
}

func TestRemoveStatus(t *testing.T) {
	dc := setupDomainsClient(t)

	domainID := testsutil.GenerateUUID(t)
	err := dc.SaveStatus(context.Background(), domainID, domains.EnabledStatus)
	assert.Nil(t, err, fmt.Sprintf("Unexpected error while trying to save: %s", err))

	cases := []struct {
		desc     string
		domainID string
		err      error
	}{
		{
			desc:     "Remove domain from cache",
			domainID: domainID,
			err:      nil,
		},
		{
			desc:     "Remove domain from cache with empty domain ID",
			domainID: "",
			err:      repoerr.ErrRemoveEntity,
		},
		{
			desc:     "Remove non existing domain from cache",
			domainID: testsutil.GenerateUUID(t),
			err:      nil,
		},
		{
			desc:     "Remove domain from cache with long id",
			domainID: strings.Repeat("a", 513*1024*1024),
			err:      repoerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := dc.RemoveStatus(context.Background(), tc.domainID)
			assert.True(t, errors.Contains(err, tc.err))
			if err == nil {
				_, err = dc.Status(context.Background(), tc.domainID)
				assert.True(t, errors.Contains(err, repoerr.ErrNotFound))
			}
		})
	}
}

func TestRemoveID(t *testing.T) {
	dc := setupDomainsClient(t)

	route := testRoute
	domainID := testsutil.GenerateUUID(t)
	err := dc.SaveID(context.Background(), route, domainID)
	assert.Nil(t, err, fmt.Sprintf("Unexpected error while trying to save: %s", err))

	cases := []struct {
		desc  string
		route string
		err   error
	}{
		{
			desc:  "Remove domain ID from cache",
			route: route,
			err:   nil,
		},
		{
			desc:  "Remove domain ID from cache with empty route",
			route: "",
			err:   repoerr.ErrRemoveEntity,
		},
		{
			desc:  "Remove non existing domain ID from cache",
			route: nonExistent,
			err:   nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := dc.RemoveID(context.Background(), tc.route)
			assert.True(t, errors.Contains(err, tc.err))
			if err == nil {
				id, err := dc.ID(context.Background(), tc.route)
				assert.True(t, errors.Contains(err, repoerr.ErrNotFound))
				assert.Equal(t, "", id)
			}
		})
	}
}
