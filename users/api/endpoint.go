// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/users"
)

func registrationEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(userReq)
		if err := req.validate(); err != nil {
			return createUserRes{}, err
		}
		uid, err := svc.Register(ctx, req.user)
		if err != nil {
			return createUserRes{}, err
		}
		ucr := createUserRes{
			ID:      uid,
			created: true,
		}

		return ucr, nil
	}
}

// Password reset request endpoint.
// When successful password reset link is generated.
// Link is generated using MF_TOKEN_RESET_ENDPOINT env.
// and value from Referer header for host.
// {Referer}+{MF_TOKEN_RESET_ENDPOINT}+{token=TOKEN}
// http://mainflux.com/reset-request?token=xxxxxxxxxxx.
// Email with a link is being sent to the user.
// When user clicks on a link it should get the ui with form to
// enter new password, when form is submitted token and new password
// must be sent as PUT request to 'password/reset' passwordResetEndpoint
func passwordResetRequestEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(passwResetReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		res := passwChangeRes{}
		email := req.Email
		if err := svc.GenerateResetToken(ctx, email, req.Host); err != nil {
			return nil, err
		}
		res.Msg = MailSent

		return res, nil
	}
}

// This is endpoint that actually sets new password in password reset flow.
// When user clicks on a link in email finally ends on this endpoint as explained in
// the comment above.
func passwordResetEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(resetTokenReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		res := passwChangeRes{}
		if err := svc.ResetPassword(ctx, req.Token, req.Password); err != nil {
			return nil, err
		}
		res.Msg = ""
		return res, nil
	}
}

func viewUserEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewUserReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		u, err := svc.ViewUser(ctx, req.token, req.userID)
		if err != nil {
			return nil, err
		}
		return viewUserRes{
			ID:       u.ID,
			Email:    u.Email,
			Metadata: u.Metadata,
		}, nil
	}
}

func viewProfileEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewUserReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		u, err := svc.ViewProfile(ctx, req.token)
		if err != nil {
			return nil, err
		}
		return viewUserRes{
			ID:       u.ID,
			Email:    u.Email,
			Metadata: u.Metadata,
		}, nil
	}
}

func listUsersEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listUsersReq)
		if err := req.validate(); err != nil {
			return users.UserPage{}, err
		}
		up, err := svc.ListUsers(ctx, req.token, req.offset, req.limit, req.email, req.metadata)
		if err != nil {
			return users.UserPage{}, err
		}
		return buildUsersResponse(up), nil
	}
}

func updateUserEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateUserReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		user := users.User{
			Metadata: req.Metadata,
		}
		err := svc.UpdateUser(ctx, req.token, user)
		if err != nil {
			return nil, err
		}
		return updateUserRes{}, nil
	}
}

func passwordChangeEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(passwChangeReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		res := passwChangeRes{}
		if err := svc.ChangePassword(ctx, req.Token, req.Password, req.OldPassword); err != nil {
			return nil, err
		}
		return res, nil
	}
}

func loginEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(userReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		token, err := svc.Login(ctx, req.user)
		if err != nil {
			return nil, err
		}

		return tokenRes{token}, nil
	}
}

func createGroupEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		group := users.Group{
			Name:        req.Name,
			ParentID:    req.ParentID,
			Description: req.Description,
			Metadata:    req.Metadata,
		}
		saved, err := svc.CreateGroup(ctx, req.token, group)
		if err != nil {
			return nil, err
		}
		res := createGroupRes{
			ID:          saved.ID,
			Name:        saved.Name,
			Description: saved.Description,
			Metadata:    saved.Metadata,
			ParentID:    saved.ParentID,
			created:     true,
		}

		return res, nil
	}
}

func assignUserToGroup(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(userGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		if err := svc.Assign(ctx, req.token, req.userID, req.groupID); err != nil {
			return nil, err
		}
		return assignUserToGroupRes{}, nil
	}
}

func removeUserFromGroup(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(userGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		if err := svc.Unassign(ctx, req.token, req.userID, req.groupID); err != nil {
			return nil, err
		}
		return removeUserFromGroupRes{}, nil
	}
}

func listMembersEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listUserGroupReq)
		if err := req.validate(); err != nil {
			return users.UserPage{}, err
		}
		up, err := svc.ListMembers(ctx, req.token, req.groupID, req.offset, req.limit, req.metadata)
		if err != nil {
			return users.UserPage{}, err
		}
		return buildUsersResponse(up), nil
	}
}

func listMembershipsEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listUserGroupReq)
		if err := req.validate(); err != nil {
			return users.UserPage{}, err
		}
		gp, err := svc.ListMemberships(ctx, req.token, req.userID, req.offset, req.limit, req.metadata)
		if err != nil {
			return groupPageRes{}, err
		}
		return buildGroupsResponse(gp), nil
	}
}

func updateGroupEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateGroupReq)
		if err := req.validate(); err != nil {
			return updateGroupRes{}, err
		}

		group := users.Group{
			ID:          req.id,
			Name:        req.Name,
			Description: req.Description,
			Metadata:    req.Metadata,
		}

		if err := svc.UpdateGroup(ctx, req.token, group); err != nil {
			return updateGroupRes{}, err
		}

		return updateGroupRes{}, nil
	}
}

func viewGroupEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupReq)
		if err := req.validate(); err != nil {
			return viewGroupRes{}, err
		}
		group, err := svc.ViewGroup(ctx, req.token, req.groupID)
		if err != nil {
			return viewGroupRes{}, err
		}
		res := viewGroupRes{
			ID:          group.ID,
			Name:        group.Name,
			ParentID:    group.ParentID,
			OwnerID:     group.OwnerID,
			Description: group.Description,
			Metadata:    group.Metadata,
		}
		return res, nil
	}
}

func listGroupsEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listUserGroupReq)
		if err := req.validate(); err != nil {
			return groupPageRes{}, err
		}
		gp, err := svc.ListGroups(ctx, req.token, req.groupID, req.offset, req.limit, req.metadata)
		if err != nil {
			return groupPageRes{}, err
		}
		return buildGroupsResponse(gp), nil
	}
}

func deleteGroupEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		if err := svc.RemoveGroup(ctx, req.token, req.groupID); err != nil {
			return nil, err
		}
		return groupDeleteRes{}, nil
	}
}

func buildGroupsResponse(gp users.GroupPage) groupPageRes {
	res := groupPageRes{
		pageRes: pageRes{
			Total:  gp.Total,
			Offset: gp.Offset,
			Limit:  gp.Limit,
		},
		Groups: []viewGroupRes{},
	}
	for _, group := range gp.Groups {
		view := viewGroupRes{
			ID:          group.ID,
			ParentID:    group.ParentID,
			Name:        group.Name,
			Description: group.Description,
			Metadata:    group.Metadata,
		}
		res.Groups = append(res.Groups, view)
	}
	return res
}

func buildUsersResponse(up users.UserPage) userPageRes {
	res := userPageRes{
		pageRes: pageRes{
			Total:  up.Total,
			Offset: up.Offset,
			Limit:  up.Limit,
		},
		Users: []viewUserRes{},
	}
	for _, user := range up.Users {
		view := viewUserRes{
			ID:       user.ID,
			Email:    user.Email,
			Metadata: user.Metadata,
		}
		res.Users = append(res.Users, view)
	}
	return res
}
