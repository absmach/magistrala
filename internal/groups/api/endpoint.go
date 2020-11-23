package groups

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/internal/groups"
	"github.com/mainflux/mainflux/pkg/errors"
)

func CreateGroupEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createGroupReq)
		if err := req.validate(); err != nil {
			return groupRes{}, err
		}

		group := groups.Group{
			Name:        req.Name,
			Description: req.Description,
			ParentID:    req.ParentID,
			Metadata:    req.Metadata,
		}

		id, err := svc.CreateGroup(ctx, req.token, group)
		if err != nil {
			return groupRes{}, errors.Wrap(groups.ErrCreateGroup, err)
		}

		return groupRes{created: true, id: id}, nil
	}
}

func ViewGroupEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupReq)
		if err := req.validate(); err != nil {
			return viewGroupRes{}, errors.Wrap(groups.ErrMalformedEntity, err)
		}

		group, err := svc.ViewGroup(ctx, req.token, req.groupID)
		if err != nil {
			return viewGroupRes{}, errors.Wrap(groups.ErrFetchGroups, err)
		}

		res := viewGroupRes{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			Metadata:    group.Metadata,
			ParentID:    group.ParentID,
			OwnerID:     group.OwnerID,
		}

		return res, nil
	}
}

func UpdateGroupEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateGroupReq)
		if err := req.validate(); err != nil {
			return groupRes{}, errors.Wrap(groups.ErrMalformedEntity, err)
		}

		group := groups.Group{
			ID:          req.id,
			Name:        req.Name,
			Description: req.Description,
			ParentID:    req.ParentID,
			Metadata:    req.Metadata,
		}

		_, err := svc.UpdateGroup(ctx, req.token, group)
		if err != nil {
			return groupRes{}, errors.Wrap(groups.ErrUpdateGroup, err)
		}

		res := groupRes{created: false}
		return res, nil
	}
}

func DeleteGroupEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(groups.ErrMalformedEntity, err)
		}

		if err := svc.RemoveGroup(ctx, req.token, req.groupID); err != nil {
			return nil, errors.Wrap(groups.ErrDeleteGroup, err)
		}

		return groupDeleteRes{}, nil
	}
}

func ListGroupsEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listGroupsReq)
		if err := req.validate(); err != nil {
			return groupPageRes{}, errors.Wrap(groups.ErrMalformedEntity, err)
		}

		page, err := svc.ListGroups(ctx, req.token, req.level, req.metadata)
		if err != nil {
			return groupPageRes{}, errors.Wrap(groups.ErrFetchGroups, err)
		}

		if req.tree {
			return buildGroupsResponseTree(page), nil
		}

		return buildGroupsResponse(page), nil
	}
}

func ListMembership(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMemberGroupReq)
		if err := req.validate(); err != nil {
			return memberPageRes{}, err
		}

		page, err := svc.ListMemberships(ctx, req.token, req.memberID, req.offset, req.limit, req.metadata)
		if err != nil {
			return memberPageRes{}, err
		}

		if req.tree {
			return buildGroupsResponseTree(page), nil
		}

		return buildGroupsResponse(page), nil
	}
}

func ListGroupChildrenEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listGroupsReq)
		if err := req.validate(); err != nil {
			return groupPageRes{}, errors.Wrap(groups.ErrMalformedEntity, err)
		}

		page, err := svc.ListChildren(ctx, req.token, req.groupID, req.level, req.metadata)
		if err != nil {
			return groupPageRes{}, errors.Wrap(groups.ErrFetchGroups, err)
		}

		if req.tree {
			return buildGroupsResponseTree(page), nil
		}

		return buildGroupsResponse(page), nil
	}
}

func ListGroupParentsEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listGroupsReq)
		if err := req.validate(); err != nil {
			return groupPageRes{}, errors.Wrap(groups.ErrMalformedEntity, err)
		}

		page, err := svc.ListParents(ctx, req.token, req.groupID, req.level, req.metadata)
		if err != nil {
			return groupPageRes{}, errors.Wrap(groups.ErrFetchGroups, err)
		}

		if req.tree {
			return buildGroupsResponseTree(page), nil
		}

		return buildGroupsResponse(page), nil
	}
}

func AssignEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(memberGroupReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(groups.ErrMalformedEntity, err)
		}

		if err := svc.Assign(ctx, req.token, req.memberID, req.groupID); err != nil {
			return nil, errors.Wrap(groups.ErrAssignToGroup, err)
		}

		return assignMemberToGroupRes{}, nil
	}
}

func UnassignEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(memberGroupReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(groups.ErrMalformedEntity, err)
		}

		if err := svc.Unassign(ctx, req.token, req.memberID, req.groupID); err != nil {
			return nil, errors.Wrap(groups.ErrUnassignFromGroup, err)
		}

		return removeMemberFromGroupRes{}, nil
	}
}

func ListMembersEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMemberGroupReq)
		if err := req.validate(); err != nil {
			return memberPageRes{}, errors.Wrap(groups.ErrMalformedEntity, err)
		}

		page, err := svc.ListMembers(ctx, req.token, req.groupID, req.offset, req.limit, req.metadata)
		if err != nil {
			return memberPageRes{}, err
		}

		return buildUsersResponse(page), nil
	}
}

func buildGroupsResponseTree(page groups.GroupPage) groupPageRes {
	groupsMap := map[string]*groups.Group{}
	// Parents map keeps its array of children.
	parentsMap := map[string][]*groups.Group{}
	for i := range page.Groups {
		if _, ok := groupsMap[page.Groups[i].ID]; !ok {
			groupsMap[page.Groups[i].ID] = &page.Groups[i]
			parentsMap[page.Groups[i].ID] = make([]*groups.Group, 0)
		}
	}

	for _, group := range groupsMap {
		if ch, ok := parentsMap[group.ParentID]; ok {
			ch = append(ch, group)
			parentsMap[group.ParentID] = ch
		}
	}

	res := groupPageRes{
		pageRes: pageRes{
			Total:  page.Total,
			Offset: page.Offset,
			Limit:  page.Limit,
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
		if children, ok := parentsMap[group.ParentID]; len(children) == 0 || !ok {
			res.Groups = append(res.Groups, view)
		}
	}

	return res
}

func toViewGroupRes(g groups.Group) viewGroupRes {
	view := viewGroupRes{
		ID:          g.ID,
		ParentID:    g.ParentID,
		OwnerID:     g.OwnerID,
		Name:        g.Name,
		Description: g.Description,
		Metadata:    g.Metadata,
		Level:       g.Level,
		Path:        g.Path,
		Children:    make([]*viewGroupRes, 0),
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
	}

	for _, ch := range g.Children {
		child := toViewGroupRes(*ch)
		view.Children = append(view.Children, &child)
	}

	return view
}

func buildGroupsResponse(gp groups.GroupPage) groupPageRes {
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
			OwnerID:     group.OwnerID,
			Name:        group.Name,
			Description: group.Description,
			Metadata:    group.Metadata,
			Level:       group.Level,
			Path:        group.Path,
			CreatedAt:   group.CreatedAt,
			UpdatedAt:   group.UpdatedAt,
		}
		res.Groups = append(res.Groups, view)
	}

	return res
}

func buildUsersResponse(mp groups.MemberPage) memberPageRes {
	res := memberPageRes{
		pageRes: pageRes{
			Total:  mp.Total,
			Offset: mp.Offset,
			Limit:  mp.Limit,
			Name:   mp.Name,
		},
		Members: []interface{}{},
	}

	for _, m := range mp.Members {
		res.Members = append(res.Members, m)
	}

	return res
}
