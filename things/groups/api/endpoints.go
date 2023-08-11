// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/pkg/errors"
	mfgroups "github.com/mainflux/mainflux/pkg/groups"
	"github.com/mainflux/mainflux/things/groups"
)

func createGroupEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createGroupReq)
		if err := req.validate(); err != nil {
			return createGroupRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		group, err := svc.CreateGroups(ctx, req.token, req.Group)
		if err != nil {
			return createGroupRes{}, err
		}

		return createGroupRes{created: true, Group: group[0]}, nil
	}
}

func createGroupsEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createGroupsReq)
		if err := req.validate(); err != nil {
			return groupPageRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		gs, err := svc.CreateGroups(ctx, req.token, req.Groups...)
		if err != nil {
			return groupPageRes{}, err
		}
		return buildGroupsResponse(mfgroups.GroupsPage{Groups: gs}), nil
	}
}

func viewGroupEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupReq)
		if err := req.validate(); err != nil {
			return viewGroupRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		group, err := svc.ViewGroup(ctx, req.token, req.id)
		if err != nil {
			return viewGroupRes{}, err
		}

		return viewGroupRes{Group: group}, nil
	}
}

func updateGroupEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateGroupReq)
		if err := req.validate(); err != nil {
			return updateGroupRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		group := mfgroups.Group{
			ID:          req.id,
			Name:        req.Name,
			Description: req.Description,
			Metadata:    req.Metadata,
		}

		group, err := svc.UpdateGroup(ctx, req.token, group)
		if err != nil {
			return updateGroupRes{}, err
		}

		return updateGroupRes{Group: group}, nil
	}
}

func enableGroupEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeGroupStatusReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		group, err := svc.EnableGroup(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}
		return changeStatusRes{Group: group}, nil
	}
}

func disableGroupEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeGroupStatusReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		group, err := svc.DisableGroup(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}
		return changeStatusRes{Group: group}, nil
	}
}

func listGroupsEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listGroupsReq)
		if err := req.validate(); err != nil {
			return groupPageRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}
		page, err := svc.ListGroups(ctx, req.token, req.GroupsPage)
		if err != nil {
			return groupPageRes{}, err
		}

		if req.tree {
			return buildGroupsResponseTree(page), nil
		}

		return buildGroupsResponse(page), nil
	}
}

func listMembershipsEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMembershipReq)
		if err := req.validate(); err != nil {
			return membershipPageRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		page, err := svc.ListMemberships(ctx, req.token, req.clientID, req.GroupsPage)
		if err != nil {
			return membershipPageRes{}, err
		}

		res := membershipPageRes{
			pageRes: pageRes{
				Total:  page.Total,
				Offset: page.Offset,
				Limit:  page.Limit,
			},
			Memberships: []viewMembershipRes{},
		}
		for _, g := range page.Memberships {
			res.Memberships = append(res.Memberships, viewMembershipRes{Group: g})
		}

		return res, nil
	}
}

func buildGroupsResponseTree(page mfgroups.GroupsPage) groupPageRes {
	groupsMap := map[string]*mfgroups.Group{}
	// Parents' map keeps its array of children.
	parentsMap := map[string][]*mfgroups.Group{}
	for i := range page.Groups {
		if _, ok := groupsMap[page.Groups[i].ID]; !ok {
			groupsMap[page.Groups[i].ID] = &page.Groups[i]
			parentsMap[page.Groups[i].ID] = make([]*mfgroups.Group, 0)
		}
	}

	for _, group := range groupsMap {
		if children, ok := parentsMap[group.Parent]; ok {
			children = append(children, group)
			parentsMap[group.Parent] = children
		}
	}

	res := groupPageRes{
		pageRes: pageRes{
			Limit:  page.Limit,
			Offset: page.Offset,
			Total:  page.Total,
		},
		Groups: []viewGroupRes{},
	}

	for _, group := range groupsMap {
		if children, ok := parentsMap[group.ID]; ok {
			group.Children = children
		}
	}

	for _, group := range groupsMap {
		view := toViewGroupRes(*group)
		if children, ok := parentsMap[group.Parent]; len(children) == 0 || !ok {
			res.Groups = append(res.Groups, view)
		}
	}

	return res
}

func toViewGroupRes(group mfgroups.Group) viewGroupRes {
	view := viewGroupRes{
		Group: group,
	}
	return view
}

func buildGroupsResponse(gp mfgroups.GroupsPage) groupPageRes {
	res := groupPageRes{
		pageRes: pageRes{
			Total: gp.Total,
		},
		Groups: []viewGroupRes{},
	}

	for _, group := range gp.Groups {
		view := viewGroupRes{
			Group: group,
		}
		res.Groups = append(res.Groups, view)
	}

	return res
}
