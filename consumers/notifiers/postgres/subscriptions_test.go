// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/absmach/supermq/consumers/notifiers"
	"github.com/absmach/supermq/consumers/notifiers/postgres"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

const (
	owner   = "owner@example.com"
	numSubs = 100
)

var tracer = otel.Tracer("tests")

func TestSave(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db, tracer)
	repo := postgres.New(dbMiddleware)

	id1, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	id2, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	sub1 := notifiers.Subscription{
		OwnerID: id1,
		ID:      id1,
		Contact: owner,
		Topic:   "topic.subtopic",
	}

	sub2 := sub1
	sub2.ID = id2

	cases := []struct {
		desc string
		sub  notifiers.Subscription
		id   string
		err  error
	}{
		{
			desc: "save successfully",
			sub:  sub1,
			id:   id1,
			err:  nil,
		},
		{
			desc: "save duplicate",
			sub:  sub2,
			id:   "",
			err:  repoerr.ErrConflict,
		},
	}

	for _, tc := range cases {
		id, err := repo.Save(context.Background(), tc.sub)
		assert.Equal(t, tc.id, id, fmt.Sprintf("%s: expected id %s got %s\n", tc.desc, tc.id, id))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestView(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db, tracer)
	repo := postgres.New(dbMiddleware)

	id, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got an error creating id: %s", err))

	sub := notifiers.Subscription{
		OwnerID: id,
		ID:      id,
		Contact: owner,
		Topic:   "view.subtopic",
	}

	ret, err := repo.Save(context.Background(), sub)
	require.Nil(t, err, fmt.Sprintf("creating subscription must not fail: %s", err))
	require.Equal(t, id, ret, fmt.Sprintf("provided id %s must be the same as the returned id %s", id, ret))

	cases := []struct {
		desc string
		sub  notifiers.Subscription
		id   string
		err  error
	}{
		{
			desc: "retrieve successfully",
			sub:  sub,
			id:   id,
			err:  nil,
		},
		{
			desc: "retrieve not existing",
			sub:  notifiers.Subscription{},
			id:   "non-existing",
			err:  repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		sub, err := repo.Retrieve(context.Background(), tc.id)
		assert.Equal(t, tc.sub, sub, fmt.Sprintf("%s: expected sub %v got %v\n", tc.desc, tc.sub, sub))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveAll(t *testing.T) {
	_, err := db.Exec("DELETE FROM subscriptions")
	require.Nil(t, err, fmt.Sprintf("cleanup must not fail: %s", err))

	dbMiddleware := postgres.NewDatabase(db, tracer)
	repo := postgres.New(dbMiddleware)

	var subs []notifiers.Subscription

	for i := 0; i < numSubs; i++ {
		id, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		sub := notifiers.Subscription{
			OwnerID: "owner",
			ID:      id,
			Contact: owner,
			Topic:   fmt.Sprintf("list.subtopic.%d", i),
		}

		ret, err := repo.Save(context.Background(), sub)
		require.Nil(t, err, fmt.Sprintf("creating subscription must not fail: %s", err))
		require.Equal(t, id, ret, fmt.Sprintf("provided id %s must be the same as the returned id %s", id, ret))
		subs = append(subs, sub)
	}

	cases := []struct {
		desc     string
		pageMeta notifiers.PageMetadata
		page     notifiers.Page
		err      error
	}{
		{
			desc: "retrieve successfully",
			pageMeta: notifiers.PageMetadata{
				Offset: 10,
				Limit:  2,
			},
			page: notifiers.Page{
				Total: numSubs,
				PageMetadata: notifiers.PageMetadata{
					Offset: 10,
					Limit:  2,
				},
				Subscriptions: subs[10:12],
			},
			err: nil,
		},
		{
			desc: "retrieve with contact",
			pageMeta: notifiers.PageMetadata{
				Offset:  10,
				Limit:   2,
				Contact: owner,
			},
			page: notifiers.Page{
				Total: numSubs,
				PageMetadata: notifiers.PageMetadata{
					Offset:  10,
					Limit:   2,
					Contact: owner,
				},
				Subscriptions: subs[10:12],
			},
			err: nil,
		},
		{
			desc: "retrieve with topic",
			pageMeta: notifiers.PageMetadata{
				Offset: 0,
				Limit:  2,
				Topic:  "list.subtopic.11",
			},
			page: notifiers.Page{
				Total: 1,
				PageMetadata: notifiers.PageMetadata{
					Offset: 0,
					Limit:  2,
					Topic:  "list.subtopic.11",
				},
				Subscriptions: subs[11:12],
			},
			err: nil,
		},
		{
			desc: "retrieve with no limit",
			pageMeta: notifiers.PageMetadata{
				Offset: 0,
				Limit:  -1,
			},
			page: notifiers.Page{
				Total: numSubs,
				PageMetadata: notifiers.PageMetadata{
					Limit: -1,
				},
				Subscriptions: subs,
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		page, err := repo.RetrieveAll(context.Background(), tc.pageMeta)
		assert.Equal(t, tc.page, page, fmt.Sprintf("%s: got unexpected page\n", tc.desc))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemove(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db, tracer)
	repo := postgres.New(dbMiddleware)
	id, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got an error creating id: %s", err))
	sub := notifiers.Subscription{
		OwnerID: id,
		ID:      id,
		Contact: owner,
		Topic:   "remove.subtopic.%d",
	}

	ret, err := repo.Save(context.Background(), sub)
	require.Nil(t, err, fmt.Sprintf("creating subscription must not fail: %s", err))
	require.Equal(t, id, ret, fmt.Sprintf("provided id %s must be the same as the returned id %s", id, ret))

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "remove successfully",
			id:   id,
			err:  nil,
		},
		{
			desc: "remove not existing",
			id:   "empty",
			err:  nil,
		},
	}

	for _, tc := range cases {
		err := repo.Remove(context.Background(), tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
