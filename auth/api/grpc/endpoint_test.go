// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	grpcapi "github.com/absmach/magistrala/auth/api/grpc"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const (
	port            = 8081
	secret          = "secret"
	email           = "test@example.com"
	id              = "testID"
	thingsType      = "things"
	usersType       = "users"
	description     = "Description"
	groupName       = "mgx"
	adminpermission = "admin"

	authoritiesObj  = "authorities"
	memberRelation  = "member"
	loginDuration   = 30 * time.Minute
	refreshDuration = 24 * time.Hour
	invalidDuration = 7 * 24 * time.Hour
)

func startGRPCServer(svc auth.Service, port int) {
	listener, _ := net.Listen("tcp", fmt.Sprintf(":%d", port))
	server := grpc.NewServer()
	magistrala.RegisterAuthServiceServer(server, grpcapi.NewServer(svc))
	go func() {
		if err := server.Serve(listener); err != nil {
			panic(fmt.Sprintf("failed to serve: %s", err))
		}
	}()
}

func TestIssue(t *testing.T) {
	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.Dial(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := grpcapi.NewClient(conn, time.Second)

	cases := []struct {
		desc  string
		id    string
		email string
		kind  auth.KeyType
		err   error
		code  codes.Code
	}{
		{
			desc:  "issue for user with valid token",
			id:    id,
			email: email,
			kind:  auth.AccessKey,
			err:   nil,
			code:  codes.OK,
		},
		{
			desc:  "issue recovery key",
			id:    id,
			email: email,
			kind:  auth.RecoveryKey,
			err:   nil,
			code:  codes.OK,
		},
		{
			desc:  "issue API key unauthenticated",
			id:    id,
			email: email,
			kind:  auth.APIKey,
			err:   errors.ErrAuthentication,
			code:  codes.Unauthenticated,
		},
		{
			desc:  "issue for invalid key type",
			id:    id,
			email: email,
			kind:  32,
			err:   errors.ErrMalformedEntity,
			code:  codes.InvalidArgument,
		},
		{
			desc:  "issue for user that exist",
			id:    "",
			email: "",
			kind:  auth.APIKey,
			err:   errors.ErrAuthentication,
			code:  codes.Unauthenticated,
		},
	}

	for _, tc := range cases {
		_, err := client.Issue(context.Background(), &magistrala.IssueReq{UserId: tc.id, Type: uint32(tc.kind)})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestIdentify(t *testing.T) {
	repocall := prepo.On("CheckPolicy", mock.Anything, mock.Anything).Return(nil)
	loginToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, User: id, IssuedAt: time.Now(), Domain: groupName})
	assert.Nil(t, err, fmt.Sprintf("Issuing user key expected to succeed: %s", err))
	repocall.Unset()

	recoveryToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.RecoveryKey, IssuedAt: time.Now(), Subject: id})
	assert.Nil(t, err, fmt.Sprintf("Issuing recovery key expected to succeed: %s", err))

	repocall1 := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, nil)
	apiToken, err := svc.Issue(context.Background(), loginToken.AccessToken, auth.Key{Type: auth.APIKey, IssuedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute), Subject: id})
	assert.Nil(t, err, fmt.Sprintf("Issuing API key expected to succeed: %s", err))
	repocall1.Unset()

	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.Dial(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := grpcapi.NewClient(conn, time.Second)

	cases := []struct {
		desc  string
		token string
		idt   *magistrala.IdentityRes
		err   error
		code  codes.Code
	}{
		{
			desc:  "identify user with user token",
			token: loginToken.AccessToken,
			idt:   &magistrala.IdentityRes{Id: id, UserId: id, DomainId: groupName},
			err:   nil,
			code:  codes.OK,
		},
		{
			desc:  "identify user with recovery token",
			token: recoveryToken.AccessToken,
			idt:   &magistrala.IdentityRes{Id: id},
			err:   nil,
			code:  codes.OK,
		},
		{
			desc:  "identify user with API token",
			token: apiToken.AccessToken,
			idt:   &magistrala.IdentityRes{Id: id},
			err:   nil,
			code:  codes.OK,
		},
		{
			desc:  "identify user with invalid user token",
			token: "invalid",
			idt:   &magistrala.IdentityRes{},
			err:   status.Error(codes.Unauthenticated, "unauthenticated access"),
			code:  codes.Unauthenticated,
		},
		{
			desc:  "identify user with empty token",
			token: "",
			idt:   &magistrala.IdentityRes{},
			err:   status.Error(codes.InvalidArgument, "received invalid token request"),
			code:  codes.Unauthenticated,
		},
	}

	for _, tc := range cases {
		repocall := krepo.On("Retrieve", mock.Anything, mock.Anything, mock.Anything).Return(auth.Key{Subject: id}, nil)
		idt, _ := client.Identify(context.Background(), &magistrala.IdentityReq{Token: tc.token})
		if idt != nil {
			assert.Equal(t, tc.idt, idt, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.idt, idt))
		}
		repocall.Unset()
	}
}

func TestAuthorize(t *testing.T) {
	repocall := prepo.On("CheckPolicy", mock.Anything, mock.Anything).Return(nil)
	token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, IssuedAt: time.Now(), Subject: id, User: id, Domain: groupName})
	assert.Nil(t, err, fmt.Sprintf("Issuing user key expected to succeed: %s", err))
	repocall.Unset()

	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.Dial(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := grpcapi.NewClient(conn, time.Second)

	cases := []struct {
		desc        string
		token       string
		subject     string
		subjecttype string
		object      string
		objecttype  string
		relation    string
		permission  string
		ar          *magistrala.AuthorizeRes
		err         error
		code        codes.Code
	}{
		{
			desc:        "authorize user with authorized token",
			token:       token.AccessToken,
			subject:     id,
			subjecttype: usersType,
			object:      authoritiesObj,
			objecttype:  usersType,
			relation:    memberRelation,
			permission:  adminpermission,
			ar:          &magistrala.AuthorizeRes{Authorized: true},
			err:         nil,
			code:        codes.OK,
		},
		{
			desc:     "authorize user with unauthorized relation",
			token:    token.AccessToken,
			subject:  id,
			object:   authoritiesObj,
			relation: "unauthorizedRelation",
			ar:       &magistrala.AuthorizeRes{Authorized: false},
			err:      nil,
			code:     codes.PermissionDenied,
		},
		{
			desc:     "authorize user with unauthorized object",
			token:    token.AccessToken,
			subject:  id,
			object:   "unauthorizedobject",
			relation: memberRelation,
			ar:       &magistrala.AuthorizeRes{Authorized: false},
			err:      nil,
			code:     codes.PermissionDenied,
		},
		{
			desc:     "authorize user with unauthorized subject",
			token:    token.AccessToken,
			subject:  "unauthorizedSubject",
			object:   authoritiesObj,
			relation: memberRelation,
			ar:       &magistrala.AuthorizeRes{Authorized: false},
			err:      nil,
			code:     codes.PermissionDenied,
		},
		{
			desc:     "authorize user with invalid ACL",
			token:    token.AccessToken,
			subject:  "",
			object:   "",
			relation: "",
			ar:       &magistrala.AuthorizeRes{Authorized: false},
			err:      nil,
			code:     codes.InvalidArgument,
		},
	}
	for _, tc := range cases {
		repocall := prepo.On("CheckPolicy", mock.Anything, mock.Anything).Return(nil)
		ar, _ := client.Authorize(context.Background(), &magistrala.AuthorizeReq{Subject: tc.subject, SubjectType: tc.subjecttype, Object: tc.object, ObjectType: tc.objecttype, Relation: tc.relation, Permission: tc.permission})
		if ar != nil {
			assert.Equal(t, tc.ar, ar, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.ar, ar))
		}
		repocall.Unset()
	}
}

func TestAddPolicy(t *testing.T) {
	token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, IssuedAt: time.Now(), Subject: id})
	assert.Nil(t, err, fmt.Sprintf("Issuing user key expected to succeed: %s", err))

	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.Dial(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := grpcapi.NewClient(conn, time.Second)

	groupAdminObj := "groupadmin"

	cases := []struct {
		desc        string
		token       string
		subject     string
		subjecttype string
		object      string
		objecttype  string
		relation    string
		permission  string
		ar          *magistrala.AddPolicyRes
		err         error
		code        codes.Code
	}{
		{
			desc:        "add groupadmin policy to user",
			token:       token.AccessToken,
			subject:     id,
			subjecttype: usersType,
			object:      groupAdminObj,
			objecttype:  usersType,
			relation:    memberRelation,
			permission:  adminpermission,
			ar:          &magistrala.AddPolicyRes{Authorized: true},
			err:         nil,
			code:        codes.OK,
		},
	}
	for _, tc := range cases {
		repocall := prepo.On("AddPolicy", mock.Anything, mock.Anything).Return(nil)
		apr, err := client.AddPolicy(context.Background(), &magistrala.AddPolicyReq{Subject: tc.subject, SubjectType: tc.subjecttype, Object: tc.object, ObjectType: tc.objecttype, Relation: tc.relation, Permission: tc.permission})
		if apr != nil {
			assert.Equal(t, tc.ar, apr, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.ar, apr))
		}
		repocall.Unset()

		e, ok := status.FromError(err)
		assert.True(t, ok, "gRPC status can't be extracted from the error")
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.code, e.Code()))
	}
}

func TestDeletePolicy(t *testing.T) {
	token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, IssuedAt: time.Now(), Subject: id})
	assert.Nil(t, err, fmt.Sprintf("Issuing user key expected to succeed: %s", err))

	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.Dial(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := grpcapi.NewClient(conn, time.Second)

	readRelation := "read"
	thingID := "thing"

	repocall := prepo.On("AddPolicy", mock.Anything, mock.Anything).Return(nil)
	apr, err := client.AddPolicy(context.Background(), &magistrala.AddPolicyReq{Domain: groupName, Subject: id, SubjectType: usersType, Object: thingID, ObjectType: thingsType, Permission: readRelation})
	assert.Nil(t, err, fmt.Sprintf("Adding read policy to user expected to succeed: %s", err))
	assert.True(t, apr.GetAuthorized(), fmt.Sprintf("Adding read policy expected to make user authorized, expected %v got %v", true, apr.GetAuthorized()))
	repocall.Unset()

	cases := []struct {
		desc        string
		token       string
		subject     string
		subjecttype string
		object      string
		objecttype  string
		relation    string
		permission  string
		dpr         *magistrala.DeletePolicyRes
		code        codes.Code
	}{
		{
			desc:        "delete valid policy",
			token:       token.AccessToken,
			subject:     id,
			subjecttype: usersType,
			object:      thingID,
			objecttype:  thingsType,
			relation:    readRelation,
			permission:  readRelation,
			dpr:         &magistrala.DeletePolicyRes{Deleted: true},
			code:        codes.OK,
		},
	}
	for _, tc := range cases {
		repocall := prepo.On("DeletePolicy", mock.Anything, mock.Anything).Return(nil)
		dpr, err := client.DeletePolicy(context.Background(), &magistrala.DeletePolicyReq{Subject: tc.subject, SubjectType: tc.subjecttype, Object: tc.object, ObjectType: tc.objecttype, Relation: tc.relation})
		e, ok := status.FromError(err)
		repocall.Unset()

		assert.True(t, ok, "gRPC status can't be extracted from the error")
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.code, e.Code()))
		assert.Equal(t, tc.dpr.GetDeleted(), dpr.GetDeleted(), fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.dpr.GetDeleted(), dpr.GetDeleted()))
	}
}
