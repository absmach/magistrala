// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package eventlogs_test

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	authmocks "github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/eventlogs"
	"github.com/absmach/magistrala/eventlogs/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/stretchr/testify/assert"
)

func TestReadAll(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.AuthClient)
	svc := eventlogs.NewService(repo, authsvc)

	validToken := "token"
	validPage := eventlogs.Page{
		Offset:     0,
		Limit:      10,
		ID:         testsutil.GenerateUUID(t),
		EntityType: auth.UserType,
	}

	cases := []struct {
		desc    string
		token   string
		page    eventlogs.Page
		resp    eventlogs.EventsPage
		authRes *magistrala.AuthorizeRes
		authErr error
		repoErr error
		err     error
	}{
		{
			desc:  "successful",
			token: validToken,
			page:  validPage,
			resp: eventlogs.EventsPage{
				Total:  1,
				Offset: 0,
				Limit:  10,
				Events: []eventlogs.Event{
					{
						ID:         testsutil.GenerateUUID(t),
						Operation:  "user.create",
						OccurredAt: time.Now().Add(-time.Hour),
						Payload: map[string]interface{}{
							"temperature": rand.Float64(),
							"humidity":    rand.Float64(),
							"sensor_id":   rand.Intn(1000),
						},
					},
				},
			},
			authRes: &magistrala.AuthorizeRes{Authorized: true},
			authErr: nil,
			repoErr: nil,
			err:     nil,
		},
		{
			desc:    "invalid token",
			token:   "invalid",
			page:    validPage,
			authRes: &magistrala.AuthorizeRes{Authorized: false},
			authErr: svcerr.ErrAuthorization,
			err:     svcerr.ErrAuthorization,
		},
		{
			desc:    "with repo error",
			token:   validToken,
			page:    validPage,
			resp:    eventlogs.EventsPage{},
			authRes: &magistrala.AuthorizeRes{Authorized: true},
			authErr: nil,
			repoErr: repoerr.ErrViewEntity,
			err:     repoerr.ErrViewEntity,
		},
		{
			desc:    "with failed to authorize",
			token:   validToken,
			page:    validPage,
			resp:    eventlogs.EventsPage{},
			authRes: &magistrala.AuthorizeRes{Authorized: false},
			authErr: nil,
			repoErr: nil,
			err:     svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authReq := &magistrala.AuthorizeReq{
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Subject:     tc.token,
				Permission:  auth.ViewPermission,
				ObjectType:  tc.page.EntityType,
				Object:      tc.page.ID,
			}
			repocall := authsvc.On("Authorize", context.Background(), authReq).Return(tc.authRes, tc.authErr)
			repocall1 := repo.On("RetrieveAll", context.Background(), tc.page).Return(tc.resp, tc.repoErr)
			resp, err := svc.ReadAll(context.Background(), tc.token, tc.page)
			if tc.err == nil {
				assert.Equal(t, tc.resp, resp, tc.desc)
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			repocall.Unset()
			repocall1.Unset()
		})
	}
}
