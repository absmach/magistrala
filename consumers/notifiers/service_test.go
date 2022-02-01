// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package notifiers_test

import (
	"context"
	"fmt"
	"testing"

	notifiers "github.com/mainflux/mainflux/consumers/notifiers"
	"github.com/mainflux/mainflux/consumers/notifiers/mocks"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	total        = 100
	exampleUser1 = "email1@example.com"
	exampleUser2 = "email2@example.com"
	invalidUser  = "invalid@example.com"
)

func newService() notifiers.Service {
	repo := mocks.NewRepo(make(map[string]notifiers.Subscription))
	auth := mocks.NewAuth(map[string]string{exampleUser1: exampleUser1, exampleUser2: exampleUser2, invalidUser: invalidUser})
	notifier := mocks.NewNotifier()
	idp := uuid.NewMock()
	from := "exampleFrom"
	return notifiers.New(auth, repo, idp, notifier, from)
}

func TestCreateSubscription(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc  string
		token string
		sub   notifiers.Subscription
		id    string
		err   error
	}{
		{
			desc:  "test success",
			token: exampleUser1,
			sub:   notifiers.Subscription{Contact: exampleUser1, Topic: "valid.topic"},
			id:    uuid.Prefix + fmt.Sprintf("%012d", 1),
			err:   nil,
		},
		{
			desc:  "test already existing",
			token: exampleUser1,
			sub:   notifiers.Subscription{Contact: exampleUser1, Topic: "valid.topic"},
			id:    "",
			err:   errors.ErrConflict,
		},
		{
			desc:  "test with empty token",
			token: "",
			sub:   notifiers.Subscription{Contact: exampleUser1, Topic: "valid.topic"},
			id:    "",
			err:   errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		id, err := svc.CreateSubscription(context.Background(), tc.token, tc.sub)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.id, id, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.id, id))
	}
}

func TestViewSubscription(t *testing.T) {
	svc := newService()
	sub := notifiers.Subscription{Contact: exampleUser1, Topic: "valid.topic"}
	id, err := svc.CreateSubscription(context.Background(), exampleUser1, sub)
	require.Nil(t, err, "Saving a Subscription must succeed")
	sub.ID = id
	sub.OwnerID = exampleUser1

	cases := []struct {
		desc  string
		token string
		id    string
		sub   notifiers.Subscription
		err   error
	}{
		{
			desc:  "test success",
			token: exampleUser1,
			id:    id,
			sub:   sub,
			err:   nil,
		},
		{
			desc:  "test not existing",
			token: exampleUser1,
			id:    "not_exist",
			sub:   notifiers.Subscription{},
			err:   errors.ErrNotFound,
		},
		{
			desc:  "test with empty token",
			token: "",
			id:    id,
			sub:   notifiers.Subscription{},
			err:   errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		sub, err := svc.ViewSubscription(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.sub, sub, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.sub, sub))
	}
}

func TestListSubscriptions(t *testing.T) {
	svc := newService()
	sub := notifiers.Subscription{Contact: exampleUser1, OwnerID: exampleUser1}
	topic := "topic.subtopic"
	var subs []notifiers.Subscription
	for i := 0; i < total; i++ {
		tmp := sub
		token := exampleUser1
		if i%2 == 0 {
			tmp.Contact = exampleUser2
			tmp.OwnerID = exampleUser2
			token = exampleUser2
		}
		tmp.Topic = fmt.Sprintf("%s.%d", topic, i)
		id, err := svc.CreateSubscription(context.Background(), token, tmp)
		require.Nil(t, err, "Saving a Subscription must succeed")
		tmp.ID = id
		subs = append(subs, tmp)
	}

	var offsetSubs []notifiers.Subscription
	for i := 20; i < 40; i += 2 {
		offsetSubs = append(offsetSubs, subs[i])
	}

	cases := []struct {
		desc     string
		token    string
		pageMeta notifiers.PageMetadata
		page     notifiers.Page
		err      error
	}{
		{
			desc:  "test success",
			token: exampleUser1,
			pageMeta: notifiers.PageMetadata{
				Offset: 0,
				Limit:  3,
			},
			err: nil,
			page: notifiers.Page{
				PageMetadata: notifiers.PageMetadata{
					Offset: 0,
					Limit:  3,
				},
				Subscriptions: subs[:3],
				Total:         total,
			},
		},
		{
			desc:  "test not existing",
			token: exampleUser1,
			pageMeta: notifiers.PageMetadata{
				Limit:   10,
				Contact: "empty@example.com",
			},
			page: notifiers.Page{},
			err:  errors.ErrNotFound,
		},
		{
			desc:  "test with empty token",
			token: "",
			pageMeta: notifiers.PageMetadata{
				Offset: 2,
				Limit:  12,
				Topic:  "topic.subtopic.13",
			},
			page: notifiers.Page{},
			err:  errors.ErrAuthentication,
		},
		{
			desc:  "test with topic",
			token: exampleUser1,
			pageMeta: notifiers.PageMetadata{
				Limit: 10,
				Topic: fmt.Sprintf("%s.%d", topic, 4),
			},
			page: notifiers.Page{
				PageMetadata: notifiers.PageMetadata{
					Limit: 10,
					Topic: fmt.Sprintf("%s.%d", topic, 4),
				},
				Subscriptions: subs[4:5],
				Total:         1,
			},
			err: nil,
		},
		{
			desc:  "test with contact and offset",
			token: exampleUser1,
			pageMeta: notifiers.PageMetadata{
				Offset:  10,
				Limit:   10,
				Contact: exampleUser2,
			},
			page: notifiers.Page{
				PageMetadata: notifiers.PageMetadata{
					Offset:  10,
					Limit:   10,
					Contact: exampleUser2,
				},
				Subscriptions: offsetSubs,
				Total:         uint(total / 2),
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListSubscriptions(context.Background(), tc.token, tc.pageMeta)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.page, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.page, page))
	}
}

func TestRemoveSubscription(t *testing.T) {
	svc := newService()
	sub := notifiers.Subscription{Contact: exampleUser1, Topic: "valid.topic"}
	id, err := svc.CreateSubscription(context.Background(), exampleUser1, sub)
	require.Nil(t, err, "Saving a Subscription must succeed")
	sub.ID = id
	sub.OwnerID = exampleUser1

	cases := []struct {
		desc  string
		token string
		id    string
		err   error
	}{
		{
			desc:  "test success",
			token: exampleUser1,
			id:    id,
			err:   nil,
		},
		{
			desc:  "test not existing",
			token: exampleUser1,
			id:    "not_exist",
			err:   nil,
		},
		{
			desc:  "test with empty token",
			token: "",
			id:    id,
			err:   errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveSubscription(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestConsume(t *testing.T) {
	svc := newService()
	sub := notifiers.Subscription{
		Contact: exampleUser1,
		OwnerID: exampleUser1,
		Topic:   "topic.subtopic",
	}
	for i := 0; i < total; i++ {
		tmp := sub
		tmp.Contact = fmt.Sprintf("contact%d@example.com", i)
		if i%2 == 0 {
			tmp.Topic = fmt.Sprintf("%s-2", sub.Topic)
		}
		_, err := svc.CreateSubscription(context.Background(), exampleUser1, tmp)
		require.Nil(t, err, "Saving a Subscription must succeed")
	}

	sub.Contact = invalidUser
	sub.Topic = fmt.Sprintf("%s-2", sub.Topic)
	_, err := svc.CreateSubscription(context.Background(), exampleUser1, sub)
	require.Nil(t, err, "Saving a Subscription must succeed")

	msg := messaging.Message{
		Channel:  "topic",
		Subtopic: "subtopic",
	}
	errMsg := messaging.Message{
		Channel:  "topic",
		Subtopic: "subtopic-2",
	}

	cases := []struct {
		desc string
		msg  messaging.Message
		err  error
	}{
		{
			desc: "test success",
			msg:  msg,
		},
		{
			desc: "test fail",
			msg:  errMsg,
			err:  notifiers.ErrNotify,
		},
	}

	for _, tc := range cases {
		err := svc.Consume(tc.msg)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
