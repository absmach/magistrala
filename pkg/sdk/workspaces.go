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
)

const (
	workspacesEndpoint = "workspaces"
	freezeEndpoint     = "freeze"
)

// Workspace represents magistrala workspace.
type Workspace struct {
	ID          string    `json:"id,omitempty"`
	Name        string    `json:"name,omitempty"`
	Metadata    Metadata  `json:"metadata,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	Route       string    `json:"route,omitempty"`
	Status      string    `json:"status,omitempty"`
	Permission  string    `json:"permission,omitempty"`
	CreatedBy   string    `json:"created_by,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedBy   string    `json:"updated_by,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
	Permissions []string  `json:"permissions,omitempty"`
}

func (sdk mgSDK) CreateWorkspace(ctx context.Context, workspace Workspace, token string) (Workspace, errors.SDKError) {
	data, err := json.Marshal(workspace)
	if err != nil {
		return Workspace{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s", sdk.workspacesURL, workspacesEndpoint)

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodPost, url, token, data, nil, http.StatusCreated)
	if sdkErr != nil {
		return Workspace{}, sdkErr
	}

	var d Workspace
	if err := json.Unmarshal(body, &d); err != nil {
		return Workspace{}, errors.NewSDKError(err)
	}
	return d, nil
}

func (sdk mgSDK) Workspaces(ctx context.Context, pm PageMetadata, token string) (WorkspacesPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.workspacesURL, workspacesEndpoint, pm)
	if err != nil {
		return WorkspacesPage{}, errors.NewSDKError(err)
	}

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkErr != nil {
		return WorkspacesPage{}, sdkErr
	}

	var dp WorkspacesPage
	if err := json.Unmarshal(body, &dp); err != nil {
		return WorkspacesPage{}, errors.NewSDKError(err)
	}

	return dp, nil
}

func (sdk mgSDK) Workspace(ctx context.Context, workspaceID, token string) (Workspace, errors.SDKError) {
	if workspaceID == "" {
		return Workspace{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s", sdk.workspacesURL, workspacesEndpoint, workspaceID)

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkErr != nil {
		return Workspace{}, sdkErr
	}

	var workspace Workspace
	if err := json.Unmarshal(body, &workspace); err != nil {
		return Workspace{}, errors.NewSDKError(err)
	}

	return workspace, nil
}

func (sdk mgSDK) UpdateWorkspace(ctx context.Context, workspace Workspace, token string) (Workspace, errors.SDKError) {
	if workspace.ID == "" {
		return Workspace{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s", sdk.workspacesURL, workspacesEndpoint, workspace.ID)

	data, err := json.Marshal(workspace)
	if err != nil {
		return Workspace{}, errors.NewSDKError(err)
	}

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodPatch, url, token, data, nil, http.StatusOK)
	if sdkErr != nil {
		return Workspace{}, sdkErr
	}

	var d Workspace
	if err := json.Unmarshal(body, &d); err != nil {
		return Workspace{}, errors.NewSDKError(err)
	}
	return d, nil
}

func (sdk mgSDK) EnableWorkspace(ctx context.Context, workspaceID, token string) errors.SDKError {
	return sdk.changeWorkspaceStatus(ctx, token, workspaceID, enableEndpoint)
}

func (sdk mgSDK) DisableWorkspace(ctx context.Context, workspaceID, token string) errors.SDKError {
	return sdk.changeWorkspaceStatus(ctx, token, workspaceID, disableEndpoint)
}

func (sdk mgSDK) FreezeWorkspace(ctx context.Context, workspaceID, token string) errors.SDKError {
	return sdk.changeWorkspaceStatus(ctx, token, workspaceID, freezeEndpoint)
}

func (sdk mgSDK) changeWorkspaceStatus(ctx context.Context, token, id, status string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.workspacesURL, workspacesEndpoint, id, status)
	_, _, sdkErr := sdk.processRequest(ctx, http.MethodPost, url, token, nil, nil, http.StatusOK)
	return sdkErr
}

func (sdk mgSDK) CreateWorkspaceRole(ctx context.Context, id string, rq RoleReq, token string) (Role, errors.SDKError) {
	return sdk.createRole(ctx, sdk.workspacesURL, workspacesEndpoint, id, "", rq, token)
}

func (sdk mgSDK) WorkspaceRoles(ctx context.Context, id string, pm PageMetadata, token string) (RolesPage, errors.SDKError) {
	return sdk.listRoles(ctx, sdk.workspacesURL, workspacesEndpoint, id, "", pm, token)
}

func (sdk mgSDK) WorkspaceRole(ctx context.Context, id, roleID, token string) (Role, errors.SDKError) {
	return sdk.viewRole(ctx, sdk.workspacesURL, workspacesEndpoint, id, roleID, "", token)
}

func (sdk mgSDK) UpdateWorkspaceRole(ctx context.Context, id, roleID, newName string, token string) (Role, errors.SDKError) {
	return sdk.updateRole(ctx, sdk.workspacesURL, workspacesEndpoint, id, roleID, newName, "", token)
}

func (sdk mgSDK) DeleteWorkspaceRole(ctx context.Context, id, roleID, token string) errors.SDKError {
	return sdk.deleteRole(ctx, sdk.workspacesURL, workspacesEndpoint, id, roleID, "", token)
}

func (sdk mgSDK) AddWorkspaceRoleActions(ctx context.Context, id, roleID string, actions []string, token string) ([]string, errors.SDKError) {
	return sdk.addRoleActions(ctx, sdk.workspacesURL, workspacesEndpoint, id, roleID, "", actions, token)
}

func (sdk mgSDK) WorkspaceRoleActions(ctx context.Context, id, roleID string, token string) ([]string, errors.SDKError) {
	return sdk.listRoleActions(ctx, sdk.workspacesURL, workspacesEndpoint, id, roleID, "", token)
}

func (sdk mgSDK) RemoveWorkspaceRoleActions(ctx context.Context, id, roleID string, actions []string, token string) errors.SDKError {
	return sdk.removeRoleActions(ctx, sdk.workspacesURL, workspacesEndpoint, id, roleID, "", actions, token)
}

func (sdk mgSDK) RemoveAllWorkspaceRoleActions(ctx context.Context, id, roleID, token string) errors.SDKError {
	return sdk.removeAllRoleActions(ctx, sdk.workspacesURL, workspacesEndpoint, id, roleID, "", token)
}

func (sdk mgSDK) AddWorkspaceRoleMembers(ctx context.Context, id, roleID string, members []string, token string) ([]string, errors.SDKError) {
	return sdk.addRoleMembers(ctx, sdk.workspacesURL, workspacesEndpoint, id, roleID, "", members, token)
}

func (sdk mgSDK) WorkspaceRoleMembers(ctx context.Context, id, roleID string, pm PageMetadata, token string) (RoleMembersPage, errors.SDKError) {
	return sdk.listRoleMembers(ctx, sdk.workspacesURL, workspacesEndpoint, id, roleID, "", pm, token)
}

func (sdk mgSDK) RemoveWorkspaceRoleMembers(ctx context.Context, id, roleID string, members []string, token string) errors.SDKError {
	return sdk.removeRoleMembers(ctx, sdk.workspacesURL, workspacesEndpoint, id, roleID, "", members, token)
}

func (sdk mgSDK) RemoveAllWorkspaceRoleMembers(ctx context.Context, id, roleID, token string) errors.SDKError {
	return sdk.removeAllRoleMembers(ctx, sdk.workspacesURL, workspacesEndpoint, id, roleID, "", token)
}

func (sdk mgSDK) AvailableWorkspaceRoleActions(ctx context.Context, token string) ([]string, errors.SDKError) {
	return sdk.listAvailableRoleActions(ctx, sdk.workspacesURL, workspacesEndpoint, "", token)
}

func (sdk mgSDK) ListWorkspaceMembers(ctx context.Context, workspaceID string, pm PageMetadata, token string) (EntityMembersPage, errors.SDKError) {
	return sdk.listEntityMembers(ctx, sdk.workspacesURL, workspaceID, workspacesEndpoint, workspaceID, token, pm)
}
