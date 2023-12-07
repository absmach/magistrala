// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/absmach/magistrala/invitations"
	"github.com/absmach/magistrala/invitations/postgres"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	invalidUUID = strings.Repeat("a", 37)
	validToken  = strings.Repeat("a", 1024)
	relation    = "relation"
)

func TestInvitationCreate(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM invitations")
		require.Nil(t, err, fmt.Sprintf("clean invitations unexpected error: %s", err))
	})
	repo := postgres.NewRepository(database)

	domain := generateUUID(t)
	userID := generateUUID(t)

	cases := []struct {
		desc       string
		invitation invitations.Invitation
		err        error
	}{
		{
			desc: "add new invitation successfully",
			invitation: invitations.Invitation{
				InvitedBy: generateUUID(t),
				UserID:    userID,
				Domain:    domain,
				Token:     validToken,
				Relation:  relation,
				CreatedAt: time.Now(),
			},
			err: nil,
		},
		{
			desc: "add new invitation with an confirmed_at date",
			invitation: invitations.Invitation{
				InvitedBy:   generateUUID(t),
				UserID:      generateUUID(t),
				Domain:      generateUUID(t),
				Token:       validToken,
				Relation:    relation,
				CreatedAt:   time.Now(),
				ConfirmedAt: time.Now(),
			},
			err: nil,
		},
		{
			desc: "add invitation with duplicate invitation",
			invitation: invitations.Invitation{
				InvitedBy: generateUUID(t),
				UserID:    userID,
				Domain:    domain,
				Token:     validToken,
				Relation:  relation,
				CreatedAt: time.Now(),
			},
			err: repoerr.ErrConflict,
		},
		{
			desc: "add invitation with invalid invitation invited_by",
			invitation: invitations.Invitation{
				InvitedBy: invalidUUID,
				UserID:    generateUUID(t),
				Domain:    generateUUID(t),
				Token:     validToken,
				Relation:  relation,
				CreatedAt: time.Now(),
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "add invitation with invalid invitation relation",
			invitation: invitations.Invitation{
				InvitedBy: generateUUID(t),
				UserID:    generateUUID(t),
				Domain:    generateUUID(t),
				Token:     validToken,
				Relation:  strings.Repeat("a", 255),
				CreatedAt: time.Now(),
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "add invitation with invalid invitation domain",
			invitation: invitations.Invitation{
				InvitedBy: generateUUID(t),
				UserID:    generateUUID(t),
				Domain:    invalidUUID,
				Token:     validToken,
				Relation:  relation,
				CreatedAt: time.Now(),
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "add invitation with invalid invitation user id",
			invitation: invitations.Invitation{
				InvitedBy: generateUUID(t),
				UserID:    invalidUUID,
				Domain:    generateUUID(t),
				Token:     validToken,
				Relation:  relation,
				CreatedAt: time.Now(),
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "add invitation with empty invitation domain",
			invitation: invitations.Invitation{
				InvitedBy: generateUUID(t),
				UserID:    generateUUID(t),
				Token:     validToken,
				Relation:  relation,
				CreatedAt: time.Now(),
			},
			err: nil,
		},
		{
			desc: "add invitation with empty invitation user id",
			invitation: invitations.Invitation{
				InvitedBy: generateUUID(t),
				Domain:    generateUUID(t),
				Token:     validToken,
				Relation:  relation,
				CreatedAt: time.Now(),
			},
			err: nil,
		},
		{
			desc: "add invitation with empty invitation invited_by",
			invitation: invitations.Invitation{
				Domain:    generateUUID(t),
				UserID:    generateUUID(t),
				Token:     validToken,
				Relation:  relation,
				CreatedAt: time.Now(),
			},
			err: nil,
		},
		{
			desc: "add invitation with empty invitation token",
			invitation: invitations.Invitation{
				InvitedBy: generateUUID(t),
				Domain:    generateUUID(t),
				UserID:    generateUUID(t),
				Relation:  relation,
				CreatedAt: time.Now(),
			},
			err: nil,
		},
	}
	for _, tc := range cases {
		switch err := repo.Create(context.Background(), tc.invitation); {
		case err == nil:
			assert.Nil(t, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		default:
			assert.ErrorIs(t, err, tc.err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		}
	}
}

func TestInvitationRetrieve(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM invitations")
		require.Nil(t, err, fmt.Sprintf("clean invitations unexpected error: %s", err))
	})
	repo := postgres.NewRepository(database)

	invitation := invitations.Invitation{
		InvitedBy: generateUUID(t),
		UserID:    generateUUID(t),
		Domain:    generateUUID(t),
		Token:     validToken,
		Relation:  relation,
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	err := repo.Create(context.Background(), invitation)
	require.Nil(t, err, fmt.Sprintf("create invitation unexpected error: %s", err))
	invitation.Token = ""

	cases := []struct {
		desc     string
		userID   string
		domainID string
		response invitations.Invitation
		err      error
	}{
		{
			desc:     "retrieve invitations successfully",
			userID:   invitation.UserID,
			domainID: invitation.Domain,
			response: invitation,
			err:      nil,
		},
		{
			desc:     "retrieve invitations with invalid invitation user id",
			userID:   generateUUID(t),
			domainID: invitation.Domain,
			response: invitations.Invitation{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve invitations with invalid invitation domain",
			userID:   invitation.UserID,
			domainID: generateUUID(t),
			response: invitations.Invitation{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve invitations with invalid invitation user id and domain",
			userID:   generateUUID(t),
			domainID: generateUUID(t),
			response: invitations.Invitation{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve invitations with empty invitation user id",
			userID:   "",
			domainID: invitation.Domain,
			response: invitations.Invitation{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve invitations with empty invitation domain",
			userID:   invitation.UserID,
			domainID: "",
			response: invitations.Invitation{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve invitations with empty invitation user id and domain",
			userID:   "",
			domainID: "",
			response: invitations.Invitation{},
			err:      repoerr.ErrNotFound,
		},
	}
	for _, tc := range cases {
		page, err := repo.Retrieve(context.Background(), tc.userID, tc.domainID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("desc: %s\n", tc.desc))
	}
}

func TestInvitationRetrieveAll(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM invitations")
		require.Nil(t, err, fmt.Sprintf("clean invitations unexpected error: %s", err))
	})
	repo := postgres.NewRepository(database)

	num := 200

	var items []invitations.Invitation
	for i := 0; i < num; i++ {
		invitation := invitations.Invitation{
			InvitedBy: generateUUID(t),
			UserID:    generateUUID(t),
			Domain:    generateUUID(t),
			Token:     validToken,
			Relation:  fmt.Sprintf("%s-%d", relation, i),
			CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		}
		err := repo.Create(context.Background(), invitation)
		require.Nil(t, err, fmt.Sprintf("create invitation unexpected error: %s", err))
		invitation.Token = ""
		items = append(items, invitation)
	}

	cases := []struct {
		desc      string
		page      invitations.Page
		withToken bool
		response  invitations.InvitationPage
		err       error
	}{
		{
			desc: "retrieve invitations successfully",
			page: invitations.Page{
				Offset: 0,
				Limit:  10,
			},
			response: invitations.InvitationPage{
				Total:       uint64(num),
				Offset:      0,
				Limit:       10,
				Invitations: items[:10],
			},
			err: nil,
		},
		{
			desc: "retrieve invitations with token",
			page: invitations.Page{
				Offset: 0,
				Limit:  10,
			},
			withToken: true,
			response: invitations.InvitationPage{
				Total:       uint64(num),
				Offset:      0,
				Limit:       10,
				Invitations: addToken(items[:10]),
			},
			err: nil,
		},
		{
			desc: "retrieve invitations with offset",
			page: invitations.Page{
				Offset: 10,
				Limit:  10,
			},
			response: invitations.InvitationPage{
				Total:       uint64(num),
				Offset:      10,
				Limit:       10,
				Invitations: items[10:20],
			},
		},
		{
			desc: "retrieve invitations with limit",
			page: invitations.Page{
				Offset: 0,
				Limit:  50,
			},
			response: invitations.InvitationPage{
				Total:       uint64(num),
				Offset:      0,
				Limit:       50,
				Invitations: items[:50],
			},
		},
		{
			desc: "retrieve invitations with offset and limit",
			page: invitations.Page{
				Offset: 10,
				Limit:  50,
			},
			response: invitations.InvitationPage{
				Total:       uint64(num),
				Offset:      10,
				Limit:       50,
				Invitations: items[10:60],
			},
		},
		{
			desc: "retrieve invitations with offset out of range",
			page: invitations.Page{
				Offset: 1000,
				Limit:  50,
			},
			response: invitations.InvitationPage{
				Total:       uint64(num),
				Offset:      1000,
				Limit:       50,
				Invitations: []invitations.Invitation(nil),
			},
		},
		{
			desc: "retrieve invitations with offset and limit out of range",
			page: invitations.Page{
				Offset: 170,
				Limit:  50,
			},
			response: invitations.InvitationPage{
				Total:       uint64(num),
				Offset:      170,
				Limit:       50,
				Invitations: items[170:200],
			},
		},
		{
			desc: "retrieve invitations with limit out of range",
			page: invitations.Page{
				Offset: 0,
				Limit:  1000,
			},
			response: invitations.InvitationPage{
				Total:       uint64(num),
				Offset:      0,
				Limit:       1000,
				Invitations: items,
			},
		},
		{
			desc: "retrieve invitations with empty page",
			page: invitations.Page{},
			response: invitations.InvitationPage{
				Total:       uint64(num),
				Offset:      0,
				Limit:       0,
				Invitations: []invitations.Invitation(nil),
			},
		},
		{
			desc: "retrieve invitations with domain",
			page: invitations.Page{
				Domain: items[0].Domain,
				Offset: 0,
				Limit:  10,
			},
			response: invitations.InvitationPage{
				Total:       1,
				Offset:      0,
				Limit:       10,
				Invitations: []invitations.Invitation{items[0]},
			},
		},
		{
			desc: "retrieve invitations with user id",
			page: invitations.Page{
				UserID: items[0].UserID,
				Offset: 0,
				Limit:  10,
			},
			response: invitations.InvitationPage{
				Total:       1,
				Offset:      0,
				Limit:       10,
				Invitations: []invitations.Invitation{items[0]},
			},
		},
		{
			desc: "retrieve invitations with invited_by",
			page: invitations.Page{
				InvitedBy: items[0].InvitedBy,
				Offset:    0,
				Limit:     10,
			},
			response: invitations.InvitationPage{
				Total:       1,
				Offset:      0,
				Limit:       10,
				Invitations: []invitations.Invitation{items[0]},
			},
		},
		{
			desc: "retrieve invitations with invited_by_or_user_id",
			page: invitations.Page{
				InvitedByOrUserID: items[0].UserID,
				Offset:            0,
				Limit:             10,
			},
			response: invitations.InvitationPage{
				Total:       1,
				Offset:      0,
				Limit:       10,
				Invitations: []invitations.Invitation{items[0]},
			},
		},
		{
			desc: "retrieve invitations with relation",
			page: invitations.Page{
				Relation: relation + "-0",
				Offset:   0,
				Limit:    10,
			},
			response: invitations.InvitationPage{
				Total:       1,
				Offset:      0,
				Limit:       10,
				Invitations: []invitations.Invitation{items[0]},
			},
		},
		{
			desc: "retrieve invitations with domain and user id",
			page: invitations.Page{
				Domain: items[0].Domain,
				UserID: items[0].UserID,
				Offset: 0,
				Limit:  10,
			},
			response: invitations.InvitationPage{
				Total:       1,
				Offset:      0,
				Limit:       10,
				Invitations: []invitations.Invitation{items[0]},
			},
		},
		{
			desc: "retrieve invitations with domain and invited_by",
			page: invitations.Page{
				Domain:    items[0].Domain,
				InvitedBy: items[0].InvitedBy,
				Offset:    0,
				Limit:     10,
			},
			response: invitations.InvitationPage{
				Total:       1,
				Offset:      0,
				Limit:       10,
				Invitations: []invitations.Invitation{items[0]},
			},
		},
		{
			desc: "retrieve invitations with user id and invited_by",
			page: invitations.Page{
				UserID:    items[0].UserID,
				InvitedBy: items[0].InvitedBy,
				Offset:    0,
				Limit:     10,
			},
			response: invitations.InvitationPage{
				Total:       1,
				Offset:      0,
				Limit:       10,
				Invitations: []invitations.Invitation{items[0]},
			},
		},
		{
			desc: "retrieve invitations with domain, user id and invited_by",
			page: invitations.Page{
				Domain:    items[0].Domain,
				UserID:    items[0].UserID,
				InvitedBy: items[0].InvitedBy,
				Offset:    0,
				Limit:     10,
			},
			response: invitations.InvitationPage{
				Total:       1,
				Offset:      0,
				Limit:       10,
				Invitations: []invitations.Invitation{items[0]},
			},
		},
		{
			desc: "retrieve invitations with domain, user id, invited_by and relation",
			page: invitations.Page{
				Domain:    items[0].Domain,
				UserID:    items[0].UserID,
				InvitedBy: items[0].InvitedBy,
				Relation:  relation + "-0",
				Offset:    0,
				Limit:     10,
			},
			response: invitations.InvitationPage{
				Total:       1,
				Offset:      0,
				Limit:       10,
				Invitations: []invitations.Invitation{items[0]},
			},
		},
		{
			desc: "retrieve invitations with invalid domain",
			page: invitations.Page{
				Domain: invalidUUID,
				Offset: 0,
				Limit:  10,
			},
			response: invitations.InvitationPage{
				Total:       0,
				Offset:      0,
				Limit:       10,
				Invitations: []invitations.Invitation(nil),
			},
		},
		{
			desc: "retrieve invitations with invalid user id",
			page: invitations.Page{
				UserID: generateUUID(t),
				Offset: 0,
				Limit:  10,
			},
			response: invitations.InvitationPage{
				Total:       0,
				Offset:      0,
				Limit:       10,
				Invitations: []invitations.Invitation(nil),
			},
		},
		{
			desc: "retrieve invitations with invalid invited_by",
			page: invitations.Page{
				InvitedBy: invalidUUID,
				Offset:    0,
				Limit:     10,
			},
			response: invitations.InvitationPage{
				Total:       0,
				Offset:      0,
				Limit:       10,
				Invitations: []invitations.Invitation(nil),
			},
		},
		{
			desc: "retrieve invitations with invalid relation",
			page: invitations.Page{
				Relation: invalidUUID,
				Offset:   0,
				Limit:    10,
			},
			response: invitations.InvitationPage{
				Total:       0,
				Offset:      0,
				Limit:       10,
				Invitations: []invitations.Invitation(nil),
			},
		},
	}
	for _, tc := range cases {
		page, err := repo.RetrieveAll(context.Background(), tc.withToken, tc.page)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("desc: %s\n", tc.desc))
	}
}

func TestInvitationUpdateToken(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM invitations")
		require.Nil(t, err, fmt.Sprintf("clean invitations unexpected error: %s", err))
	})
	repo := postgres.NewRepository(database)

	invitation := invitations.Invitation{
		InvitedBy: generateUUID(t),
		UserID:    generateUUID(t),
		Domain:    generateUUID(t),
		Token:     validToken,
		CreatedAt: time.Now(),
	}
	err := repo.Create(context.Background(), invitation)
	require.Nil(t, err, fmt.Sprintf("create invitation unexpected error: %s", err))

	cases := []struct {
		desc       string
		invitation invitations.Invitation
		err        error
	}{
		{
			desc: "update invitation successfully",
			invitation: invitations.Invitation{
				Domain:    invitation.Domain,
				UserID:    invitation.UserID,
				Token:     validToken,
				UpdatedAt: time.Now(),
			},
			err: nil,
		},
		{
			desc: "update invitation with invalid user id",
			invitation: invitations.Invitation{
				UserID:    generateUUID(t),
				Domain:    invitation.Domain,
				Token:     validToken,
				UpdatedAt: time.Now(),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "update invitation with invalid domain",
			invitation: invitations.Invitation{
				UserID:    invitation.UserID,
				Domain:    generateUUID(t),
				Token:     validToken,
				UpdatedAt: time.Now(),
			},
			err: repoerr.ErrNotFound,
		},
	}
	for _, tc := range cases {
		err := repo.UpdateToken(context.Background(), tc.invitation)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestInvitationUpdateConfirmation(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM invitations")
		require.Nil(t, err, fmt.Sprintf("clean invitations unexpected error: %s", err))
	})
	repo := postgres.NewRepository(database)

	invitation := invitations.Invitation{
		InvitedBy: generateUUID(t),
		UserID:    generateUUID(t),
		Domain:    generateUUID(t),
		Token:     validToken,
		CreatedAt: time.Now(),
	}
	err := repo.Create(context.Background(), invitation)
	require.Nil(t, err, fmt.Sprintf("create invitation unexpected error: %s", err))

	cases := []struct {
		desc       string
		invitation invitations.Invitation
		err        error
	}{
		{
			desc: "update invitation successfully",
			invitation: invitations.Invitation{
				Domain:      invitation.Domain,
				UserID:      invitation.UserID,
				ConfirmedAt: time.Now(),
			},
			err: nil,
		},
		{
			desc: "update invitation with invalid user id",
			invitation: invitations.Invitation{
				UserID:      generateUUID(t),
				Domain:      invitation.UserID,
				ConfirmedAt: time.Now(),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "update invitation with invalid domain",
			invitation: invitations.Invitation{
				UserID:      invitation.UserID,
				Domain:      generateUUID(t),
				ConfirmedAt: time.Now(),
			},
			err: repoerr.ErrNotFound,
		},
	}
	for _, tc := range cases {
		err := repo.UpdateConfirmation(context.Background(), tc.invitation)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestInvitationDelete(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM invitations")
		require.Nil(t, err, fmt.Sprintf("clean invitations unexpected error: %s", err))
	})
	repo := postgres.NewRepository(database)

	invitation := invitations.Invitation{
		InvitedBy: generateUUID(t),
		UserID:    generateUUID(t),
		Domain:    generateUUID(t),
		Token:     validToken,
		CreatedAt: time.Now(),
	}
	err := repo.Create(context.Background(), invitation)
	require.Nil(t, err, fmt.Sprintf("create invitation unexpected error: %s", err))

	cases := []struct {
		desc       string
		invitation invitations.Invitation
		err        error
	}{
		{
			desc: "delete invitation successfully",
			invitation: invitations.Invitation{
				UserID: invitation.UserID,
				Domain: invitation.Domain,
			},
			err: nil,
		},
		{
			desc: "delete invitation with invalid invitation id",
			invitation: invitations.Invitation{
				UserID: generateUUID(t),
				Domain: generateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:       "delete invitation with empty invitation id",
			invitation: invitations.Invitation{},
			err:        repoerr.ErrNotFound,
		},
	}
	for _, tc := range cases {
		err := repo.Delete(context.Background(), tc.invitation.UserID, tc.invitation.Domain)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func generateUUID(t *testing.T) string {
	idProvider := uuid.New()
	ulid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	return ulid
}

func addToken(invs []invitations.Invitation) []invitations.Invitation {
	invscopy := make([]invitations.Invitation, len(invs))
	copy(invscopy, invs)
	for i := range invscopy {
		invscopy[i].Token = validToken
	}
	return invscopy
}
