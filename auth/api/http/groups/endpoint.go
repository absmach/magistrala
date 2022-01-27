package groups

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/auth"
)

func createGroupEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createGroupReq)
		if err := req.validate(); err != nil {
			return groupRes{}, err
		}

		group := auth.Group{
			Name:        req.Name,
			Description: req.Description,
			ParentID:    req.ParentID,
			Metadata:    req.Metadata,
		}

		group, err := svc.CreateGroup(ctx, req.token, group)
		if err != nil {
			return groupRes{}, err
		}

		return groupRes{created: true, id: group.ID}, nil
	}
}

func viewGroupEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupReq)
		if err := req.validate(); err != nil {
			return viewGroupRes{}, err
		}

		group, err := svc.ViewGroup(ctx, req.token, req.id)
		if err != nil {
			return viewGroupRes{}, err
		}

		res := viewGroupRes{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			Metadata:    group.Metadata,
			ParentID:    group.ParentID,
			OwnerID:     group.OwnerID,
			CreatedAt:   group.CreatedAt,
			UpdatedAt:   group.UpdatedAt,
		}

		return res, nil
	}
}

func updateGroupEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateGroupReq)
		if err := req.validate(); err != nil {
			return groupRes{}, err
		}

		group := auth.Group{
			ID:          req.id,
			Name:        req.Name,
			Description: req.Description,
			Metadata:    req.Metadata,
		}

		_, err := svc.UpdateGroup(ctx, req.token, group)
		if err != nil {
			return groupRes{}, err
		}

		res := groupRes{created: false}
		return res, nil
	}
}

func deleteGroupEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveGroup(ctx, req.token, req.id); err != nil {
			return nil, err
		}

		return deleteRes{}, nil
	}
}

func listGroupsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listGroupsReq)
		if err := req.validate(); err != nil {
			return groupPageRes{}, err
		}
		pm := auth.PageMetadata{
			Level:    req.level,
			Metadata: req.metadata,
		}
		page, err := svc.ListGroups(ctx, req.token, pm)
		if err != nil {
			return groupPageRes{}, err
		}

		if req.tree {
			return buildGroupsResponseTree(page), nil
		}

		return buildGroupsResponse(page), nil
	}
}

func listMemberships(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMembershipsReq)
		if err := req.validate(); err != nil {
			return memberPageRes{}, err
		}

		pm := auth.PageMetadata{
			Offset:   req.offset,
			Limit:    req.limit,
			Metadata: req.metadata,
		}

		page, err := svc.ListMemberships(ctx, req.token, req.id, pm)
		if err != nil {
			return memberPageRes{}, err
		}

		return buildGroupsResponse(page), nil
	}
}

func shareGroupAccessEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(shareGroupAccessReq)
		if err := req.validate(); err != nil {
			return shareGroupRes{}, err
		}

		if err := svc.AssignGroupAccessRights(ctx, req.token, req.ThingGroupID, req.userGroupID); err != nil {
			return shareGroupRes{}, err
		}
		return shareGroupRes{}, nil
	}
}

func listChildrenEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listGroupsReq)
		if err := req.validate(); err != nil {
			return groupPageRes{}, err
		}

		pm := auth.PageMetadata{
			Level:    req.level,
			Metadata: req.metadata,
		}
		page, err := svc.ListChildren(ctx, req.token, req.id, pm)
		if err != nil {
			return groupPageRes{}, err
		}

		if req.tree {
			return buildGroupsResponseTree(page), nil
		}

		return buildGroupsResponse(page), nil
	}
}

func listParentsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listGroupsReq)
		if err := req.validate(); err != nil {
			return groupPageRes{}, err
		}
		pm := auth.PageMetadata{
			Level:    req.level,
			Metadata: req.metadata,
		}

		page, err := svc.ListParents(ctx, req.token, req.id, pm)
		if err != nil {
			return groupPageRes{}, err
		}

		if req.tree {
			return buildGroupsResponseTree(page), nil
		}

		return buildGroupsResponse(page), nil
	}
}

func assignEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(assignReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.Assign(ctx, req.token, req.groupID, req.Type, req.Members...); err != nil {
			return nil, err
		}

		return assignRes{}, nil
	}
}

func unassignEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(unassignReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.Unassign(ctx, req.token, req.groupID, req.Members...); err != nil {
			return nil, err
		}

		return unassignRes{}, nil
	}
}

func listMembersEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMembersReq)
		if err := req.validate(); err != nil {
			return memberPageRes{}, err
		}

		pm := auth.PageMetadata{
			Offset:   req.offset,
			Limit:    req.limit,
			Metadata: req.metadata,
		}
		page, err := svc.ListMembers(ctx, req.token, req.id, req.groupType, pm)
		if err != nil {
			return memberPageRes{}, err
		}

		return buildUsersResponse(page, req.groupType), nil
	}
}

func buildGroupsResponseTree(page auth.GroupPage) groupPageRes {
	groupsMap := map[string]*auth.Group{}
	// Parents' map keeps its array of children.
	parentsMap := map[string][]*auth.Group{}
	for i := range page.Groups {
		if _, ok := groupsMap[page.Groups[i].ID]; !ok {
			groupsMap[page.Groups[i].ID] = &page.Groups[i]
			parentsMap[page.Groups[i].ID] = make([]*auth.Group, 0)
		}
	}

	for _, group := range groupsMap {
		if children, ok := parentsMap[group.ParentID]; ok {
			children = append(children, group)
			parentsMap[group.ParentID] = children
		}
	}

	res := groupPageRes{
		pageRes: pageRes{
			Limit:  page.Limit,
			Offset: page.Offset,
			Total:  page.Total,
			Level:  page.Level,
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

func toViewGroupRes(group auth.Group) viewGroupRes {
	view := viewGroupRes{
		ID:          group.ID,
		ParentID:    group.ParentID,
		OwnerID:     group.OwnerID,
		Name:        group.Name,
		Description: group.Description,
		Metadata:    group.Metadata,
		Level:       group.Level,
		Path:        group.Path,
		Children:    make([]*viewGroupRes, 0),
		CreatedAt:   group.CreatedAt,
		UpdatedAt:   group.UpdatedAt,
	}

	for _, ch := range group.Children {
		child := toViewGroupRes(*ch)
		view.Children = append(view.Children, &child)
	}

	return view
}

func buildGroupsResponse(gp auth.GroupPage) groupPageRes {
	res := groupPageRes{
		pageRes: pageRes{
			Total: gp.Total,
			Level: gp.Level,
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

func buildUsersResponse(mp auth.MemberPage, groupType string) memberPageRes {
	res := memberPageRes{
		pageRes: pageRes{
			Total:  mp.Total,
			Offset: mp.Offset,
			Limit:  mp.Limit,
			Name:   mp.Name,
		},
		Type:    groupType,
		Members: []string{},
	}

	for _, m := range mp.Members {
		res.Members = append(res.Members, m.ID)
	}

	return res
}
