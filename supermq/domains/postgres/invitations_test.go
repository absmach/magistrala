// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/absmach/supermq/domains"
	"github.com/absmach/supermq/domains/postgres"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const roleName = "roleName"

var invalidUUID = strings.Repeat("a", 37)

func TestSaveInvitation(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM invitations")
		require.Nil(t, err, fmt.Sprintf("clean invitations unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM domains")
		require.Nil(t, err, fmt.Sprintf("clean domains unexpected error: %s", err))
	})
	repo := postgres.NewRepository(database)

	dom := saveDomain(t, repo)
	userID := testsutil.GenerateUUID(t)
	roleID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc       string
		invitation domains.Invitation
		err        error
	}{
		{
			desc: "add new invitation successfully",
			invitation: domains.Invitation{
				InvitedBy:     testsutil.GenerateUUID(t),
				InviteeUserID: userID,
				DomainID:      dom.ID,
				RoleID:        roleID,
				CreatedAt:     time.Now(),
			},
			err: nil,
		},
		{
			desc: "add new invitation with an confirmed_at date",
			invitation: domains.Invitation{
				InvitedBy:     testsutil.GenerateUUID(t),
				InviteeUserID: testsutil.GenerateUUID(t),
				DomainID:      dom.ID,
				CreatedAt:     time.Now(),
				RoleID:        roleID,
				ConfirmedAt:   time.Now(),
			},
			err: nil,
		},
		{
			desc: "add invitation with duplicate invitation",
			invitation: domains.Invitation{
				InvitedBy:     testsutil.GenerateUUID(t),
				InviteeUserID: userID,
				DomainID:      dom.ID,
				RoleID:        roleID,
				CreatedAt:     time.Now(),
			},
			err: repoerr.ErrConflict,
		},
		{
			desc: "add invitation with invalid invitation invited_by",
			invitation: domains.Invitation{
				InvitedBy:     invalidUUID,
				InviteeUserID: testsutil.GenerateUUID(t),
				DomainID:      dom.ID,
				RoleID:        roleID,
				CreatedAt:     time.Now(),
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "add invitation with invalid invitation domain",
			invitation: domains.Invitation{
				InvitedBy:     testsutil.GenerateUUID(t),
				InviteeUserID: testsutil.GenerateUUID(t),
				DomainID:      invalidUUID,
				RoleID:        roleID,
				CreatedAt:     time.Now(),
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "add invitation with invalid invitation invitee user id",
			invitation: domains.Invitation{
				InvitedBy:     testsutil.GenerateUUID(t),
				InviteeUserID: invalidUUID,
				DomainID:      testsutil.GenerateUUID(t),
				RoleID:        roleID,
				CreatedAt:     time.Now(),
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "add invitation with empty invitation domain",
			invitation: domains.Invitation{
				InvitedBy:     testsutil.GenerateUUID(t),
				InviteeUserID: testsutil.GenerateUUID(t),
				RoleID:        roleID,
				CreatedAt:     time.Now(),
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "add invitation with empty invitation invitee user id",
			invitation: domains.Invitation{
				InvitedBy: testsutil.GenerateUUID(t),
				DomainID:  dom.ID,
				RoleID:    roleID,
				CreatedAt: time.Now(),
			},
			err: nil,
		},
		{
			desc: "add invitation with empty invitation invited_by",
			invitation: domains.Invitation{
				DomainID:      dom.ID,
				InviteeUserID: testsutil.GenerateUUID(t),
				RoleID:        roleID,
				CreatedAt:     time.Now(),
			},
			err: nil,
		},
		{
			desc: "add invitation with empty invitation role id",
			invitation: domains.Invitation{
				InvitedBy:     testsutil.GenerateUUID(t),
				InviteeUserID: testsutil.GenerateUUID(t),
				DomainID:      dom.ID,
				CreatedAt:     time.Now(),
			},
			err: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.SaveInvitation(context.Background(), tc.invitation)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
		})
	}
}

func TestInvitationRetrieve(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM invitations")
		require.Nil(t, err, fmt.Sprintf("clean invitations unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM domains")
		require.Nil(t, err, fmt.Sprintf("clean domains unexpected error: %s", err))
	})
	repo := postgres.NewRepository(database)

	dom := saveDomain(t, repo)

	invitation := domains.Invitation{
		InvitedBy:     testsutil.GenerateUUID(t),
		InviteeUserID: testsutil.GenerateUUID(t),
		DomainID:      dom.ID,
		RoleID:        testsutil.GenerateUUID(t),
		CreatedAt:     time.Now().UTC().Truncate(time.Microsecond),
	}

	err := repo.SaveInvitation(context.Background(), invitation)
	require.Nil(t, err, fmt.Sprintf("create invitation unexpected error: %s", err))

	cases := []struct {
		desc     string
		userID   string
		domainID string
		response domains.Invitation
		err      error
	}{
		{
			desc:     "retrieve invitations successfully",
			userID:   invitation.InviteeUserID,
			domainID: invitation.DomainID,
			response: invitation,
			err:      nil,
		},
		{
			desc:     "retrieve invitations with invalid invitee user id",
			userID:   testsutil.GenerateUUID(t),
			domainID: invitation.DomainID,
			response: domains.Invitation{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve invitations with invalid invitation domain_id",
			userID:   invitation.InviteeUserID,
			domainID: testsutil.GenerateUUID(t),
			response: domains.Invitation{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve invitations with invalid invitee user id and domain_id",
			userID:   testsutil.GenerateUUID(t),
			domainID: testsutil.GenerateUUID(t),
			response: domains.Invitation{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve invitations with empty invitee user id",
			userID:   "",
			domainID: invitation.DomainID,
			response: domains.Invitation{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve invitations with empty invitation domain_id",
			userID:   invitation.InviteeUserID,
			domainID: "",
			response: domains.Invitation{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve invitations with empty invitation user id and domain_id",
			userID:   "",
			domainID: "",
			response: domains.Invitation{},
			err:      repoerr.ErrNotFound,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			inv, err := repo.RetrieveInvitation(context.Background(), tc.userID, tc.domainID)
			assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.response, inv, fmt.Sprintf("desc: %s\n", tc.desc))
		})
	}
}

func TestInvitationRetrieveAll(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM invitations")
		require.Nil(t, err, fmt.Sprintf("clean invitations unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM domains")
		require.Nil(t, err, fmt.Sprintf("clean domains unexpected error: %s", err))
	})
	repo := postgres.NewRepository(database)

	dom := saveDomain(t, repo)

	num := 200

	var items []domains.Invitation
	for i := 0; i < num; i++ {
		invitation := domains.Invitation{
			InvitedBy:     testsutil.GenerateUUID(t),
			InviteeUserID: testsutil.GenerateUUID(t),
			DomainID:      dom.ID,
			DomainName:    dom.Name,
			RoleID:        testsutil.GenerateUUID(t),
			CreatedAt:     time.Now().UTC().Truncate(time.Microsecond),
		}
		err := repo.SaveInvitation(context.Background(), invitation)
		require.Nil(t, err, fmt.Sprintf("create invitation unexpected error: %s", err))
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
		page     domains.InvitationPageMeta
		response domains.InvitationPage
		err      error
	}{
		{
			desc: "retrieve invitations successfully",
			page: domains.InvitationPageMeta{
				Offset: 0,
				Limit:  10,
			},
			response: domains.InvitationPage{
				Total:       uint64(num),
				Offset:      0,
				Limit:       10,
				Invitations: items[:10],
			},
			err: nil,
		},
		{
			desc: "retrieve invitations with offset",
			page: domains.InvitationPageMeta{
				Offset: 10,
				Limit:  10,
			},
			response: domains.InvitationPage{
				Total:       uint64(num),
				Offset:      10,
				Limit:       10,
				Invitations: items[10:20],
			},
		},
		{
			desc: "retrieve invitations with limit",
			page: domains.InvitationPageMeta{
				Offset: 0,
				Limit:  50,
			},
			response: domains.InvitationPage{
				Total:       uint64(num),
				Offset:      0,
				Limit:       50,
				Invitations: items[:50],
			},
		},
		{
			desc: "retrieve invitations with offset and limit",
			page: domains.InvitationPageMeta{
				Offset: 10,
				Limit:  50,
			},
			response: domains.InvitationPage{
				Total:       uint64(num),
				Offset:      10,
				Limit:       50,
				Invitations: items[10:60],
			},
		},
		{
			desc: "retrieve invitations with offset out of range",
			page: domains.InvitationPageMeta{
				Offset: 1000,
				Limit:  50,
			},
			response: domains.InvitationPage{
				Total:       uint64(num),
				Offset:      1000,
				Limit:       50,
				Invitations: []domains.Invitation(nil),
			},
		},
		{
			desc: "retrieve invitations with offset and limit out of range",
			page: domains.InvitationPageMeta{
				Offset: 170,
				Limit:  50,
			},
			response: domains.InvitationPage{
				Total:       uint64(num),
				Offset:      170,
				Limit:       50,
				Invitations: items[170:200],
			},
		},
		{
			desc: "retrieve invitations with limit out of range",
			page: domains.InvitationPageMeta{
				Offset: 0,
				Limit:  1000,
			},
			response: domains.InvitationPage{
				Total:       uint64(num),
				Offset:      0,
				Limit:       1000,
				Invitations: items,
			},
		},
		{
			desc: "retrieve invitations with empty page",
			page: domains.InvitationPageMeta{},
			response: domains.InvitationPage{
				Total:       uint64(num),
				Offset:      0,
				Limit:       0,
				Invitations: []domains.Invitation(nil),
			},
		},
		{
			desc: "retrieve invitations with domain",
			page: domains.InvitationPageMeta{
				DomainID: items[0].DomainID,
				Offset:   0,
				Limit:    10,
			},
			response: domains.InvitationPage{
				Total:       uint64(num),
				Offset:      0,
				Limit:       10,
				Invitations: items[:10],
			},
		},
		{
			desc: "retrieve invitations with invitee user id",
			page: domains.InvitationPageMeta{
				InviteeUserID: items[0].InviteeUserID,
				Offset:        0,
				Limit:         10,
			},
			response: domains.InvitationPage{
				Total:       1,
				Offset:      0,
				Limit:       10,
				Invitations: []domains.Invitation{items[0]},
			},
		},
		{
			desc: "retrieve invitations with invited_by",
			page: domains.InvitationPageMeta{
				InvitedBy: items[0].InvitedBy,
				Offset:    0,
				Limit:     10,
			},
			response: domains.InvitationPage{
				Total:       1,
				Offset:      0,
				Limit:       10,
				Invitations: []domains.Invitation{items[0]},
			},
		},
		{
			desc: "retrieve invitations with role_id",
			page: domains.InvitationPageMeta{
				RoleID: items[3].RoleID,
				Offset: 0,
				Limit:  10,
			},
			response: domains.InvitationPage{
				Total:       1,
				Offset:      0,
				Limit:       10,
				Invitations: []domains.Invitation{items[3]},
			},
		},
		{
			desc: "retrieve invitations with invited_by_or_user_id",
			page: domains.InvitationPageMeta{
				InvitedByOrUserID: items[0].InviteeUserID,
				Offset:            0,
				Limit:             10,
			},
			response: domains.InvitationPage{
				Total:       1,
				Offset:      0,
				Limit:       10,
				Invitations: []domains.Invitation{items[0]},
			},
		},
		{
			desc: "retrieve invitations with domain_id and invitee user id",
			page: domains.InvitationPageMeta{
				DomainID:      items[0].DomainID,
				InviteeUserID: items[0].InviteeUserID,
				Offset:        0,
				Limit:         10,
			},
			response: domains.InvitationPage{
				Total:       1,
				Offset:      0,
				Limit:       10,
				Invitations: []domains.Invitation{items[0]},
			},
		},
		{
			desc: "retrieve invitations with domain_id and invited_by",
			page: domains.InvitationPageMeta{
				DomainID:  items[0].DomainID,
				InvitedBy: items[0].InvitedBy,
				Offset:    0,
				Limit:     10,
			},
			response: domains.InvitationPage{
				Total:       1,
				Offset:      0,
				Limit:       10,
				Invitations: []domains.Invitation{items[0]},
			},
		},
		{
			desc: "retrieve invitations with invitee user id and invited_by",
			page: domains.InvitationPageMeta{
				InviteeUserID: items[0].InviteeUserID,
				InvitedBy:     items[0].InvitedBy,
				Offset:        0,
				Limit:         10,
			},
			response: domains.InvitationPage{
				Total:       1,
				Offset:      0,
				Limit:       10,
				Invitations: []domains.Invitation{items[0]},
			},
		},
		{
			desc: "retrieve invitations with domain_id, invitee user id and invited_by",
			page: domains.InvitationPageMeta{
				DomainID:      items[0].DomainID,
				InviteeUserID: items[0].InviteeUserID,
				InvitedBy:     items[0].InvitedBy,
				Offset:        0,
				Limit:         10,
			},
			response: domains.InvitationPage{
				Total:       1,
				Offset:      0,
				Limit:       10,
				Invitations: []domains.Invitation{items[0]},
			},
		},
		{
			desc: "retrieve invitations with domain_id, invitee user id, invited_by and role_id",
			page: domains.InvitationPageMeta{
				DomainID:      items[0].DomainID,
				InviteeUserID: items[0].InviteeUserID,
				InvitedBy:     items[0].InvitedBy,
				RoleID:        items[0].RoleID,
				Offset:        0,
				Limit:         10,
			},
			response: domains.InvitationPage{
				Total:       1,
				Offset:      0,
				Limit:       10,
				Invitations: []domains.Invitation{items[0]},
			},
		},
		{
			desc: "retrieve invitations with invalid domain",
			page: domains.InvitationPageMeta{
				DomainID: invalidUUID,
				Offset:   0,
				Limit:    10,
			},
			response: domains.InvitationPage{
				Total:       0,
				Offset:      0,
				Limit:       10,
				Invitations: []domains.Invitation(nil),
			},
		},
		{
			desc: "retrieve invitations with invalid invitee user id",
			page: domains.InvitationPageMeta{
				InviteeUserID: testsutil.GenerateUUID(t),
				Offset:        0,
				Limit:         10,
			},
			response: domains.InvitationPage{
				Total:       0,
				Offset:      0,
				Limit:       10,
				Invitations: []domains.Invitation(nil),
			},
		},
		{
			desc: "retrieve invitations with invalid invited_by",
			page: domains.InvitationPageMeta{
				InvitedBy: invalidUUID,
				Offset:    0,
				Limit:     10,
			},
			response: domains.InvitationPage{
				Total:       0,
				Offset:      0,
				Limit:       10,
				Invitations: []domains.Invitation(nil),
			},
		},
		{
			desc: "retrieve invitations with invalid role_id",
			page: domains.InvitationPageMeta{
				RoleID: invalidUUID,
				Offset: 0,
				Limit:  10,
			},
			response: domains.InvitationPage{
				Total:       0,
				Offset:      0,
				Limit:       10,
				Invitations: []domains.Invitation(nil),
			},
		},
		{
			desc: "retrieve invitations with accepted state",
			page: domains.InvitationPageMeta{
				State:  domains.Accepted,
				Offset: 0,
				Limit:  10,
			},
			response: domains.InvitationPage{
				Total:       1,
				Offset:      0,
				Limit:       10,
				Invitations: []domains.Invitation{items[num-1]},
			},
		},
		{
			desc: "retrieve invitations with pending state",
			page: domains.InvitationPageMeta{
				State:  domains.Pending,
				Offset: 0,
				Limit:  10,
			},
			response: domains.InvitationPage{
				Total:       uint64(num - 1),
				Offset:      0,
				Limit:       10,
				Invitations: items[0:10],
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			page, err := repo.RetrieveAllInvitations(context.Background(), tc.page)
			assert.Equal(t, tc.response.Total, page.Total, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Total, page.Total))
			assert.Equal(t, tc.response.Offset, page.Offset, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Offset, page.Offset))
			assert.Equal(t, tc.response.Limit, page.Limit, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Limit, page.Limit))
			assert.ElementsMatch(t, page.Invitations, tc.response.Invitations, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response.Invitations, page.Invitations))
			assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestInvitationUpdateConfirmation(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM invitations")
		require.Nil(t, err, fmt.Sprintf("clean invitations unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM domains")
		require.Nil(t, err, fmt.Sprintf("clean domains unexpected error: %s", err))
	})
	repo := postgres.NewRepository(database)

	dom := saveDomain(t, repo)

	invitation := domains.Invitation{
		InvitedBy:     testsutil.GenerateUUID(t),
		InviteeUserID: testsutil.GenerateUUID(t),
		DomainID:      dom.ID,
		DomainName:    dom.Name,
		RoleID:        testsutil.GenerateUUID(t),
		RoleName:      roleName,
		CreatedAt:     time.Now(),
	}
	err := repo.SaveInvitation(context.Background(), invitation)
	require.Nil(t, err, fmt.Sprintf("create invitation unexpected error: %s", err))

	cases := []struct {
		desc       string
		invitation domains.Invitation
		err        error
	}{
		{
			desc: "update invitation successfully",
			invitation: domains.Invitation{
				DomainID:      invitation.DomainID,
				InviteeUserID: invitation.InviteeUserID,
				ConfirmedAt:   time.Now(),
			},
			err: nil,
		},
		{
			desc: "update invitation with invalid invitee user id",
			invitation: domains.Invitation{
				InviteeUserID: testsutil.GenerateUUID(t),
				DomainID:      invitation.InviteeUserID,
				ConfirmedAt:   time.Now(),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "update invitation with invalid domain",
			invitation: domains.Invitation{
				InviteeUserID: invitation.InviteeUserID,
				DomainID:      testsutil.GenerateUUID(t),
				ConfirmedAt:   time.Now(),
			},
			err: repoerr.ErrNotFound,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.UpdateConfirmation(context.Background(), tc.invitation)
			assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestInvitationUpdateRejection(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM invitations")
		require.Nil(t, err, fmt.Sprintf("clean invitations unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM domains")
		require.Nil(t, err, fmt.Sprintf("clean domains unexpected error: %s", err))
	})
	repo := postgres.NewRepository(database)

	dom := saveDomain(t, repo)

	invitation := domains.Invitation{
		InvitedBy:     testsutil.GenerateUUID(t),
		InviteeUserID: testsutil.GenerateUUID(t),
		DomainID:      dom.ID,
		DomainName:    dom.Name,
		RoleID:        testsutil.GenerateUUID(t),
		RoleName:      roleName,
		CreatedAt:     time.Now(),
	}
	err := repo.SaveInvitation(context.Background(), invitation)
	require.Nil(t, err, fmt.Sprintf("create invitation unexpected error: %s", err))

	cases := []struct {
		desc       string
		invitation domains.Invitation
		err        error
	}{
		{
			desc: "update invitation successfully",
			invitation: domains.Invitation{
				DomainID:      invitation.DomainID,
				InviteeUserID: invitation.InviteeUserID,
				RejectedAt:    time.Now(),
			},
			err: nil,
		},
		{
			desc: "update invitation with invalid invitee user id",
			invitation: domains.Invitation{
				InviteeUserID: testsutil.GenerateUUID(t),
				DomainID:      invitation.InviteeUserID,
				RejectedAt:    time.Now(),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "update invitation with invalid domain",
			invitation: domains.Invitation{
				InviteeUserID: invitation.InviteeUserID,
				DomainID:      testsutil.GenerateUUID(t),
				RejectedAt:    time.Now(),
			},
			err: repoerr.ErrNotFound,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.UpdateRejection(context.Background(), tc.invitation)
			assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestInvitationDelete(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM invitations")
		require.Nil(t, err, fmt.Sprintf("clean invitations unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM domains")
		require.Nil(t, err, fmt.Sprintf("clean domains unexpected error: %s", err))
	})
	repo := postgres.NewRepository(database)

	dom := saveDomain(t, repo)

	invitation := domains.Invitation{
		InvitedBy:     testsutil.GenerateUUID(t),
		InviteeUserID: testsutil.GenerateUUID(t),
		DomainID:      dom.ID,
		DomainName:    dom.Name,
		RoleID:        testsutil.GenerateUUID(t),
		RoleName:      roleName,
		CreatedAt:     time.Now(),
	}
	err := repo.SaveInvitation(context.Background(), invitation)
	require.Nil(t, err, fmt.Sprintf("create invitation unexpected error: %s", err))

	cases := []struct {
		desc       string
		invitation domains.Invitation
		err        error
	}{
		{
			desc: "delete invitation successfully",
			invitation: domains.Invitation{
				InviteeUserID: invitation.InviteeUserID,
				DomainID:      invitation.DomainID,
			},
			err: nil,
		},
		{
			desc: "delete invitation with invalid invitation id",
			invitation: domains.Invitation{
				InviteeUserID: testsutil.GenerateUUID(t),
				DomainID:      testsutil.GenerateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:       "delete invitation with empty invitation id",
			invitation: domains.Invitation{},
			err:        repoerr.ErrNotFound,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.DeleteInvitation(context.Background(), tc.invitation.InviteeUserID, tc.invitation.DomainID)
			assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func saveDomain(t *testing.T, repo domains.Repository) domains.Domain {
	domain := domains.Domain{
		ID:    testsutil.GenerateUUID(t),
		Name:  "test",
		Alias: "test",
		Tags:  []string{"test"},
		Metadata: map[string]interface{}{
			"test": "test",
		},
		CreatedBy: userID,
		UpdatedBy: userID,
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Millisecond),
		Status:    domains.EnabledStatus,
	}

	_, err := repo.SaveDomain(context.Background(), domain)
	require.Nil(t, err, fmt.Sprintf("failed to save domain %s", domain.ID))

	return domain
}
