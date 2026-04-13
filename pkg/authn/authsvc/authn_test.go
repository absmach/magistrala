// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package authsvc

import (
	"context"
	"testing"

	grpcAuthV1 "github.com/absmach/magistrala/api/grpc/auth/v1"
	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

type authClient struct {
	res *grpcAuthV1.AuthNRes
	err error
}

func (ac authClient) Authenticate(context.Context, *grpcAuthV1.AuthNReq, ...grpc.CallOption) (*grpcAuthV1.AuthNRes, error) {
	return ac.res, ac.err
}

func (ac authClient) Authorize(context.Context, *grpcAuthV1.AuthZReq, ...grpc.CallOption) (*grpcAuthV1.AuthZRes, error) {
	return &grpcAuthV1.AuthZRes{}, nil
}

func TestAuthenticateUsesTokenTypeForPAT(t *testing.T) {
	const (
		token   = "token"
		tokenID = "token-id"
		userID  = "user-id"
	)

	cases := []struct {
		desc string
		res  *grpcAuthV1.AuthNRes
		want authn.Session
	}{
		{
			desc: "access token with id is not treated as PAT",
			res:  &grpcAuthV1.AuthNRes{Id: tokenID, UserId: userID, UserRole: uint32(auth.UserRole), TokenType: uint32(auth.AccessKey), Verified: true},
			want: authn.Session{Type: authn.AccessToken, UserID: userID, Role: authn.UserRole, Verified: true},
		},
		{
			desc: "PAT token type is treated as PAT",
			res:  &grpcAuthV1.AuthNRes{Id: tokenID, UserId: userID, UserRole: uint32(auth.UserRole), TokenType: uint32(auth.PersonalAccessToken)},
			want: authn.Session{Type: authn.PersonalAccessToken, PatID: tokenID, UserID: userID, Role: authn.UserRole},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svc := authentication{authSvcClient: authClient{res: tc.res}}
			got, err := svc.Authenticate(context.Background(), token)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
