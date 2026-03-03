// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	grpcTokenV1 "github.com/absmach/supermq/api/grpc/token/v1"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/users"
	"github.com/absmach/supermq/users/events"
	"github.com/absmach/supermq/users/mocks"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	storeClient  *redis.Client
	storeURL     string
	validSession = authn.Session{
		UserID: testsutil.GenerateUUID(&testing.T{}),
	}
	validUser      = generateTestUser(&testing.T{})
	validUsersPage = users.UsersPage{
		Page: users.Page{
			Limit:  10,
			Offset: 0,
			Total:  1,
		},
		Users: []users.User{validUser},
	}
)

func newEventStoreMiddleware(t *testing.T) (*mocks.Service, users.Service) {
	svc := new(mocks.Service)
	nsvc, err := events.NewEventStoreMiddleware(context.Background(), svc, storeURL)
	require.Nil(t, err, fmt.Sprintf("create events store middleware failed with unexpected error: %s", err))

	return svc, nsvc
}

func TestMain(m *testing.M) {
	code := testsutil.RunRedisTest(m, &storeClient, &storeURL)
	os.Exit(code)
}

func TestRegister(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validID := testsutil.GenerateUUID(t)
	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, validID)

	cases := []struct {
		desc         string
		session      authn.Session
		user         users.User
		selfRegister bool
		svcRes       users.User
		svcErr       error
		resp         users.User
		err          error
	}{
		{
			desc:         "publish successfully",
			session:      validSession,
			user:         validUser,
			selfRegister: true,
			svcRes:       validUser,
			svcErr:       nil,
			resp:         validUser,
			err:          nil,
		},
		{
			desc:         "failed to pusblish with service error",
			session:      validSession,
			user:         validUser,
			selfRegister: true,
			svcRes:       users.User{},
			svcErr:       svcerr.ErrCreateEntity,
			resp:         users.User{},
			err:          svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("Register", validCtx, tc.session, tc.user, tc.selfRegister).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.Register(validCtx, tc.session, tc.user, tc.selfRegister)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestSendVerification(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))
	cases := []struct {
		desc    string
		session authn.Session
		userID  string
		svcErr  error
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			userID:  validUser.ID,
			svcErr:  nil,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			userID:  validUser.ID,
			svcErr:  svcerr.ErrCreateEntity,
			err:     svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("SendVerification", validCtx, tc.session).Return(tc.svcErr)
			err := nsvc.SendVerification(validCtx, tc.session)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestVerifyEmail(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))
	validToken := "validVerificationToken"
	cases := []struct {
		desc              string
		verificationToken string
		svcRes            users.User
		svcErr            error
		resp              users.User
		err               error
	}{
		{
			desc:              "publish successfully",
			verificationToken: validToken,
			svcRes:            validUser,
			svcErr:            nil,
			resp:              validUser,
			err:               nil,
		},
		{
			desc:              "failed to publish with service error",
			verificationToken: validToken,
			svcRes:            users.User{},
			svcErr:            svcerr.ErrCreateEntity,
			resp:              users.User{},
			err:               svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("VerifyEmail", validCtx, tc.verificationToken).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.VerifyEmail(validCtx, tc.verificationToken)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestUpdate(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	updatedUser := validUser
	updatedUser.FirstName = "updatedFirstName"

	cases := []struct {
		desc    string
		session authn.Session
		userID  string
		userReq users.UserReq
		svcRes  users.User
		svcErr  error
		resp    users.User
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			userID:  validSession.UserID,
			userReq: users.UserReq{
				FirstName: &updatedUser.FirstName,
			},
			svcRes: updatedUser,
			svcErr: nil,
			resp:   updatedUser,
			err:    nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			userID:  validSession.UserID,
			userReq: users.UserReq{
				FirstName: &updatedUser.FirstName,
			},
			svcRes: users.User{},
			svcErr: svcerr.ErrUpdateEntity,
			resp:   users.User{},
			err:    svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("Update", validCtx, tc.session, tc.userID, tc.userReq).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.Update(validCtx, tc.session, tc.userID, tc.userReq)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestUpdateRole(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))
	updatedUser := validUser
	updatedUser.Role = users.AdminRole

	cases := []struct {
		desc    string
		session authn.Session
		user    users.User
		svcRes  users.User
		svcErr  error
		resp    users.User
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			user:    updatedUser,
			svcRes:  updatedUser,
			svcErr:  nil,
			resp:    updatedUser,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			user:    updatedUser,
			svcRes:  users.User{},
			svcErr:  svcerr.ErrUpdateEntity,
			resp:    users.User{},
			err:     svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UpdateRole", validCtx, tc.session, tc.user).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.UpdateRole(validCtx, tc.session, tc.user)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestUpdateTags(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))
	updatedUser := validUser
	updatedUser.Tags = []string{"newTag1", "newTag2"}

	cases := []struct {
		desc    string
		session authn.Session
		userID  string
		userReq users.UserReq
		svcRes  users.User
		svcErr  error
		resp    users.User
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			userID:  validSession.UserID,
			userReq: users.UserReq{
				Tags: &updatedUser.Tags,
			},
			svcRes: updatedUser,
			svcErr: nil,
			resp:   updatedUser,
			err:    nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			userID:  validSession.UserID,
			userReq: users.UserReq{
				Tags: &updatedUser.Tags,
			},
			svcRes: users.User{},
			svcErr: svcerr.ErrUpdateEntity,
			resp:   users.User{},
			err:    svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UpdateTags", validCtx, tc.session, tc.userID, tc.userReq).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.UpdateTags(validCtx, tc.session, tc.userID, tc.userReq)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestUpdateSecret(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))
	updatedUser := validUser
	updatedUser.Credentials.Secret = "newSecret"

	cases := []struct {
		desc      string
		session   authn.Session
		oldSecret string
		newSecret string
		svcRes    users.User
		svcErr    error
		resp      users.User
		err       error
	}{
		{
			desc:      "publish successfully",
			session:   validSession,
			oldSecret: "secret",
			newSecret: "newSecret",
			svcRes:    updatedUser,
			svcErr:    nil,
			resp:      updatedUser,
			err:       nil,
		},
		{
			desc:      "failed to publish with service error",
			session:   validSession,
			oldSecret: "secret",
			newSecret: "newSecret",
			svcRes:    users.User{},
			svcErr:    svcerr.ErrUpdateEntity,
			resp:      users.User{},
			err:       svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UpdateSecret", validCtx, tc.session, tc.oldSecret, tc.newSecret).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.UpdateSecret(validCtx, tc.session, tc.oldSecret, tc.newSecret)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestUpdateUsername(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))
	updatedUser := validUser
	updatedUser.Credentials.Username = "newUsername"

	cases := []struct {
		desc        string
		session     authn.Session
		userID      string
		newUsername string
		svcRes      users.User
		svcErr      error
		resp        users.User
		err         error
	}{
		{
			desc:        "publish successfully",
			session:     validSession,
			userID:      validSession.UserID,
			newUsername: "newUsername",
			svcRes:      updatedUser,
			svcErr:      nil,
			resp:        updatedUser,
			err:         nil,
		},
		{
			desc:        "failed to publish with service error",
			session:     validSession,
			userID:      validSession.UserID,
			newUsername: "newUsername",
			svcRes:      users.User{},
			svcErr:      svcerr.ErrUpdateEntity,
			resp:        users.User{},
			err:         svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UpdateUsername", validCtx, tc.session, tc.userID, tc.newUsername).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.UpdateUsername(validCtx, tc.session, tc.userID, tc.newUsername)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestUpdateProfilePicture(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))
	updatedUser := validUser
	updatedUser.ProfilePicture = "https://example.com/newprofilepic.jpg"

	cases := []struct {
		desc    string
		session authn.Session
		userID  string
		userReq users.UserReq
		svcRes  users.User
		svcErr  error
		resp    users.User
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			userID:  validSession.UserID,
			userReq: users.UserReq{
				ProfilePicture: &updatedUser.ProfilePicture,
			},
			svcRes: updatedUser,
			svcErr: nil,
			resp:   updatedUser,
			err:    nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			userID:  validSession.UserID,
			userReq: users.UserReq{
				ProfilePicture: &updatedUser.ProfilePicture,
			},
			svcRes: users.User{},
			svcErr: svcerr.ErrUpdateEntity,
			resp:   users.User{},
			err:    svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UpdateProfilePicture", validCtx, tc.session, tc.userID, tc.userReq).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.UpdateProfilePicture(validCtx, tc.session, tc.userID, tc.userReq)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestUpdateEmail(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))
	updatedUser := validUser
	updatedUser.Email = "updatedemail@example.com"

	cases := []struct {
		desc     string
		session  authn.Session
		userID   string
		newEmail string
		svcRes   users.User
		svcErr   error
		resp     users.User
		err      error
	}{
		{
			desc:     "publish successfully",
			session:  validSession,
			userID:   validSession.UserID,
			newEmail: "updatedemail@example.com",
			svcRes:   updatedUser,
			svcErr:   nil,
			resp:     updatedUser,
			err:      nil,
		},
		{
			desc:     "failed to publish with service error",
			session:  validSession,
			userID:   validSession.UserID,
			newEmail: "updatedemail@example.com",
			svcRes:   users.User{},
			svcErr:   svcerr.ErrUpdateEntity,
			resp:     users.User{},
			err:      svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UpdateEmail", validCtx, tc.session, tc.userID, tc.newEmail).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.UpdateEmail(validCtx, tc.session, tc.userID, tc.newEmail)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestView(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc    string
		session authn.Session
		userID  string
		svcRes  users.User
		svcErr  error
		resp    users.User
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			userID:  validSession.UserID,
			svcRes:  validUser,
			svcErr:  nil,
			resp:    validUser,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			userID:  validSession.UserID,
			svcRes:  users.User{},
			svcErr:  svcerr.ErrViewEntity,
			resp:    users.User{},
			err:     svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("View", validCtx, tc.session, tc.userID).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.View(validCtx, tc.session, tc.userID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestViewProfile(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc    string
		session authn.Session
		svcRes  users.User
		svcErr  error
		resp    users.User
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			svcRes:  validUser,
			svcErr:  nil,
			resp:    validUser,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			svcRes:  users.User{},
			svcErr:  svcerr.ErrViewEntity,
			resp:    users.User{},
			err:     svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ViewProfile", validCtx, tc.session).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.ViewProfile(validCtx, tc.session)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestListUsers(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		pageMeta users.Page
		svcRes   users.UsersPage
		svcErr   error
		resp     users.UsersPage
		err      error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			pageMeta: users.Page{
				Limit:  10,
				Offset: 0,
			},
			svcRes: validUsersPage,
			svcErr: nil,
			resp:   validUsersPage,
			err:    nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			pageMeta: users.Page{
				Limit:  10,
				Offset: 0,
			},
			svcRes: users.UsersPage{},
			svcErr: svcerr.ErrViewEntity,
			resp:   users.UsersPage{},
			err:    svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ListUsers", validCtx, tc.session, tc.pageMeta).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.ListUsers(validCtx, tc.session, tc.pageMeta)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestSearchUsers(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		pageMeta users.Page
		svcRes   users.UsersPage
		svcErr   error
		resp     users.UsersPage
		err      error
	}{
		{
			desc: "publish successfully",
			pageMeta: users.Page{
				Limit:  10,
				Offset: 0,
			},
			svcRes: validUsersPage,
			svcErr: nil,
			resp:   validUsersPage,
			err:    nil,
		},
		{
			desc: "failed to publish with service error",
			pageMeta: users.Page{
				Limit:  10,
				Offset: 0,
			},
			svcRes: users.UsersPage{},
			svcErr: svcerr.ErrViewEntity,
			resp:   users.UsersPage{},
			err:    svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("SearchUsers", validCtx, tc.pageMeta).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.SearchUsers(validCtx, tc.pageMeta)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestEnable(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc    string
		session authn.Session
		userID  string
		svcRes  users.User
		svcErr  error
		resp    users.User
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			userID:  validSession.UserID,
			svcRes:  validUser,
			svcErr:  nil,
			resp:    validUser,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			userID:  validSession.UserID,
			svcRes:  users.User{},
			svcErr:  svcerr.ErrUpdateEntity,
			resp:    users.User{},
			err:     svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("Enable", validCtx, tc.session, tc.userID).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.Enable(validCtx, tc.session, tc.userID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestDisable(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))
	cases := []struct {
		desc    string
		session authn.Session
		userID  string
		svcRes  users.User
		svcErr  error
		resp    users.User
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			userID:  validSession.UserID,
			svcRes:  validUser,
			svcErr:  nil,
			resp:    validUser,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			userID:  validSession.UserID,
			svcRes:  users.User{},
			svcErr:  svcerr.ErrUpdateEntity,
			resp:    users.User{},
			err:     svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("Disable", validCtx, tc.session, tc.userID).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.Disable(validCtx, tc.session, tc.userID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestIdentify(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc    string
		session authn.Session
		svcRes  string
		svcErr  error
		resp    string
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			svcRes:  validUser.ID,
			svcErr:  nil,
			resp:    validUser.ID,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			svcRes:  "",
			svcErr:  svcerr.ErrViewEntity,
			resp:    "",
			err:     svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("Identify", validCtx, tc.session).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.Identify(validCtx, tc.session)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestSendPasswordReset(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))
	cases := []struct {
		desc   string
		email  string
		svcErr error
		err    error
	}{
		{
			desc:   "publish successfully",
			email:  validUser.Email,
			svcErr: nil,
			err:    nil,
		},
		{
			desc:   "failed to publish with service error",
			email:  validUser.Email,
			svcErr: svcerr.ErrCreateEntity,
			err:    svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("SendPasswordReset", validCtx, tc.email).Return(tc.svcErr)
			err := nsvc.SendPasswordReset(validCtx, tc.email)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestIssueToken(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))
	validToken := &grpcTokenV1.Token{
		AccessToken: "validAccessToken",
	}

	cases := []struct {
		desc        string
		username    string
		secret      string
		description string
		svcRes      *grpcTokenV1.Token
		svcErr      error
		resp        *grpcTokenV1.Token
		err         error
	}{
		{
			desc:        "publish successfully",
			username:    validUser.Credentials.Username,
			secret:      validUser.Credentials.Secret,
			description: "valid token",
			svcRes:      validToken,
			svcErr:      nil,
			resp:        validToken,
			err:         nil,
		},
		{
			desc:     "failed to publish with service error",
			username: validUser.Credentials.Username,
			secret:   validUser.Credentials.Secret,
			svcRes:   nil,
			svcErr:   svcerr.ErrCreateEntity,
			resp:     nil,
			err:      svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("IssueToken", validCtx, tc.username, tc.secret, tc.description).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.IssueToken(validCtx, tc.username, tc.secret, tc.description)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestRefreshToken(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))
	validRefreshToken := "validRefreshToken"
	validToken := &grpcTokenV1.Token{
		AccessToken: "validAccessToken",
	}

	cases := []struct {
		desc         string
		session      authn.Session
		refreshToken string
		svcRes       *grpcTokenV1.Token
		svcErr       error
		resp         *grpcTokenV1.Token
		err          error
	}{
		{
			desc:         "publish successfully",
			session:      validSession,
			refreshToken: validRefreshToken,
			svcRes:       validToken,
			svcErr:       nil,
			resp:         validToken,
			err:          nil,
		},
		{
			desc:         "failed to publish with service error",
			session:      validSession,
			refreshToken: validRefreshToken,
			svcRes:       nil,
			svcErr:       svcerr.ErrCreateEntity,
			resp:         nil,
			err:          svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("RefreshToken", validCtx, tc.session, tc.refreshToken).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.RefreshToken(validCtx, tc.session, tc.refreshToken)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestRevokeRefreshToken(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))
	validTokenID := "validTokenID"

	cases := []struct {
		desc    string
		session authn.Session
		tokenID string
		svcErr  error
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			tokenID: validTokenID,
			svcErr:  nil,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			tokenID: validTokenID,
			svcErr:  svcerr.ErrUpdateEntity,
			err:     svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("RevokeRefreshToken", validCtx, tc.session, tc.tokenID).Return(tc.svcErr)
			err := nsvc.RevokeRefreshToken(validCtx, tc.session, tc.tokenID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestListActiveRefreshTokens(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))
	validTokensList := &grpcTokenV1.ListUserRefreshTokensRes{
		RefreshTokens: []*grpcTokenV1.RefreshToken{
			{Id: "token1", Description: "token1"},
			{Id: "token2", Description: "token2"},
		},
	}

	cases := []struct {
		desc    string
		session authn.Session
		svcRes  *grpcTokenV1.ListUserRefreshTokensRes
		svcErr  error
		resp    *grpcTokenV1.ListUserRefreshTokensRes
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			svcRes:  validTokensList,
			svcErr:  nil,
			resp:    validTokensList,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			svcRes:  nil,
			svcErr:  svcerr.ErrViewEntity,
			resp:    nil,
			err:     svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ListActiveRefreshTokens", validCtx, tc.session).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.ListActiveRefreshTokens(validCtx, tc.session)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestResetSecret(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))
	newSecret := "newSecret"

	cases := []struct {
		desc    string
		session authn.Session
		secret  string
		svcErr  error
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			secret:  newSecret,
			svcErr:  nil,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			secret:  newSecret,
			svcErr:  svcerr.ErrUpdateEntity,
			err:     svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ResetSecret", validCtx, tc.session, tc.secret).Return(tc.svcErr)
			err := nsvc.ResetSecret(validCtx, tc.session, tc.secret)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestOAuthCallback(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc   string
		user   users.User
		svcRes users.User
		svcErr error
		resp   users.User
		err    error
	}{
		{
			desc:   "publish successfully",
			user:   validUser,
			svcRes: validUser,
			svcErr: nil,
			resp:   validUser,
			err:    nil,
		},
		{
			desc:   "failed to publish with service error",
			user:   validUser,
			svcRes: users.User{},
			svcErr: svcerr.ErrCreateEntity,
			resp:   users.User{},
			err:    svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("OAuthCallback", validCtx, tc.user).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.OAuthCallback(validCtx, tc.user)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestDelete(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc    string
		session authn.Session
		userID  string
		svcErr  error
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			userID:  validSession.UserID,
			svcErr:  nil,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			userID:  validSession.UserID,
			svcErr:  svcerr.ErrRemoveEntity,
			err:     svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("Delete", validCtx, tc.session, tc.userID).Return(tc.svcErr)
			err := nsvc.Delete(validCtx, tc.session, tc.userID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestOAuthAddUserPolicy(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc   string
		user   users.User
		svcErr error
		err    error
	}{
		{
			desc:   "publish successfully",
			user:   validUser,
			svcErr: nil,
			err:    nil,
		},
		{
			desc:   "failed to publish with service error",
			user:   validUser,
			svcErr: svcerr.ErrCreateEntity,
			err:    svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("OAuthAddUserPolicy", validCtx, tc.user).Return(tc.svcErr)
			err := nsvc.OAuthAddUserPolicy(validCtx, tc.user)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func generateTestUser(t *testing.T) users.User {
	createdAt, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	assert.Nil(t, err, fmt.Sprintf("Unexpected error parsing time: %v", err))
	return users.User{
		ID:        testsutil.GenerateUUID(t),
		FirstName: "userfirstname",
		LastName:  "userlastname",
		Email:     "useremail@example.com",
		Credentials: users.Credentials{
			Username: "username",
			Secret:   "secret",
		},
		Tags: []string{"tag1", "tag2"},
		PrivateMetadata: users.Metadata{
			"key1": "value1",
			"key2": "value2",
		},
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		Status:    users.EnabledStatus,
		Role:      users.UserRole,
	}
}
