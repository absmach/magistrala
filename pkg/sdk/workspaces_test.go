// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkspaceSDKRoutes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/workspaces":
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprint(w, `{"id":"workspace-1","name":"test workspace"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/workspaces":
			_, _ = fmt.Fprint(w, `{"total":1,"offset":0,"limit":10,"workspaces":[{"id":"workspace-1","name":"test workspace"}]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/workspaces/workspace-1":
			_, _ = fmt.Fprint(w, `{"id":"workspace-1","name":"test workspace"}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/workspaces/workspace-1":
			_, _ = fmt.Fprint(w, `{"id":"workspace-1","name":"updated workspace"}`)
		case r.Method == http.MethodPost && (r.URL.Path == "/workspaces/workspace-1/enable" || r.URL.Path == "/workspaces/workspace-1/disable" || r.URL.Path == "/workspaces/workspace-1/freeze"):
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && r.URL.Path == "/workspaces/workspace-1/roles":
			_, _ = fmt.Fprint(w, `{"total":1,"offset":0,"limit":10,"roles":[{"id":"role-1","name":"reader"}]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/workspaces/workspace-1/roles/role-1":
			_, _ = fmt.Fprint(w, `{"id":"role-1","name":"reader"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/workspaces/workspace-1/roles":
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprint(w, `{"id":"role-1","name":"reader"}`)
		case r.Method == http.MethodPut && r.URL.Path == "/workspaces/workspace-1/roles/role-1":
			_, _ = fmt.Fprint(w, `{"id":"role-1","name":"writer"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/workspaces/roles/available-actions":
			_, _ = fmt.Fprint(w, `{"available_actions":["read"]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/workspaces/workspace-1/roles/role-1/actions":
			_, _ = fmt.Fprint(w, `{"actions":["read"]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/workspaces/workspace-1/roles/role-1/actions":
			_, _ = fmt.Fprint(w, `{"actions":["read"]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/workspaces/workspace-1/roles/role-1/members":
			_, _ = fmt.Fprint(w, `{"total":1,"offset":0,"limit":10,"members":["user-1"]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/workspaces/workspace-1/roles/role-1/members":
			_, _ = fmt.Fprint(w, `{"members":["user-1"]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/workspaces/workspace-1/roles/members":
			_, _ = fmt.Fprint(w, `{"total":1,"offset":0,"limit":10,"members":[{"member_id":"user-1","roles":[]}]}`)
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()

	sdk := mgSDK{
		workspacesURL:  server.URL,
		client:         server.Client(),
		msgContentType: CTJSON,
	}
	ctx := context.Background()

	workspace, sdkErr := sdk.CreateWorkspace(ctx, Workspace{Name: "test workspace"}, "token")
	require.Nil(t, sdkErr)
	require.Equal(t, "workspace-1", workspace.ID)

	page, sdkErr := sdk.Workspaces(ctx, PageMetadata{Limit: 10}, "token")
	require.Nil(t, sdkErr)
	require.Len(t, page.Workspaces, 1)

	workspace, sdkErr = sdk.Workspace(ctx, "workspace-1", "token")
	require.Nil(t, sdkErr)
	require.Equal(t, "workspace-1", workspace.ID)

	workspace, sdkErr = sdk.UpdateWorkspace(ctx, Workspace{ID: "workspace-1", Name: "updated workspace"}, "token")
	require.Nil(t, sdkErr)
	require.Equal(t, "updated workspace", workspace.Name)
	require.Nil(t, sdk.EnableWorkspace(ctx, "workspace-1", "token"))
	require.Nil(t, sdk.DisableWorkspace(ctx, "workspace-1", "token"))
	require.Nil(t, sdk.FreezeWorkspace(ctx, "workspace-1", "token"))

	role, sdkErr := sdk.CreateWorkspaceRole(ctx, "workspace-1", RoleReq{RoleName: "reader"}, "token")
	require.Nil(t, sdkErr)
	require.Equal(t, "role-1", role.ID)
	roles, sdkErr := sdk.WorkspaceRoles(ctx, "workspace-1", PageMetadata{Limit: 10}, "token")
	require.Nil(t, sdkErr)
	require.Len(t, roles.Roles, 1)
	role, sdkErr = sdk.WorkspaceRole(ctx, "workspace-1", "role-1", "token")
	require.Nil(t, sdkErr)
	require.Equal(t, "role-1", role.ID)
	role, sdkErr = sdk.UpdateWorkspaceRole(ctx, "workspace-1", "role-1", "writer", "token")
	require.Nil(t, sdkErr)
	require.Equal(t, "writer", role.Name)
	require.Nil(t, sdk.DeleteWorkspaceRole(ctx, "workspace-1", "role-1", "token"))

	actions, sdkErr := sdk.AvailableWorkspaceRoleActions(ctx, "token")
	require.Nil(t, sdkErr)
	require.Equal(t, []string{"read"}, actions)
	actions, sdkErr = sdk.AddWorkspaceRoleActions(ctx, "workspace-1", "role-1", []string{"read"}, "token")
	require.Nil(t, sdkErr)
	require.Equal(t, []string{"read"}, actions)
	actions, sdkErr = sdk.WorkspaceRoleActions(ctx, "workspace-1", "role-1", "token")
	require.Nil(t, sdkErr)
	require.Equal(t, []string{"read"}, actions)
	require.Nil(t, sdk.RemoveWorkspaceRoleActions(ctx, "workspace-1", "role-1", []string{"read"}, "token"))
	require.Nil(t, sdk.RemoveAllWorkspaceRoleActions(ctx, "workspace-1", "role-1", "token"))

	members, sdkErr := sdk.AddWorkspaceRoleMembers(ctx, "workspace-1", "role-1", []string{"user-1"}, "token")
	require.Nil(t, sdkErr)
	require.Equal(t, []string{"user-1"}, members)
	roleMembers, sdkErr := sdk.WorkspaceRoleMembers(ctx, "workspace-1", "role-1", PageMetadata{Limit: 10}, "token")
	require.Nil(t, sdkErr)
	require.Equal(t, []string{"user-1"}, roleMembers.Members)
	require.Nil(t, sdk.RemoveWorkspaceRoleMembers(ctx, "workspace-1", "role-1", []string{"user-1"}, "token"))
	require.Nil(t, sdk.RemoveAllWorkspaceRoleMembers(ctx, "workspace-1", "role-1", "token"))

	entityMembers, sdkErr := sdk.ListWorkspaceMembers(ctx, "workspace-1", PageMetadata{Limit: 10}, "token")
	require.Nil(t, sdkErr)
	require.Len(t, entityMembers.Members, 1)
}
