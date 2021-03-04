// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package auth_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/auth/jwt"
	"github.com/mainflux/mainflux/auth/mocks"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var idProvider = uuid.New()

const (
	secret      = "secret"
	email       = "test@example.com"
	id          = "testID"
	groupName   = "mfx"
	description = "Description"
)

func newService() auth.Service {
	repo := mocks.NewKeyRepository()
	groupRepo := mocks.NewGroupRepository()
	idProvider := uuid.NewMock()
	t := jwt.New(secret)
	return auth.New(repo, groupRepo, idProvider, t)
}

func TestIssue(t *testing.T) {
	svc := newService()
	_, secret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.UserKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	cases := []struct {
		desc  string
		key   auth.Key
		token string
		err   error
	}{
		{
			desc: "issue user key",
			key: auth.Key{
				Type:     auth.UserKey,
				IssuedAt: time.Now(),
			},
			token: secret,
			err:   nil,
		},
		{
			desc: "issue user key with no time",
			key: auth.Key{
				Type: auth.UserKey,
			},
			token: secret,
			err:   auth.ErrInvalidKeyIssuedAt,
		},
		{
			desc: "issue API key",
			key: auth.Key{
				Type:     auth.APIKey,
				IssuedAt: time.Now(),
			},
			token: secret,
			err:   nil,
		},
		{
			desc: "issue API key unauthorized",
			key: auth.Key{
				Type:     auth.APIKey,
				IssuedAt: time.Now(),
			},
			token: "invalid",
			err:   auth.ErrUnauthorizedAccess,
		},
		{
			desc: "issue API key with no time",
			key: auth.Key{
				Type: auth.APIKey,
			},
			token: secret,
			err:   auth.ErrInvalidKeyIssuedAt,
		},
		{
			desc: "issue recovery key",
			key: auth.Key{
				Type:     auth.RecoveryKey,
				IssuedAt: time.Now(),
			},
			token: "",
			err:   nil,
		},
		{
			desc: "issue recovery with no issue time",
			key: auth.Key{
				Type: auth.RecoveryKey,
			},
			token: secret,
			err:   auth.ErrInvalidKeyIssuedAt,
		},
	}

	for _, tc := range cases {
		_, _, err := svc.Issue(context.Background(), tc.token, tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRevoke(t *testing.T) {
	svc := newService()
	_, secret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.UserKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	key := auth.Key{
		Type:     auth.APIKey,
		IssuedAt: time.Now(),
		IssuerID: id,
		Subject:  email,
	}
	newKey, _, err := svc.Issue(context.Background(), secret, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing user's key expected to succeed: %s", err))

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "revoke user key",
			id:    newKey.ID,
			token: secret,
			err:   nil,
		},
		{
			desc:  "revoke non-existing user key",
			id:    newKey.ID,
			token: secret,
			err:   nil,
		},
		{
			desc:  "revoke unauthorized",
			id:    newKey.ID,
			token: "",
			err:   auth.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		err := svc.Revoke(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieve(t *testing.T) {
	svc := newService()
	_, secret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.UserKey, IssuedAt: time.Now(), Subject: email, IssuerID: id})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	key := auth.Key{
		ID:       "id",
		Type:     auth.APIKey,
		IssuerID: id,
		Subject:  email,
		IssuedAt: time.Now(),
	}

	_, userToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.UserKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing user key expected to succeed: %s", err))

	apiKey, apiToken, err := svc.Issue(context.Background(), secret, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing user's key expected to succeed: %s", err))

	_, resetToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.RecoveryKey, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing reset key expected to succeed: %s", err))

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "retrieve user key",
			id:    apiKey.ID,
			token: userToken,
			err:   nil,
		},
		{
			desc:  "retrieve non-existing user key",
			id:    "invalid",
			token: userToken,
			err:   auth.ErrNotFound,
		},
		{
			desc:  "retrieve unauthorized",
			id:    apiKey.ID,
			token: "wrong",
			err:   auth.ErrUnauthorizedAccess,
		},
		{
			desc:  "retrieve with API token",
			id:    apiKey.ID,
			token: apiToken,
			err:   auth.ErrUnauthorizedAccess,
		},
		{
			desc:  "retrieve with reset token",
			id:    apiKey.ID,
			token: resetToken,
			err:   auth.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		_, err := svc.RetrieveKey(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestIdentify(t *testing.T) {
	svc := newService()

	_, loginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.UserKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	_, recoverySecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.RecoveryKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing reset key expected to succeed: %s", err))

	_, apiSecret, err := svc.Issue(context.Background(), loginSecret, auth.Key{Type: auth.APIKey, IssuerID: id, Subject: email, IssuedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute)})
	assert.Nil(t, err, fmt.Sprintf("Issuing user key expected to succeed: %s", err))

	exp1 := time.Now().Add(-2 * time.Second)
	_, expSecret, err := svc.Issue(context.Background(), loginSecret, auth.Key{Type: auth.APIKey, IssuedAt: time.Now(), ExpiresAt: exp1})
	assert.Nil(t, err, fmt.Sprintf("Issuing expired user key expected to succeed: %s", err))

	_, invalidSecret, err := svc.Issue(context.Background(), loginSecret, auth.Key{Type: 22, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing user key expected to succeed: %s", err))

	cases := []struct {
		desc string
		key  string
		idt  auth.Identity
		err  error
	}{
		{
			desc: "identify login key",
			key:  loginSecret,
			idt:  auth.Identity{id, email},
			err:  nil,
		},
		{
			desc: "identify recovery key",
			key:  recoverySecret,
			idt:  auth.Identity{id, email},
			err:  nil,
		},
		{
			desc: "identify API key",
			key:  apiSecret,
			idt:  auth.Identity{id, email},
			err:  nil,
		},
		{
			desc: "identify expired API key",
			key:  expSecret,
			idt:  auth.Identity{},
			err:  auth.ErrAPIKeyExpired,
		},
		{
			desc: "identify expired key",
			key:  invalidSecret,
			idt:  auth.Identity{},
			err:  auth.ErrUnauthorizedAccess,
		},
		{
			desc: "identify invalid key",
			key:  "invalid",
			idt:  auth.Identity{},
			err:  auth.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		idt, err := svc.Identify(context.Background(), tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.idt, idt, fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.idt, idt))
	}
}

func TestCreateGroup(t *testing.T) {
	svc := newService()
	_, secret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.UserKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	key := auth.Key{
		ID:       "id",
		Type:     auth.APIKey,
		IssuerID: id,
		Subject:  email,
		IssuedAt: time.Now(),
	}

	_, apiToken, err := svc.Issue(context.Background(), secret, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing user's key expected to succeed: %s", err))

	group := auth.Group{
		Name:        "Group",
		Description: description,
	}

	parentGroup := auth.Group{
		Name:        "ParentGroup",
		Description: description,
	}

	parent, err := svc.CreateGroup(context.Background(), apiToken, parentGroup)
	assert.Nil(t, err, fmt.Sprintf("Creating parent group expected to succeed: %s", err))

	cases := []struct {
		desc  string
		group auth.Group
		err   error
	}{
		{
			desc:  "create new group",
			group: group,
			err:   nil,
		},
		{
			desc:  "create group with existing name",
			group: group,
			err:   nil,
		},
		{
			desc: "create group with parent",
			group: auth.Group{
				Name:     groupName,
				ParentID: parent.ID,
			},
			err: nil,
		},
		{
			desc: "create group with invalid parent",
			group: auth.Group{
				Name:     groupName,
				ParentID: "xxxxxxxxxx",
			},
			err: auth.ErrCreateGroup,
		},
	}

	for _, tc := range cases {
		_, err := svc.CreateGroup(context.Background(), apiToken, tc.group)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateGroup(t *testing.T) {
	svc := newService()
	_, secret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.UserKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	key := auth.Key{
		ID:       "id",
		Type:     auth.APIKey,
		IssuerID: id,
		Subject:  email,
		IssuedAt: time.Now(),
	}

	_, apiToken, err := svc.Issue(context.Background(), secret, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing user's key expected to succeed: %s", err))

	group := auth.Group{
		Name:        "Group",
		Description: description,
		Metadata: auth.GroupMetadata{
			"field": "value",
		},
	}

	group, err = svc.CreateGroup(context.Background(), apiToken, group)
	assert.Nil(t, err, fmt.Sprintf("Creating parent group failed: %s", err))

	cases := []struct {
		desc  string
		group auth.Group
		err   error
	}{
		{
			desc: "update group",
			group: auth.Group{
				ID:          group.ID,
				Name:        "NewName",
				Description: "NewDescription",
				Metadata: auth.GroupMetadata{
					"field": "value2",
				},
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		g, err := svc.UpdateGroup(context.Background(), apiToken, tc.group)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, g.ID, tc.group.ID, fmt.Sprintf("ID: expected %s got %s\n", g.ID, tc.group.ID))
		assert.Equal(t, g.Name, tc.group.Name, fmt.Sprintf("Name: expected %s got %s\n", g.Name, tc.group.Name))
		assert.Equal(t, g.Description, tc.group.Description, fmt.Sprintf("Description: expected %s got %s\n", g.Description, tc.group.Description))
		assert.Equal(t, g.Metadata["field"], g.Metadata["field"], fmt.Sprintf("Metadata: expected %s got %s\n", g.Metadata, tc.group.Metadata))
	}

}

func TestViewGroup(t *testing.T) {
	svc := newService()
	_, secret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.UserKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	key := auth.Key{
		ID:       "id",
		Type:     auth.APIKey,
		IssuerID: id,
		Subject:  email,
		IssuedAt: time.Now(),
	}

	_, apiToken, err := svc.Issue(context.Background(), secret, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing user's key expected to succeed: %s", err))

	group := auth.Group{
		Name:        "Group",
		Description: description,
		Metadata: auth.GroupMetadata{
			"field": "value",
		},
	}

	group, err = svc.CreateGroup(context.Background(), apiToken, group)
	assert.Nil(t, err, fmt.Sprintf("Creating parent group failed: %s", err))

	cases := []struct {
		desc    string
		token   string
		groupID string
		err     error
	}{
		{

			desc:    "view group",
			token:   apiToken,
			groupID: group.ID,
			err:     nil,
		},
		{
			desc:    "view group with unauthorized token",
			token:   "wrongtoken",
			groupID: group.ID,
			err:     auth.ErrUnauthorizedAccess,
		},
		{
			desc:    "view group for wrong id",
			token:   apiToken,
			groupID: "wrong",
			err:     auth.ErrGroupNotFound,
		},
	}

	for _, tc := range cases {
		_, err := svc.ViewGroup(context.Background(), tc.token, tc.groupID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestListGroups(t *testing.T) {
	svc := newService()
	_, secret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.UserKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	key := auth.Key{
		ID:       "id",
		Type:     auth.APIKey,
		IssuerID: id,
		Subject:  email,
		IssuedAt: time.Now(),
	}

	_, apiToken, err := svc.Issue(context.Background(), secret, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing user's key expected to succeed: %s", err))

	group := auth.Group{
		Description: description,
		Metadata: auth.GroupMetadata{
			"field": "value",
		},
	}
	n := uint64(10)
	parentID := ""
	for i := uint64(0); i < n; i++ {
		group.Name = fmt.Sprintf("Group%d", i)
		group.ParentID = parentID
		g, err := svc.CreateGroup(context.Background(), apiToken, group)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
		parentID = g.ID
	}

	cases := map[string]struct {
		token    string
		level    uint64
		size     uint64
		metadata auth.GroupMetadata
		err      error
	}{
		"list all groups": {
			token: apiToken,
			level: 5,
			size:  n,
			err:   nil,
		},
		"list groups for level 1": {
			token: apiToken,
			level: 1,
			size:  n,
			err:   nil,
		},
		"list all groups with wrong token": {
			token: "wrongToken",
			level: 5,
			size:  0,
			err:   auth.ErrUnauthorizedAccess,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListGroups(context.Background(), tc.token, auth.PageMetadata{Level: tc.level, Metadata: tc.metadata})
		size := uint64(len(page.Groups))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}

}

func TestListChildren(t *testing.T) {
	svc := newService()
	_, secret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.UserKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	key := auth.Key{
		ID:       "id",
		Type:     auth.APIKey,
		IssuerID: id,
		Subject:  email,
		IssuedAt: time.Now(),
	}

	_, apiToken, err := svc.Issue(context.Background(), secret, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing user's key expected to succeed: %s", err))

	group := auth.Group{
		Description: description,
		Metadata: auth.GroupMetadata{
			"field": "value",
		},
	}
	n := uint64(10)
	parentID := ""
	groupIDs := make([]string, n)
	for i := uint64(0); i < n; i++ {
		group.Name = fmt.Sprintf("Group%d", i)
		group.ParentID = parentID
		g, err := svc.CreateGroup(context.Background(), apiToken, group)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
		parentID = g.ID
		groupIDs[i] = g.ID
	}

	cases := map[string]struct {
		token    string
		level    uint64
		size     uint64
		id       string
		metadata auth.GroupMetadata
		err      error
	}{
		"list all children": {
			token: apiToken,
			level: 5,
			id:    groupIDs[0],
			size:  n,
			err:   nil,
		},
		"list all groups with wrong token": {
			token: "wrongToken",
			level: 5,
			size:  0,
			err:   auth.ErrUnauthorizedAccess,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListChildren(context.Background(), tc.token, tc.id, auth.PageMetadata{Level: tc.level, Metadata: tc.metadata})
		size := uint64(len(page.Groups))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListParents(t *testing.T) {
	svc := newService()
	_, secret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.UserKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	key := auth.Key{
		ID:       "id",
		Type:     auth.APIKey,
		IssuerID: id,
		Subject:  email,
		IssuedAt: time.Now(),
	}

	_, apiToken, err := svc.Issue(context.Background(), secret, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing user's key expected to succeed: %s", err))

	group := auth.Group{
		Description: description,
		Metadata: auth.GroupMetadata{
			"field": "value",
		},
	}
	n := uint64(10)
	parentID := ""
	groupIDs := make([]string, n)
	for i := uint64(0); i < n; i++ {
		group.Name = fmt.Sprintf("Group%d", i)
		group.ParentID = parentID
		g, err := svc.CreateGroup(context.Background(), apiToken, group)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
		parentID = g.ID
		groupIDs[i] = g.ID
	}

	cases := map[string]struct {
		token    string
		level    uint64
		size     uint64
		id       string
		metadata auth.GroupMetadata
		err      error
	}{
		"list all parents": {
			token: apiToken,
			level: 5,
			id:    groupIDs[n-1],
			size:  n,
			err:   nil,
		},
		"list all parents with wrong token": {
			token: "wrongToken",
			level: 5,
			size:  0,
			err:   auth.ErrUnauthorizedAccess,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListParents(context.Background(), tc.token, tc.id, auth.PageMetadata{Level: tc.level, Metadata: tc.metadata})
		size := uint64(len(page.Groups))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListMembers(t *testing.T) {
	svc := newService()
	_, secret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.UserKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	key := auth.Key{
		ID:       "id",
		Type:     auth.APIKey,
		IssuerID: id,
		Subject:  email,
		IssuedAt: time.Now(),
	}

	_, apiToken, err := svc.Issue(context.Background(), secret, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing user's key expected to succeed: %s", err))

	group := auth.Group{
		Description: description,
		Metadata: auth.GroupMetadata{
			"field": "value",
		},
	}
	g, err := svc.CreateGroup(context.Background(), apiToken, group)
	assert.Nil(t, err, fmt.Sprintf("Creating group expected to succeed: %s", err))
	group.ID = g.ID

	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		uid, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		err = svc.Assign(context.Background(), apiToken, group.ID, "things", uid)
		require.Nil(t, err, fmt.Sprintf("Assign member expected to succeed: %s\n", err))
	}

	cases := map[string]struct {
		token    string
		size     uint64
		offset   uint64
		limit    uint64
		group    auth.Group
		metadata auth.GroupMetadata
		err      error
	}{
		"list all members": {
			token:  apiToken,
			offset: 0,
			limit:  n,
			group:  group,
			size:   n,
			err:    nil,
		},
		"list half members": {
			token:  apiToken,
			offset: n / 2,
			limit:  n,
			group:  group,
			size:   n / 2,
			err:    nil,
		},
		"list all members with wrong token": {
			token:  "wrongToken",
			offset: 0,
			limit:  n,
			size:   0,
			err:    auth.ErrUnauthorizedAccess,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListMembers(context.Background(), tc.token, tc.group.ID, "things", auth.PageMetadata{Offset: tc.offset, Limit: tc.limit, Metadata: tc.metadata})
		size := uint64(len(page.Members))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}

}

func TestListMemberships(t *testing.T) {
	svc := newService()
	_, secret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.UserKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	key := auth.Key{
		ID:       "id",
		Type:     auth.APIKey,
		IssuerID: id,
		Subject:  email,
		IssuedAt: time.Now(),
	}

	_, apiToken, err := svc.Issue(context.Background(), secret, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing user's key expected to succeed: %s", err))

	group := auth.Group{
		Description: description,
		Metadata: auth.GroupMetadata{
			"field": "value",
		},
	}

	memberID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		group.Name = fmt.Sprintf("Group%d", i)
		g, err := svc.CreateGroup(context.Background(), apiToken, group)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		err = svc.Assign(context.Background(), apiToken, g.ID, "things", memberID)
		require.Nil(t, err, fmt.Sprintf("Assign member expected to succeed: %s\n", err))
	}

	cases := map[string]struct {
		token    string
		size     uint64
		offset   uint64
		limit    uint64
		group    auth.Group
		metadata auth.GroupMetadata
		err      error
	}{
		"list all members": {
			token:  apiToken,
			offset: 0,
			limit:  n,
			group:  group,
			size:   n,
			err:    nil,
		},
		"list half members": {
			token:  apiToken,
			offset: n / 2,
			limit:  n,
			group:  group,
			size:   n / 2,
			err:    nil,
		},
		"list all members with wrong token": {
			token:  "wrongToken",
			offset: 0,
			limit:  n,
			size:   0,
			err:    auth.ErrUnauthorizedAccess,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListMemberships(context.Background(), tc.token, memberID, auth.PageMetadata{Limit: tc.limit, Offset: tc.offset, Metadata: tc.metadata})
		size := uint64(len(page.Groups))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRemoveGroup(t *testing.T) {
	svc := newService()
	_, secret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.UserKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	key := auth.Key{
		ID:       "id",
		Type:     auth.APIKey,
		IssuerID: id,
		Subject:  email,
		IssuedAt: time.Now(),
	}

	_, apiToken, err := svc.Issue(context.Background(), secret, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing user's key expected to succeed: %s", err))

	uid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	creationTime := time.Now().UTC()
	group := auth.Group{
		Name:      groupName,
		OwnerID:   uid,
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
	}

	group, err = svc.CreateGroup(context.Background(), apiToken, group)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	err = svc.RemoveGroup(context.Background(), "wrongToken", group.ID)
	assert.True(t, errors.Contains(err, auth.ErrUnauthorizedAccess), fmt.Sprintf("Unauthorized access: expected %v got %v", auth.ErrUnauthorizedAccess, err))

	err = svc.RemoveGroup(context.Background(), apiToken, "wrongID")
	assert.True(t, errors.Contains(err, auth.ErrGroupNotFound), fmt.Sprintf("Remove group with wrong id: expected %v got %v", auth.ErrGroupNotFound, err))

	gp, err := svc.ListGroups(context.Background(), apiToken, auth.PageMetadata{Level: auth.MaxLevel})
	require.Nil(t, err, fmt.Sprintf("list groups unexpected error: %s", err))
	assert.True(t, gp.Total == 1, fmt.Sprintf("retrieve members of a group: expected %d got %d\n", 1, gp.Total))

	err = svc.RemoveGroup(context.Background(), apiToken, group.ID)
	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("Unauthorized access: expected %v got %v", nil, err))

	gp, err = svc.ListGroups(context.Background(), apiToken, auth.PageMetadata{Level: auth.MaxLevel})
	require.Nil(t, err, fmt.Sprintf("list groups save unexpected error: %s", err))
	assert.True(t, gp.Total == 0, fmt.Sprintf("retrieve members of a group: expected %d got %d\n", 0, gp.Total))

}

func TestAssign(t *testing.T) {
	svc := newService()
	_, secret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.UserKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	key := auth.Key{
		ID:       "id",
		Type:     auth.APIKey,
		IssuerID: id,
		Subject:  email,
		IssuedAt: time.Now(),
	}

	_, apiToken, err := svc.Issue(context.Background(), secret, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing user's key expected to succeed: %s", err))

	uid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	creationTime := time.Now().UTC()
	group := auth.Group{
		Name:      groupName + "Updated",
		OwnerID:   uid,
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
	}

	group, err = svc.CreateGroup(context.Background(), apiToken, group)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	mid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	err = svc.Assign(context.Background(), apiToken, group.ID, "things", mid)
	require.Nil(t, err, fmt.Sprintf("member assign save unexpected error: %s", err))

	mp, err := svc.ListMembers(context.Background(), apiToken, group.ID, "things", auth.PageMetadata{Offset: 0, Limit: 10})
	require.Nil(t, err, fmt.Sprintf("member assign save unexpected error: %s", err))
	assert.True(t, mp.Total == 1, fmt.Sprintf("retrieve members of a group: expected %d got %d\n", 1, mp.Total))

	err = svc.Assign(context.Background(), "wrongToken", group.ID, "things", mid)
	assert.True(t, errors.Contains(err, auth.ErrUnauthorizedAccess), fmt.Sprintf("Unauthorized access: expected %v got %v", auth.ErrUnauthorizedAccess, err))

}

func TestUnassign(t *testing.T) {
	svc := newService()
	_, secret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.UserKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	key := auth.Key{
		ID:       "id",
		Type:     auth.APIKey,
		IssuerID: id,
		Subject:  email,
		IssuedAt: time.Now(),
	}

	_, apiToken, err := svc.Issue(context.Background(), secret, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing user's key expected to succeed: %s", err))

	uid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	creationTime := time.Now().UTC()
	group := auth.Group{
		Name:      groupName + "Updated",
		OwnerID:   uid,
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
	}

	group, err = svc.CreateGroup(context.Background(), apiToken, group)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	mid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	err = svc.Assign(context.Background(), apiToken, group.ID, "things", mid)
	require.Nil(t, err, fmt.Sprintf("member assign save unexpected error: %s", err))

	mp, err := svc.ListMembers(context.Background(), apiToken, group.ID, "things", auth.PageMetadata{Limit: 10, Offset: 0})
	require.Nil(t, err, fmt.Sprintf("member assign save unexpected error: %s", err))
	assert.True(t, mp.Total == 1, fmt.Sprintf("retrieve members of a group: expected %d got %d\n", 1, mp.Total))

	err = svc.Unassign(context.Background(), apiToken, group.ID, mid)
	require.Nil(t, err, fmt.Sprintf("member unassign save unexpected error: %s", err))

	mp, err = svc.ListMembers(context.Background(), apiToken, group.ID, "things", auth.PageMetadata{Limit: 10, Offset: 0})
	require.Nil(t, err, fmt.Sprintf("member assign save unexpected error: %s", err))
	assert.True(t, mp.Total == 0, fmt.Sprintf("retrieve members of a group: expected %d got %d\n", 0, mp.Total))

	err = svc.Unassign(context.Background(), "wrongToken", group.ID, mid)
	assert.True(t, errors.Contains(err, auth.ErrUnauthorizedAccess), fmt.Sprintf("Unauthorized access: expected %v got %v", auth.ErrUnauthorizedAccess, err))

	err = svc.Unassign(context.Background(), apiToken, group.ID, mid)
	assert.True(t, errors.Contains(err, auth.ErrGroupNotFound), fmt.Sprintf("Unauthorized access: expected %v got %v", nil, err))
}
