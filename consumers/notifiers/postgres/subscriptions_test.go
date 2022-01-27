// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"

	notifiers "github.com/mainflux/mainflux/consumers/notifiers"
	"github.com/mainflux/mainflux/consumers/notifiers/postgres"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	owner   = "owner@example.com"
	numSubs = 100
)

func TestSave(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.New(dbMiddleware)

	id1, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	id2, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

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
			err:  errors.ErrConflict,
		},
	}

	for _, tc := range cases {
		id, err := repo.Save(context.Background(), tc.sub)
		assert.Equal(t, tc.id, id, fmt.Sprintf("%s: expected id %s got %s\n", tc.desc, tc.id, id))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

	}
}

func TestView(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
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
			err:  errors.ErrNotFound,
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

	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.New(dbMiddleware)

	var subs []notifiers.Subscription

	for i := 0; i < numSubs; i++ {
		id, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
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
		assert.Equal(t, tc.page, page, fmt.Sprintf("%s: expected page %v got %v\n", tc.desc, tc.page, page))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemove(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
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
