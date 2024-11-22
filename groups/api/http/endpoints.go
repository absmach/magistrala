// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/absmach/magistrala/groups"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/go-kit/kit/endpoint"
)

func CreateGroupEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createGroupReq)
		if err := req.validate(); err != nil {
			return createGroupRes{created: false}, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return createGroupRes{created: false}, svcerr.ErrAuthentication
		}

		group, err := svc.CreateGroup(ctx, session, req.Group)
		if err != nil {
			return createGroupRes{created: false}, err
		}

		return createGroupRes{created: true, Group: group}, nil
	}
}

func ViewGroupEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupReq)
		if err := req.validate(); err != nil {
			return viewGroupRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return viewGroupRes{}, svcerr.ErrAuthentication
		}

		group, err := svc.ViewGroup(ctx, session, req.id)
		if err != nil {
			return viewGroupRes{}, err
		}

		return viewGroupRes{Group: group}, nil
	}
}

func UpdateGroupEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateGroupReq)
		if err := req.validate(); err != nil {
			return updateGroupRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return updateGroupRes{}, svcerr.ErrAuthentication
		}

		group := groups.Group{
			ID:          req.id,
			Name:        req.Name,
			Description: req.Description,
			Metadata:    req.Metadata,
		}

		group, err := svc.UpdateGroup(ctx, session, group)
		if err != nil {
			return updateGroupRes{}, err
		}

		return updateGroupRes{Group: group}, nil
	}
}

func EnableGroupEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeGroupStatusReq)
		if err := req.validate(); err != nil {
			return changeStatusRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return changeStatusRes{}, svcerr.ErrAuthentication
		}

		group, err := svc.EnableGroup(ctx, session, req.id)
		if err != nil {
			return changeStatusRes{}, err
		}
		return changeStatusRes{Group: group}, nil
	}
}

func DisableGroupEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeGroupStatusReq)
		if err := req.validate(); err != nil {
			return changeStatusRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return changeStatusRes{}, svcerr.ErrAuthentication
		}

		group, err := svc.DisableGroup(ctx, session, req.id)
		if err != nil {
			return changeStatusRes{}, err
		}
		return changeStatusRes{Group: group}, nil
	}
}

func ListGroupsEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listGroupsReq)

		if err := req.validate(); err != nil {
			return groupPageRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return groupPageRes{}, svcerr.ErrAuthentication
		}

		var page groups.Page
		var err error
		switch {
		case req.userID != "":
			page, err = svc.ListUserGroups(ctx, session, req.userID, req.PageMeta)
		default:
			page, err = svc.ListGroups(ctx, session, req.PageMeta)
		}
		if err != nil {
			return groupPageRes{}, err
		}

		groups := []viewGroupRes{}
		for _, g := range page.Groups {
			groups = append(groups, toViewGroupRes(g))
		}

		return groupPageRes{
			pageRes: pageRes{
				Limit:  page.Limit,
				Offset: page.Offset,
				Total:  page.Total,
			},
			Groups: groups,
		}, nil
	}
}

func DeleteGroupEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupReq)
		if err := req.validate(); err != nil {
			return deleteGroupRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return deleteGroupRes{}, svcerr.ErrAuthentication
		}
		if err := svc.DeleteGroup(ctx, session, req.id); err != nil {
			return deleteGroupRes{}, err
		}
		return deleteGroupRes{deleted: true}, nil
	}
}

func retrieveGroupHierarchyEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(retrieveGroupHierarchyReq)
		if err := req.validate(); err != nil {
			return retrieveGroupHierarchyRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return changeStatusRes{}, svcerr.ErrAuthentication
		}

		hp, err := svc.RetrieveGroupHierarchy(ctx, session, req.id, req.HierarchyPageMeta)
		if err != nil {
			return retrieveGroupHierarchyRes{}, err
		}

		groups := []viewGroupRes{}
		for _, g := range hp.Groups {
			groups = append(groups, toViewGroupRes(g))
		}
		return retrieveGroupHierarchyRes{Level: hp.Level, Direction: hp.Direction, Groups: groups}, nil
	}
}

func addParentGroupEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addParentGroupReq)
		if err := req.validate(); err != nil {
			return addParentGroupRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return changeStatusRes{}, svcerr.ErrAuthentication
		}

		if err := svc.AddParentGroup(ctx, session, req.id, req.ParentID); err != nil {
			return addParentGroupRes{}, err
		}
		return addParentGroupRes{}, nil
	}
}

func removeParentGroupEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(removeParentGroupReq)
		if err := req.validate(); err != nil {
			return removeParentGroupRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return changeStatusRes{}, svcerr.ErrAuthentication
		}

		if err := svc.RemoveParentGroup(ctx, session, req.id); err != nil {
			return removeParentGroupRes{}, err
		}
		return removeParentGroupRes{}, nil
	}
}

func addChildrenGroupsEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addChildrenGroupsReq)
		if err := req.validate(); err != nil {
			return addChildrenGroupsRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return changeStatusRes{}, svcerr.ErrAuthentication
		}

		if err := svc.AddChildrenGroups(ctx, session, req.id, req.ChildrenIDs); err != nil {
			return addChildrenGroupsRes{}, err
		}
		return addChildrenGroupsRes{}, nil
	}
}

func removeChildrenGroupsEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(removeChildrenGroupsReq)
		if err := req.validate(); err != nil {
			return removeChildrenGroupsRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return changeStatusRes{}, svcerr.ErrAuthentication
		}

		if err := svc.RemoveChildrenGroups(ctx, session, req.id, req.ChildrenIDs); err != nil {
			return removeChildrenGroupsRes{}, err
		}
		return removeChildrenGroupsRes{}, nil
	}
}

func removeAllChildrenGroupsEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(removeAllChildrenGroupsReq)
		if err := req.validate(); err != nil {
			return removeAllChildrenGroupsRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return changeStatusRes{}, svcerr.ErrAuthentication
		}

		if err := svc.RemoveAllChildrenGroups(ctx, session, req.id); err != nil {
			return removeAllChildrenGroupsRes{}, err
		}
		return removeAllChildrenGroupsRes{}, nil
	}
}

func listChildrenGroupsEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listChildrenGroupsReq)
		if err := req.validate(); err != nil {
			return listChildrenGroupsRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return changeStatusRes{}, svcerr.ErrAuthentication
		}

		gp, err := svc.ListChildrenGroups(ctx, session, req.id, req.startLevel, req.endLevel, req.PageMeta)
		if err != nil {
			return listChildrenGroupsRes{}, err
		}
		viewGroups := []viewGroupRes{}

		for _, group := range gp.Groups {
			viewGroups = append(viewGroups, toViewGroupRes(group))
		}
		return listChildrenGroupsRes{
			pageRes: pageRes{
				Limit:  gp.Limit,
				Offset: gp.Offset,
				Total:  gp.Total,
			},
			Groups: viewGroups,
		}, nil
	}
}

func toViewGroupRes(group groups.Group) viewGroupRes {
	view := viewGroupRes{
		Group: group,
	}
	return view
}
