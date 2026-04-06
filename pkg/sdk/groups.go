// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/roles"
)

const (
	groupsEndpoint   = "groups"
	childrenEndpoint = "children"
	MaxLevel         = uint64(5)
	MinLevel         = uint64(1)
)

// Group represents the group of Clients.
// Indicates a level in tree hierarchy. Root node is level 1.
// Path in a tree consisting of group IDs
// Paths are unique per owner.
type Group struct {
	ID                        string                    `json:"id,omitempty"`
	DomainID                  string                    `json:"domain_id,omitempty"`
	ParentID                  string                    `json:"parent_id,omitempty"`
	Name                      string                    `json:"name,omitempty"`
	Description               string                    `json:"description,omitempty"`
	Tags                      []string                  `json:"tags,omitempty"`
	Metadata                  Metadata                  `json:"metadata,omitempty"`
	Level                     int                       `json:"level,omitempty"`
	Path                      string                    `json:"path,omitempty"`
	Children                  []*Group                  `json:"children,omitempty"`
	CreatedAt                 time.Time                 `json:"created_at,omitempty"`
	UpdatedAt                 time.Time                 `json:"updated_at,omitempty"`
	UpdatedBy                 string                    `json:"updated_by,omitempty"`
	Status                    string                    `json:"status,omitempty"`
	RoleID                    string                    `json:"role_id,omitempty"`
	RoleName                  string                    `json:"role_name,omitempty"`
	Actions                   []string                  `json:"actions,omitempty"`
	AccessType                string                    `json:"access_type,omitempty"`
	AccessProviderId          string                    `json:"access_provider_id,omitempty"`
	AccessProviderRoleId      string                    `json:"access_provider_role_id,omitempty"`
	AccessProviderRoleName    string                    `json:"access_provider_role_name,omitempty"`
	AccessProviderRoleActions []string                  `json:"access_provider_role_actions,omitempty"`
	Roles                     []roles.MemberRoleActions `json:"roles,omitempty"`
}

func (sdk mgSDK) CreateGroup(ctx context.Context, g Group, domainID, token string) (Group, errors.SDKError) {
	data, err := json.Marshal(g)
	if err != nil {
		return Group{}, errors.NewSDKError(err)
	}
	url := fmt.Sprintf("%s/%s/%s", sdk.groupsURL, domainID, groupsEndpoint)

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodPost, url, token, data, nil, http.StatusCreated)
	if sdkErr != nil {
		return Group{}, sdkErr
	}

	g = Group{}
	if err := json.Unmarshal(body, &g); err != nil {
		return Group{}, errors.NewSDKError(err)
	}

	return g, nil
}

func (sdk mgSDK) Groups(ctx context.Context, pm PageMetadata, domainID, token string) (GroupsPage, errors.SDKError) {
	endpoint := fmt.Sprintf("%s/%s", domainID, groupsEndpoint)
	url, err := sdk.withQueryParams(sdk.groupsURL, endpoint, pm)
	if err != nil {
		return GroupsPage{}, errors.NewSDKError(err)
	}

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkErr != nil {
		return GroupsPage{}, sdkErr
	}

	gp := GroupsPage{}
	if err := json.Unmarshal(body, &gp); err != nil {
		return GroupsPage{}, errors.NewSDKError(err)
	}

	return gp, nil
}

func (sdk mgSDK) Group(ctx context.Context, id, domainID, token string) (Group, errors.SDKError) {
	if id == "" {
		return Group{}, errors.NewSDKError(apiutil.ErrMissingID)
	}

	url := fmt.Sprintf("%s/%s/%s/%s", sdk.groupsURL, domainID, groupsEndpoint, id)

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkErr != nil {
		return Group{}, sdkErr
	}

	var t Group
	if err := json.Unmarshal(body, &t); err != nil {
		return Group{}, errors.NewSDKError(err)
	}

	return t, nil
}

func (sdk mgSDK) UpdateGroup(ctx context.Context, g Group, domainID, token string) (Group, errors.SDKError) {
	data, err := json.Marshal(g)
	if err != nil {
		return Group{}, errors.NewSDKError(err)
	}

	if g.ID == "" {
		return Group{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.groupsURL, domainID, groupsEndpoint, g.ID)

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodPut, url, token, data, nil, http.StatusOK)
	if sdkErr != nil {
		return Group{}, sdkErr
	}

	g = Group{}
	if err := json.Unmarshal(body, &g); err != nil {
		return Group{}, errors.NewSDKError(err)
	}

	return g, nil
}

func (sdk mgSDK) UpdateGroupTags(ctx context.Context, g Group, domainID, token string) (Group, errors.SDKError) {
	if g.ID == "" {
		return Group{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s/tags", sdk.groupsURL, domainID, groupsEndpoint, g.ID)

	data, err := json.Marshal(g)
	if err != nil {
		return Group{}, errors.NewSDKError(err)
	}

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodPatch, url, token, data, nil, http.StatusOK)
	if sdkErr != nil {
		return Group{}, sdkErr
	}

	g = Group{}
	if err := json.Unmarshal(body, &g); err != nil {
		return Group{}, errors.NewSDKError(err)
	}

	return g, nil
}

func (sdk mgSDK) SetGroupParent(ctx context.Context, id, domainID, groupID, token string) errors.SDKError {
	scpg := groupParentReq{ParentID: groupID}
	data, err := json.Marshal(scpg)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.groupsURL, domainID, groupsEndpoint, id, parentEndpoint)
	_, _, sdkErr := sdk.processRequest(ctx, http.MethodPost, url, token, data, nil, http.StatusOK)

	return sdkErr
}

func (sdk mgSDK) RemoveGroupParent(ctx context.Context, id, domainID, groupID, token string) errors.SDKError {
	pgr := groupParentReq{ParentID: groupID}
	data, err := json.Marshal(pgr)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.groupsURL, domainID, groupsEndpoint, id, parentEndpoint)
	_, _, sdkErr := sdk.processRequest(ctx, http.MethodDelete, url, token, data, nil, http.StatusNoContent)

	return sdkErr
}

func (sdk mgSDK) AddChildren(ctx context.Context, id, domainID string, groupIDs []string, token string) errors.SDKError {
	acg := childrenGroupsReq{ChildrenIDs: groupIDs}
	data, err := json.Marshal(acg)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.groupsURL, domainID, groupsEndpoint, id, childrenEndpoint)
	_, _, sdkErr := sdk.processRequest(ctx, http.MethodPost, url, token, data, nil, http.StatusOK)

	return sdkErr
}

func (sdk mgSDK) RemoveChildren(ctx context.Context, id, domainID string, groupIDs []string, token string) errors.SDKError {
	rcg := childrenGroupsReq{ChildrenIDs: groupIDs}
	data, err := json.Marshal(rcg)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.groupsURL, domainID, groupsEndpoint, id, childrenEndpoint)
	_, _, sdkErr := sdk.processRequest(ctx, http.MethodDelete, url, token, data, nil, http.StatusNoContent)

	return sdkErr
}

func (sdk mgSDK) RemoveAllChildren(ctx context.Context, id, domainID, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s", sdk.groupsURL, domainID, groupsEndpoint, id, childrenEndpoint, "all")
	_, _, sdkErr := sdk.processRequest(ctx, http.MethodDelete, url, token, nil, nil, http.StatusNoContent)

	return sdkErr
}

func (sdk mgSDK) Children(ctx context.Context, id, domainID string, pm PageMetadata, token string) (GroupsPage, errors.SDKError) {
	endpoint := fmt.Sprintf("%s/%s/%s/%s", domainID, groupsEndpoint, id, childrenEndpoint)
	url, err := sdk.withQueryParams(sdk.groupsURL, endpoint, pm)
	if err != nil {
		return GroupsPage{}, errors.NewSDKError(err)
	}

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkErr != nil {
		return GroupsPage{}, sdkErr
	}

	gp := GroupsPage{}
	if err := json.Unmarshal(body, &gp); err != nil {
		return GroupsPage{}, errors.NewSDKError(err)
	}

	return gp, nil
}

func (sdk mgSDK) EnableGroup(ctx context.Context, id, domainID, token string) (Group, errors.SDKError) {
	return sdk.changeGroupStatus(ctx, id, enableEndpoint, domainID, token)
}

func (sdk mgSDK) DisableGroup(ctx context.Context, id, domainID, token string) (Group, errors.SDKError) {
	return sdk.changeGroupStatus(ctx, id, disableEndpoint, domainID, token)
}

func (sdk mgSDK) changeGroupStatus(ctx context.Context, id, status, domainID, token string) (Group, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.groupsURL, domainID, groupsEndpoint, id, status)

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodPost, url, token, nil, nil, http.StatusOK)
	if sdkErr != nil {
		return Group{}, sdkErr
	}
	g := Group{}
	if err := json.Unmarshal(body, &g); err != nil {
		return Group{}, errors.NewSDKError(err)
	}

	return g, nil
}

func (sdk mgSDK) DeleteGroup(ctx context.Context, id, domainID, token string) errors.SDKError {
	if id == "" {
		return errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.groupsURL, domainID, groupsEndpoint, id)
	_, _, sdkErr := sdk.processRequest(ctx, http.MethodDelete, url, token, nil, nil, http.StatusNoContent)
	return sdkErr
}

func (sdk mgSDK) Hierarchy(ctx context.Context, id, domainID string, pm PageMetadata, token string) (GroupsHierarchyPage, errors.SDKError) {
	endpoint := fmt.Sprintf("%s/%s/%s/hierarchy", domainID, groupsEndpoint, id)
	url, err := sdk.withQueryParams(sdk.groupsURL, endpoint, pm)
	if err != nil {
		return GroupsHierarchyPage{}, errors.NewSDKError(err)
	}

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkErr != nil {
		return GroupsHierarchyPage{}, sdkErr
	}

	hp := GroupsHierarchyPage{}
	if err := json.Unmarshal(body, &hp); err != nil {
		return GroupsHierarchyPage{}, errors.NewSDKError(err)
	}

	return hp, nil
}

func (sdk mgSDK) CreateGroupRole(ctx context.Context, id, domainID string, rq RoleReq, token string) (Role, errors.SDKError) {
	return sdk.createRole(ctx, sdk.groupsURL, groupsEndpoint, id, domainID, rq, token)
}

func (sdk mgSDK) GroupRoles(ctx context.Context, id, domainID string, pm PageMetadata, token string) (RolesPage, errors.SDKError) {
	return sdk.listRoles(ctx, sdk.groupsURL, groupsEndpoint, id, domainID, pm, token)
}

func (sdk mgSDK) GroupRole(ctx context.Context, id, roleID, domainID, token string) (Role, errors.SDKError) {
	return sdk.viewRole(ctx, sdk.groupsURL, groupsEndpoint, id, roleID, domainID, token)
}

func (sdk mgSDK) UpdateGroupRole(ctx context.Context, id, roleID, newName, domainID string, token string) (Role, errors.SDKError) {
	return sdk.updateRole(ctx, sdk.groupsURL, groupsEndpoint, id, roleID, newName, domainID, token)
}

func (sdk mgSDK) DeleteGroupRole(ctx context.Context, id, roleID, domainID, token string) errors.SDKError {
	return sdk.deleteRole(ctx, sdk.groupsURL, groupsEndpoint, id, roleID, domainID, token)
}

func (sdk mgSDK) AddGroupRoleActions(ctx context.Context, id, roleID, domainID string, actions []string, token string) ([]string, errors.SDKError) {
	return sdk.addRoleActions(ctx, sdk.groupsURL, groupsEndpoint, id, roleID, domainID, actions, token)
}

func (sdk mgSDK) GroupRoleActions(ctx context.Context, id, roleID, domainID string, token string) ([]string, errors.SDKError) {
	return sdk.listRoleActions(ctx, sdk.groupsURL, groupsEndpoint, id, roleID, domainID, token)
}

func (sdk mgSDK) RemoveGroupRoleActions(ctx context.Context, id, roleID, domainID string, actions []string, token string) errors.SDKError {
	return sdk.removeRoleActions(ctx, sdk.groupsURL, groupsEndpoint, id, roleID, domainID, actions, token)
}

func (sdk mgSDK) RemoveAllGroupRoleActions(ctx context.Context, id, roleID, domainID, token string) errors.SDKError {
	return sdk.removeAllRoleActions(ctx, sdk.groupsURL, groupsEndpoint, id, roleID, domainID, token)
}

func (sdk mgSDK) AddGroupRoleMembers(ctx context.Context, id, roleID, domainID string, members []string, token string) ([]string, errors.SDKError) {
	return sdk.addRoleMembers(ctx, sdk.groupsURL, groupsEndpoint, id, roleID, domainID, members, token)
}

func (sdk mgSDK) GroupRoleMembers(ctx context.Context, id, roleID, domainID string, pm PageMetadata, token string) (RoleMembersPage, errors.SDKError) {
	return sdk.listRoleMembers(ctx, sdk.groupsURL, groupsEndpoint, id, roleID, domainID, pm, token)
}

func (sdk mgSDK) RemoveGroupRoleMembers(ctx context.Context, id, roleID, domainID string, members []string, token string) errors.SDKError {
	return sdk.removeRoleMembers(ctx, sdk.groupsURL, groupsEndpoint, id, roleID, domainID, members, token)
}

func (sdk mgSDK) RemoveAllGroupRoleMembers(ctx context.Context, id, roleID, domainID, token string) errors.SDKError {
	return sdk.removeAllRoleMembers(ctx, sdk.groupsURL, groupsEndpoint, id, roleID, domainID, token)
}

func (sdk mgSDK) AvailableGroupRoleActions(ctx context.Context, domainID, token string) ([]string, errors.SDKError) {
	return sdk.listAvailableRoleActions(ctx, sdk.groupsURL, groupsEndpoint, domainID, token)
}

func (sdk mgSDK) ListGroupMembers(ctx context.Context, groupID, domainID string, pm PageMetadata, token string) (EntityMembersPage, errors.SDKError) {
	return sdk.listEntityMembers(ctx, sdk.groupsURL, domainID, groupsEndpoint, groupID, token, pm)
}
