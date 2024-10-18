// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package journal_test

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/journal"
	"github.com/absmach/magistrala/journal/mocks"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	authnmocks "github.com/absmach/magistrala/pkg/authn/mocks"
	mgauthz "github.com/absmach/magistrala/pkg/authz"
	authzmocks "github.com/absmach/magistrala/pkg/authz/mocks"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	validJournal = journal.Journal{
		Operation:  "user.create",
		OccurredAt: time.Now().Add(-time.Hour),
		Attributes: map[string]interface{}{
			"temperature": rand.Float64(),
			"humidity":    rand.Float64(),
		},
		Metadata: map[string]interface{}{
			"sensor_id": rand.Intn(1000),
		},
	}
	idProvider = uuid.New()
)

func TestSave(t *testing.T) {
	repo := new(mocks.Repository)
	authn := new(authnmocks.Authentication)
	authz := new(authzmocks.Authorization)
	svc := journal.NewService(authn, authz, idProvider, repo)

	cases := []struct {
		desc    string
		journal journal.Journal
		repoErr error
		err     error
	}{
		{
			desc:    "successful with ID and EntityType",
			journal: validJournal,
			repoErr: nil,
			err:     nil,
		},
		{
			desc:    "with repo error",
			repoErr: repoerr.ErrCreateEntity,
			err:     repoerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("Save", context.Background(), mock.Anything).Return(tc.repoErr)
			err := svc.Save(context.Background(), tc.journal)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			repoCall.Unset()
		})
	}
}

func TestReadAll(t *testing.T) {
	repo := new(mocks.Repository)
	authn := new(authnmocks.Authentication)
	authz := new(authzmocks.Authorization)
	svc := journal.NewService(authn, authz, idProvider, repo)

	validToken := "token"
	validPage := journal.Page{
		Offset:     0,
		Limit:      10,
		EntityID:   testsutil.GenerateUUID(t),
		EntityType: journal.ThingEntity,
	}

	cases := []struct {
		desc        string
		token       string
		page        journal.Page
		resp        journal.JournalsPage
		identifyRes mgauthn.Session
		identifyErr error
		authErr     error
		repoErr     error
		err         error
	}{
		{
			desc:  "successful",
			token: validToken,
			page:  validPage,
			resp: journal.JournalsPage{
				Total:    1,
				Offset:   0,
				Limit:    10,
				Journals: []journal.Journal{validJournal},
			},
			identifyRes: mgauthn.Session{DomainUserID: testsutil.GenerateUUID(t), UserID: testsutil.GenerateUUID(t)},
			authErr:     nil,
			repoErr:     nil,
			err:         nil,
		},
		{
			desc:  "successful for user",
			token: validToken,
			page: journal.Page{
				Offset:     0,
				Limit:      10,
				EntityID:   testsutil.GenerateUUID(t),
				EntityType: journal.UserEntity,
			},
			resp: journal.JournalsPage{
				Total:    1,
				Offset:   0,
				Limit:    10,
				Journals: []journal.Journal{validJournal},
			},
			identifyRes: mgauthn.Session{DomainUserID: testsutil.GenerateUUID(t), UserID: testsutil.GenerateUUID(t)},
			authErr:     nil,
			repoErr:     nil,
			err:         nil,
		},
		{
			desc:        "with identify error",
			token:       validToken,
			page:        validPage,
			resp:        journal.JournalsPage{},
			identifyRes: mgauthn.Session{},
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "with repo error",
			token:       validToken,
			page:        validPage,
			resp:        journal.JournalsPage{},
			identifyRes: mgauthn.Session{DomainUserID: testsutil.GenerateUUID(t), UserID: testsutil.GenerateUUID(t)},
			repoErr:     repoerr.ErrViewEntity,
			err:         repoerr.ErrViewEntity,
		},
		{
			desc:        "with failed to authorize",
			token:       validToken,
			page:        validPage,
			resp:        journal.JournalsPage{},
			identifyRes: mgauthn.Session{DomainUserID: testsutil.GenerateUUID(t), UserID: testsutil.GenerateUUID(t)},
			authErr:     svcerr.ErrAuthorization,
			repoErr:     nil,
			err:         svcerr.ErrAuthorization,
		},
		{
			desc:        "with error on authorize",
			token:       validToken,
			page:        validPage,
			resp:        journal.JournalsPage{},
			identifyRes: mgauthn.Session{DomainUserID: testsutil.GenerateUUID(t), UserID: testsutil.GenerateUUID(t)},
			authErr:     svcerr.ErrAuthorization,
			repoErr:     nil,
			err:         svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authReq := mgauthz.PolicyReq{
				SubjectType: policies.UserType,
				SubjectKind: policies.UsersKind,
				Subject:     tc.identifyRes.DomainUserID,
				ObjectType:  tc.page.EntityType.AuthString(),
				Object:      tc.page.EntityID,
				Permission:  policies.ViewPermission,
			}
			if tc.page.EntityType == journal.UserEntity {
				authReq.Permission = policies.AdminPermission
				authReq.ObjectType = policies.PlatformType
				authReq.Object = policies.MagistralaObject
				authReq.Subject = tc.identifyRes.UserID
			}
			authCall := authn.On("Authenticate", context.Background(), tc.token).Return(tc.identifyRes, tc.identifyErr)
			authCall1 := authz.On("Authorize", context.Background(), authReq).Return(tc.authErr)
			repoCall := repo.On("RetrieveAll", context.Background(), tc.page).Return(tc.resp, tc.repoErr)
			resp, err := svc.RetrieveAll(context.Background(), tc.token, tc.page)
			if tc.err == nil {
				assert.Equal(t, tc.resp, resp, tc.desc)
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			repoCall.Unset()
			authCall.Unset()
			authCall1.Unset()
		})
	}
}
