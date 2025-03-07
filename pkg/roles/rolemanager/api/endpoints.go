// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/roles"
	"github.com/go-kit/kit/endpoint"
)

func CreateRoleEndpoint(svc roles.RoleManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createRoleReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		rp, err := svc.AddRole(ctx, session, req.entityID, req.RoleName, req.OptionalActions, req.OptionalMembers)
		if err != nil {
			return nil, err
		}
		return createRoleRes{RoleProvision: rp}, nil
	}
}

func ListRolesEndpoint(svc roles.RoleManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listRolesReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		ros, err := svc.RetrieveAllRoles(ctx, session, req.entityID, req.limit, req.offset)
		if err != nil {
			return nil, err
		}
		return listRolesRes{RolePage: ros}, nil
	}
}

func ListEntityMembersEndpoint(svc roles.RoleManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listEntityMembersReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		pageQuery := roles.MembersRolePageQuery{
			Offset:           req.offset,
			Limit:            req.limit,
			AccessProviderID: req.accessProviderID,
			Order:            req.order,
			Dir:              req.dir,
			RoleID:           req.roleId,
			RoleName:         req.roleName,
			Actions:          req.actions,
			AccessType:       req.accessType,
		}

		mems, err := svc.ListEntityMembers(ctx, session, req.entityID, pageQuery)
		if err != nil {
			return nil, err
		}
		return listEntityMembersRes{mems}, nil
	}
}

func RemoveEntityMembersEndpoint(svc roles.RoleManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(removeEntityMembersReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		if err := svc.RemoveEntityMembers(ctx, session, req.entityID, req.MemberIDs); err != nil {
			return nil, err
		}
		return deleteEntityMembersRes{}, nil
	}
}

func ViewRoleEndpoint(svc roles.RoleManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewRoleReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		ro, err := svc.RetrieveRole(ctx, session, req.entityID, req.roleID)
		if err != nil {
			return nil, err
		}
		return viewRoleRes{Role: ro}, nil
	}
}

func UpdateRoleEndpoint(svc roles.RoleManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateRoleReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		ro, err := svc.UpdateRoleName(ctx, session, req.entityID, req.roleID, req.Name)
		if err != nil {
			return nil, err
		}
		return updateRoleRes{Role: ro}, nil
	}
}

func DeleteRoleEndpoint(svc roles.RoleManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteRoleReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		if err := svc.RemoveRole(ctx, session, req.entityID, req.roleID); err != nil {
			return nil, err
		}
		return deleteRoleRes{}, nil
	}
}

func ListAvailableActionsEndpoint(svc roles.RoleManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listAvailableActionsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		acts, err := svc.ListAvailableActions(ctx, session)
		if err != nil {
			return nil, err
		}
		return listAvailableActionsRes{acts}, nil
	}
}

func AddRoleActionsEndpoint(svc roles.RoleManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addRoleActionsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		caps, err := svc.RoleAddActions(ctx, session, req.entityID, req.roleID, req.Actions)
		if err != nil {
			return nil, err
		}
		return addRoleActionsRes{Actions: caps}, nil
	}
}

func ListRoleActionsEndpoint(svc roles.RoleManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listRoleActionsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		caps, err := svc.RoleListActions(ctx, session, req.entityID, req.roleID)
		if err != nil {
			return nil, err
		}
		return listRoleActionsRes{Actions: caps}, nil
	}
}

func DeleteRoleActionsEndpoint(svc roles.RoleManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteRoleActionsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		if err := svc.RoleRemoveActions(ctx, session, req.entityID, req.roleID, req.Actions); err != nil {
			return nil, err
		}
		return deleteRoleActionsRes{}, nil
	}
}

func DeleteAllRoleActionsEndpoint(svc roles.RoleManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteAllRoleActionsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		if err := svc.RoleRemoveAllActions(ctx, session, req.entityID, req.roleID); err != nil {
			return nil, err
		}
		return deleteAllRoleActionsRes{}, nil
	}
}

func AddRoleMembersEndpoint(svc roles.RoleManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addRoleMembersReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		members, err := svc.RoleAddMembers(ctx, session, req.entityID, req.roleID, req.Members)
		if err != nil {
			return nil, err
		}
		return addRoleMembersRes{members}, nil
	}
}

func ListRoleMembersEndpoint(svc roles.RoleManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listRoleMembersReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		mp, err := svc.RoleListMembers(ctx, session, req.entityID, req.roleID, req.limit, req.offset)
		if err != nil {
			return nil, err
		}
		return listRoleMembersRes{mp}, nil
	}
}

func DeleteRoleMembersEndpoint(svc roles.RoleManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteRoleMembersReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		if err := svc.RoleRemoveMembers(ctx, session, req.entityID, req.roleID, req.Members); err != nil {
			return nil, err
		}
		return deleteRoleMembersRes{}, nil
	}
}

func DeleteAllRoleMembersEndpoint(svc roles.RoleManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteAllRoleMembersReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		if err := svc.RoleRemoveAllMembers(ctx, session, req.entityID, req.roleID); err != nil {
			return nil, err
		}
		return deleteAllRoleMemberRes{}, nil
	}
}
