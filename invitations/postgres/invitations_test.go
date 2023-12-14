// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/invitations"
	"github.com/absmach/magistrala/invitations/postgres"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
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

	domainID := testsutil.GenerateUUID(t)
	userID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc       string
		invitation invitations.Invitation
		err        error
	}{
		{
			desc: "add new invitation successfully",
			invitation: invitations.Invitation{
				InvitedBy: testsutil.GenerateUUID(t),
				UserID:    userID,
				DomainID:  domainID,
				Token:     validToken,
				Relation:  relation,
				CreatedAt: time.Now(),
			},
			err: nil,
		},
		{
			desc: "add new invitation with an confirmed_at date",
			invitation: invitations.Invitation{
				InvitedBy:   testsutil.GenerateUUID(t),
				UserID:      testsutil.GenerateUUID(t),
				DomainID:    testsutil.GenerateUUID(t),
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
				InvitedBy: testsutil.GenerateUUID(t),
				UserID:    userID,
				DomainID:  domainID,
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
				UserID:    testsutil.GenerateUUID(t),
				DomainID:  testsutil.GenerateUUID(t),
				Token:     validToken,
				Relation:  relation,
				CreatedAt: time.Now(),
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "add invitation with invalid invitation relation",
			invitation: invitations.Invitation{
				InvitedBy: testsutil.GenerateUUID(t),
				UserID:    testsutil.GenerateUUID(t),
				DomainID:  testsutil.GenerateUUID(t),
				Token:     validToken,
				Relation:  strings.Repeat("a", 255),
				CreatedAt: time.Now(),
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "add invitation with invalid invitation domain",
			invitation: invitations.Invitation{
				InvitedBy: testsutil.GenerateUUID(t),
				UserID:    testsutil.GenerateUUID(t),
				DomainID:  invalidUUID,
				Token:     validToken,
				Relation:  relation,
				CreatedAt: time.Now(),
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "add invitation with invalid invitation user id",
			invitation: invitations.Invitation{
				InvitedBy: testsutil.GenerateUUID(t),
				UserID:    invalidUUID,
				DomainID:  testsutil.GenerateUUID(t),
				Token:     validToken,
				Relation:  relation,
				CreatedAt: time.Now(),
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "add invitation with empty invitation domain",
			invitation: invitations.Invitation{
				InvitedBy: testsutil.GenerateUUID(t),
				UserID:    testsutil.GenerateUUID(t),
				Token:     validToken,
				Relation:  relation,
				CreatedAt: time.Now(),
			},
			err: nil,
		},
		{
			desc: "add invitation with empty invitation user id",
			invitation: invitations.Invitation{
				InvitedBy: testsutil.GenerateUUID(t),
				DomainID:  testsutil.GenerateUUID(t),
				Token:     validToken,
				Relation:  relation,
				CreatedAt: time.Now(),
			},
			err: nil,
		},
		{
			desc: "add invitation with empty invitation invited_by",
			invitation: invitations.Invitation{
				DomainID:  testsutil.GenerateUUID(t),
				UserID:    testsutil.GenerateUUID(t),
				Token:     validToken,
				Relation:  relation,
				CreatedAt: time.Now(),
			},
			err: nil,
		},
		{
			desc: "add invitation with empty invitation token",
			invitation: invitations.Invitation{
				InvitedBy: testsutil.GenerateUUID(t),
				DomainID:  testsutil.GenerateUUID(t),
				UserID:    testsutil.GenerateUUID(t),
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
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
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
		InvitedBy: testsutil.GenerateUUID(t),
		UserID:    testsutil.GenerateUUID(t),
		DomainID:  testsutil.GenerateUUID(t),
		Token:     validToken,
		Relation:  relation,
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	err := repo.Create(context.Background(), invitation)
	require.Nil(t, err, fmt.Sprintf("create invitation unexpected error: %s", err))

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
			domainID: invitation.DomainID,
			response: invitation,
			err:      nil,
		},
		{
			desc:     "retrieve invitations with invalid invitation user id",
			userID:   testsutil.GenerateUUID(t),
			domainID: invitation.DomainID,
			response: invitations.Invitation{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve invitations with invalid invitation domain_id",
			userID:   invitation.UserID,
			domainID: testsutil.GenerateUUID(t),
			response: invitations.Invitation{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve invitations with invalid invitation user id and domain_id",
			userID:   testsutil.GenerateUUID(t),
			domainID: testsutil.GenerateUUID(t),
			response: invitations.Invitation{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve invitations with empty invitation user id",
			userID:   "",
			domainID: invitation.DomainID,
			response: invitations.Invitation{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve invitations with empty invitation domain_id",
			userID:   invitation.UserID,
			domainID: "",
			response: invitations.Invitation{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve invitations with empty invitation user id and domain_id",
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
			InvitedBy: testsutil.GenerateUUID(t),
			UserID:    testsutil.GenerateUUID(t),
			DomainID:  testsutil.GenerateUUID(t),
			Token:     validToken,
			Relation:  fmt.Sprintf("%s-%d", relation, i),
			CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		}
		err := repo.Create(context.Background(), invitation)
		require.Nil(t, err, fmt.Sprintf("create invitation unexpected error: %s", err))
		invitation.Token = ""
		items = append(items, invitation)
	}
	items[100].ConfirmedAt = time.Now().UTC().Truncate(time.Microsecond)
	err := repo.UpdateConfirmation(context.Background(), items[100])
	require.Nil(t, err, fmt.Sprintf("update invitation unexpected error: %s", err))

	swap := items[100]
	items = append(items[:100], items[101:]...)
	items = append(items, swap)

	cases := []struct {
		desc     string
		page     invitations.Page
		response invitations.InvitationPage
		err      error
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
				DomainID: items[0].DomainID,
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
			desc: "retrieve invitations with domain_id and user id",
			page: invitations.Page{
				DomainID: items[0].DomainID,
				UserID:   items[0].UserID,
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
			desc: "retrieve invitations with domain_id and invited_by",
			page: invitations.Page{
				DomainID:  items[0].DomainID,
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
			desc: "retrieve invitations with domain_id, user id and invited_by",
			page: invitations.Page{
				DomainID:  items[0].DomainID,
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
			desc: "retrieve invitations with domain_id, user id, invited_by and relation",
			page: invitations.Page{
				DomainID:  items[0].DomainID,
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
				DomainID: invalidUUID,
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
		{
			desc: "retrieve invitations with invalid user id",
			page: invitations.Page{
				UserID: testsutil.GenerateUUID(t),
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
		{
			desc: "retrieve invitations with accepted state",
			page: invitations.Page{
				State:  invitations.Accepted,
				Offset: 0,
				Limit:  10,
			},
			response: invitations.InvitationPage{
				Total:       1,
				Offset:      0,
				Limit:       10,
				Invitations: []invitations.Invitation{items[num-1]},
			},
		},
		{
			desc: "retrieve invitations with pending state",
			page: invitations.Page{
				State:  invitations.Pending,
				Offset: 0,
				Limit:  10,
			},
			response: invitations.InvitationPage{
				Total:       uint64(num - 1),
				Offset:      0,
				Limit:       10,
				Invitations: items[0:10],
			},
		},
	}
	for _, tc := range cases {
		page, err := repo.RetrieveAll(context.Background(), tc.page)
		assert.Equal(t, tc.response.Total, page.Total, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Total, page.Total))
		assert.Equal(t, tc.response.Offset, page.Offset, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Offset, page.Offset))
		assert.Equal(t, tc.response.Limit, page.Limit, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Limit, page.Limit))
		assert.ElementsMatch(t, page.Invitations, tc.response.Invitations, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response.Invitations, page.Invitations))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestInvitationUpdateToken(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM invitations")
		require.Nil(t, err, fmt.Sprintf("clean invitations unexpected error: %s", err))
	})
	repo := postgres.NewRepository(database)

	invitation := invitations.Invitation{
		InvitedBy: testsutil.GenerateUUID(t),
		UserID:    testsutil.GenerateUUID(t),
		DomainID:  testsutil.GenerateUUID(t),
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
				DomainID:  invitation.DomainID,
				UserID:    invitation.UserID,
				Token:     validToken,
				UpdatedAt: time.Now(),
			},
			err: nil,
		},
		{
			desc: "update invitation with invalid user id",
			invitation: invitations.Invitation{
				UserID:    testsutil.GenerateUUID(t),
				DomainID:  invitation.DomainID,
				Token:     validToken,
				UpdatedAt: time.Now(),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "update invitation with invalid domain_id",
			invitation: invitations.Invitation{
				UserID:    invitation.UserID,
				DomainID:  testsutil.GenerateUUID(t),
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
		InvitedBy: testsutil.GenerateUUID(t),
		UserID:    testsutil.GenerateUUID(t),
		DomainID:  testsutil.GenerateUUID(t),
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
				DomainID:    invitation.DomainID,
				UserID:      invitation.UserID,
				ConfirmedAt: time.Now(),
			},
			err: nil,
		},
		{
			desc: "update invitation with invalid user id",
			invitation: invitations.Invitation{
				UserID:      testsutil.GenerateUUID(t),
				DomainID:    invitation.UserID,
				ConfirmedAt: time.Now(),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "update invitation with invalid domain",
			invitation: invitations.Invitation{
				UserID:      invitation.UserID,
				DomainID:    testsutil.GenerateUUID(t),
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
		InvitedBy: testsutil.GenerateUUID(t),
		UserID:    testsutil.GenerateUUID(t),
		DomainID:  testsutil.GenerateUUID(t),
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
				UserID:   invitation.UserID,
				DomainID: invitation.DomainID,
			},
			err: nil,
		},
		{
			desc: "delete invitation with invalid invitation id",
			invitation: invitations.Invitation{
				UserID:   testsutil.GenerateUUID(t),
				DomainID: testsutil.GenerateUUID(t),
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
		err := repo.Delete(context.Background(), tc.invitation.UserID, tc.invitation.DomainID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
