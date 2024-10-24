// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/users"
	"github.com/go-kit/kit/endpoint"
)

func registrationEndpoint(svc users.Service, selfRegister bool) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createUserReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		session := authn.Session{}

		var ok bool
		if !selfRegister {
			session, ok = ctx.Value(api.SessionKey).(authn.Session)
			if !ok {
				return nil, svcerr.ErrAuthorization
			}
		}

		user, err := svc.Register(ctx, session, req.user, selfRegister)
		if err != nil {
			return nil, err
		}

		return createUserRes{
			User:    user,
			created: true,
		}, nil
	}
}

func viewEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewUserReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}
		user, err := svc.View(ctx, session, req.id)
		if err != nil {
			return nil, err
		}

		return viewUserRes{User: user}, nil
	}
}

func viewProfileEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}
		client, err := svc.ViewProfile(ctx, session)
		if err != nil {
			return nil, err
		}

		return viewUserRes{User: client}, nil
	}
}

func listUsersEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listUsersReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		pm := users.Page{
			Status:    req.status,
			Offset:    req.offset,
			Limit:     req.limit,
			Username:  req.userName,
			Tag:       req.tag,
			Metadata:  req.metadata,
			FirstName: req.firstName,
			LastName:  req.lastName,
			Email:     req.email,
			Order:     req.order,
			Dir:       req.dir,
			Id:        req.id,
		}

		page, err := svc.ListUsers(ctx, session, pm)
		if err != nil {
			return nil, err
		}

		res := usersPageRes{
			pageRes: pageRes{
				Total:  page.Total,
				Offset: page.Offset,
				Limit:  page.Limit,
			},
			Users: []viewUserRes{},
		}
		for _, user := range page.Users {
			res.Users = append(res.Users, viewUserRes{User: user})
		}

		return res, nil
	}
}

func searchUsersEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(searchUsersReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		pm := users.Page{
			Offset:   req.Offset,
			Limit:    req.Limit,
			Username: req.Username,
			Id:       req.Id,
			Order:    req.Order,
			Dir:      req.Dir,
		}
		page, err := svc.SearchUsers(ctx, pm)
		if err != nil {
			return nil, err
		}

		res := usersPageRes{
			pageRes: pageRes{
				Total:  page.Total,
				Offset: page.Offset,
				Limit:  page.Limit,
			},
			Users: []viewUserRes{},
		}
		for _, user := range page.Users {
			res.Users = append(res.Users, viewUserRes{User: user})
		}

		return res, nil
	}
}

func listMembersByGroupEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMembersByObjectReq)
		req.objectKind = "groups"
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		page, err := svc.ListMembers(ctx, session, req.objectKind, req.objectID, req.Page)
		if err != nil {
			return nil, err
		}

		return buildUsersResponse(page), nil
	}
}

func listMembersByChannelEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMembersByObjectReq)
		// In spiceDB schema, using the same 'group' type for both channels and groups, rather than having a separate type for channels.
		req.objectKind = "groups"
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		page, err := svc.ListMembers(ctx, session, req.objectKind, req.objectID, req.Page)
		if err != nil {
			return nil, err
		}

		return buildUsersResponse(page), nil
	}
}

func listMembersByThingEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMembersByObjectReq)
		req.objectKind = "things"
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		page, err := svc.ListMembers(ctx, session, req.objectKind, req.objectID, req.Page)
		if err != nil {
			return nil, err
		}

		return buildUsersResponse(page), nil
	}
}

func listMembersByDomainEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMembersByObjectReq)
		req.objectKind = "domains"
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		page, err := svc.ListMembers(ctx, session, req.objectKind, req.objectID, req.Page)
		if err != nil {
			return nil, err
		}

		return buildUsersResponse(page), nil
	}
}

func updateEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateUserReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		user := users.User{
			ID: req.id,
			Credentials: users.Credentials{
				Username: req.Username,
			},
			FirstName: req.FirstName,
			LastName:  req.LastName,
			Metadata:  req.Metadata,
		}

		user, err := svc.Update(ctx, session, user)
		if err != nil {
			return nil, err
		}

		return updateUserRes{User: user}, nil
	}
}

func updateTagsEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateUserTagsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		user := users.User{
			ID:   req.id,
			Tags: req.Tags,
		}

		user, err := svc.UpdateTags(ctx, session, user)
		if err != nil {
			return nil, err
		}

		return updateUserRes{User: user}, nil
	}
}

func updateEmailEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateUserEmailReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		user, err := svc.UpdateEmail(ctx, session, req.id, req.Email)
		if err != nil {
			return nil, err
		}

		return updateUserRes{User: user}, nil
	}
}

// Password reset request endpoint.
// When successful password reset link is generated.
// Link is generated using MG_TOKEN_RESET_ENDPOINT env.
// and value from Referer header for host.
// {Referer}+{MG_TOKEN_RESET_ENDPOINT}+{token=TOKEN}
// http://magistrala.com/reset-request?token=xxxxxxxxxxx.
// Email with a link is being sent to the user.
// When user clicks on a link it should get the ui with form to
// enter new password, when form is submitted token and new password
// must be sent as PUT request to 'password/reset' passwordResetEndpoint.
func passwordResetRequestEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(passwResetReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		if err := svc.GenerateResetToken(ctx, req.Email, req.Host); err != nil {
			return nil, err
		}

		return passwResetReqRes{Msg: MailSent}, nil
	}
}

// This is endpoint that actually sets new password in password reset flow.
// When user clicks on a link in email finally ends on this endpoint as explained in
// the comment above.
func passwordResetEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(resetTokenReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}
		if err := svc.ResetSecret(ctx, session, req.Password); err != nil {
			return nil, err
		}

		return passwChangeRes{}, nil
	}
}

func updateSecretEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateUserSecretReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}
		user, err := svc.UpdateSecret(ctx, session, req.OldSecret, req.NewSecret)
		if err != nil {
			return nil, err
		}

		return updateUserRes{User: user}, nil
	}
}

func updateUsernameEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateUsernameReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		user := users.User{
			ID:          req.id,
			Credentials: users.Credentials{Username: req.Username},
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		user, err := svc.UpdateUsername(ctx, session, user)
		if err != nil {
			return nil, err
		}

		return updateUserRes{User: user}, nil
	}
}

func updateProfilePictureEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateProfilePictureReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		user := users.User{
			ID:             req.id,
			ProfilePicture: req.ProfilePicture,
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		user, err := svc.Update(ctx, session, user)
		if err != nil {
			return nil, err
		}

		return updateUserRes{User: user}, nil
	}
}

func updateRoleEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateUserRoleReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		user := users.User{
			ID:   req.id,
			Role: req.role,
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		user, err := svc.Update(ctx, session, user)
		if err != nil {
			return nil, err
		}

		return updateUserRes{User: user}, nil
	}
}

func issueTokenEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(loginUserReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		token, err := svc.IssueToken(ctx, req.Email, req.Secret, req.DomainID)
		if err != nil {
			return nil, err
		}

		return tokenRes{
			AccessToken:  token.GetAccessToken(),
			RefreshToken: token.GetRefreshToken(),
			AccessType:   token.GetAccessType(),
		}, nil
	}
}

func refreshTokenEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(tokenReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		token, err := svc.RefreshToken(ctx, session, req.RefreshToken, req.DomainID)
		if err != nil {
			return nil, err
		}

		return tokenRes{
			AccessToken:  token.GetAccessToken(),
			RefreshToken: token.GetRefreshToken(),
			AccessType:   token.GetAccessType(),
		}, nil
	}
}

func enableEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeUserStatusReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		user, err := svc.Enable(ctx, session, req.id)
		if err != nil {
			return nil, err
		}

		return changeUserStatusRes{User: user}, nil
	}
}

func disableEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeUserStatusReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		user, err := svc.Disable(ctx, session, req.id)
		if err != nil {
			return nil, err
		}

		return changeUserStatusRes{User: user}, nil
	}
}

func deleteEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeUserStatusReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		if err := svc.Delete(ctx, session, req.id); err != nil {
			return nil, err
		}

		return deleteUserRes{true}, nil
	}
}

func buildUsersResponse(cp users.MembersPage) usersPageRes {
	res := usersPageRes{
		pageRes: pageRes{
			Total:  cp.Total,
			Offset: cp.Offset,
			Limit:  cp.Limit,
		},
		Users: []viewUserRes{},
	}

	for _, user := range cp.Members {
		res.Users = append(res.Users, viewUserRes{User: user})
	}

	return res
}
