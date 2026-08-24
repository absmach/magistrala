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
	permissionsEndpoint = "permissions"
	clientsEndpoint     = "clients"
	connectEndpoint     = "connect"
	disconnectEndpoint  = "disconnect"
	identifyEndpoint    = "identify"
	rolesEndpoint       = "roles"
	actionsEndpoint     = "actions"
)

// Client represents magistrala client.
type Client struct {
	ID              string            `json:"id,omitempty"`
	Name            string            `json:"name,omitempty"`
	Tags            []string          `json:"tags,omitempty"`
	WorkspaceID     string            `json:"workspace_id,omitempty"`
	ParentGroup     string            `json:"parent_group_id,omitempty"`
	Credentials     ClientCredentials `json:"credentials"`
	Metadata        map[string]any    `json:"metadata,omitempty"`
	PrivateMetadata map[string]any    `json:"private_metadata,omitempty"`
	CreatedAt       time.Time         `json:"created_at,omitempty"`
	UpdatedAt       time.Time         `json:"updated_at,omitempty"`
	UpdatedBy       string            `json:"updated_by,omitempty"`
	Status          string            `json:"status,omitempty"`
	Permissions     []string          `json:"permissions,omitempty"`
}

type ClientCredentials struct {
	Identity string `json:"identity,omitempty"`
	Secret   string `json:"secret,omitempty"`
}

func (sdk mgSDK) CreateClient(ctx context.Context, client Client, workspaceID, token string) (Client, errors.SDKError) {
	data, err := json.Marshal(client)
	if err != nil {
		return Client{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.clientsURL, workspaceID, clientsEndpoint)

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodPost, url, token, data, nil, http.StatusCreated)
	if sdkErr != nil {
		return Client{}, sdkErr
	}

	client = Client{}
	if err := json.Unmarshal(body, &client); err != nil {
		return Client{}, errors.NewSDKError(err)
	}

	return client, nil
}

func (sdk mgSDK) CreateClients(ctx context.Context, clients []Client, workspaceID, token string) ([]Client, errors.SDKError) {
	data, err := json.Marshal(clients)
	if err != nil {
		return []Client{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s", sdk.clientsURL, workspaceID, clientsEndpoint, "bulk")

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodPost, url, token, data, nil, http.StatusOK)
	if sdkErr != nil {
		return []Client{}, sdkErr
	}

	var ctr createClientsRes
	if err := json.Unmarshal(body, &ctr); err != nil {
		return []Client{}, errors.NewSDKError(err)
	}

	return ctr.Clients, nil
}

func (sdk mgSDK) Clients(ctx context.Context, pm PageMetadata, workspaceID, token string) (ClientsPage, errors.SDKError) {
	endpoint := fmt.Sprintf("%s/%s", workspaceID, clientsEndpoint)
	url, err := sdk.withQueryParams(sdk.clientsURL, endpoint, pm)
	if err != nil {
		return ClientsPage{}, errors.NewSDKError(err)
	}

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkErr != nil {
		return ClientsPage{}, sdkErr
	}

	var cp ClientsPage
	if err := json.Unmarshal(body, &cp); err != nil {
		return ClientsPage{}, errors.NewSDKError(err)
	}

	return cp, nil
}

func (sdk mgSDK) Client(ctx context.Context, id, workspaceID, token string) (Client, errors.SDKError) {
	if id == "" {
		return Client{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.clientsURL, workspaceID, clientsEndpoint, id)

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkErr != nil {
		return Client{}, sdkErr
	}

	var t Client
	if err := json.Unmarshal(body, &t); err != nil {
		return Client{}, errors.NewSDKError(err)
	}

	return t, nil
}

func (sdk mgSDK) UpdateClient(ctx context.Context, t Client, workspaceID, token string) (Client, errors.SDKError) {
	if t.ID == "" {
		return Client{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.clientsURL, workspaceID, clientsEndpoint, t.ID)

	data, err := json.Marshal(t)
	if err != nil {
		return Client{}, errors.NewSDKError(err)
	}

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodPatch, url, token, data, nil, http.StatusOK)
	if sdkErr != nil {
		return Client{}, sdkErr
	}

	t = Client{}
	if err := json.Unmarshal(body, &t); err != nil {
		return Client{}, errors.NewSDKError(err)
	}

	return t, nil
}

func (sdk mgSDK) UpdateClientTags(ctx context.Context, t Client, workspaceID, token string) (Client, errors.SDKError) {
	data, err := json.Marshal(t)
	if err != nil {
		return Client{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/tags", sdk.clientsURL, workspaceID, clientsEndpoint, t.ID)

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodPatch, url, token, data, nil, http.StatusOK)
	if sdkErr != nil {
		return Client{}, sdkErr
	}

	t = Client{}
	if err := json.Unmarshal(body, &t); err != nil {
		return Client{}, errors.NewSDKError(err)
	}

	return t, nil
}

func (sdk mgSDK) UpdateClientSecret(ctx context.Context, id, secret, workspaceID, token string) (Client, errors.SDKError) {
	ucsr := updateClientSecretReq{Secret: secret}

	data, err := json.Marshal(ucsr)
	if err != nil {
		return Client{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/secret", sdk.clientsURL, workspaceID, clientsEndpoint, id)

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodPatch, url, token, data, nil, http.StatusOK)
	if sdkErr != nil {
		return Client{}, sdkErr
	}

	var t Client
	if err = json.Unmarshal(body, &t); err != nil {
		return Client{}, errors.NewSDKError(err)
	}

	return t, nil
}

func (sdk mgSDK) EnableClient(ctx context.Context, id, workspaceID, token string) (Client, errors.SDKError) {
	return sdk.changeClientStatus(ctx, id, enableEndpoint, workspaceID, token)
}

func (sdk mgSDK) DisableClient(ctx context.Context, id, workspaceID, token string) (Client, errors.SDKError) {
	return sdk.changeClientStatus(ctx, id, disableEndpoint, workspaceID, token)
}

func (sdk mgSDK) changeClientStatus(ctx context.Context, id, status, workspaceID, token string) (Client, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.clientsURL, workspaceID, clientsEndpoint, id, status)

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodPost, url, token, nil, nil, http.StatusOK)
	if sdkErr != nil {
		return Client{}, sdkErr
	}

	t := Client{}
	if err := json.Unmarshal(body, &t); err != nil {
		return Client{}, errors.NewSDKError(err)
	}

	return t, nil
}

func (sdk mgSDK) SetClientParent(ctx context.Context, id, workspaceID, groupID, token string) errors.SDKError {
	scpg := parentGroupReq{ParentGroupID: groupID}
	data, err := json.Marshal(scpg)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.clientsURL, workspaceID, clientsEndpoint, id, parentEndpoint)
	_, _, sdkErr := sdk.processRequest(ctx, http.MethodPost, url, token, data, nil, http.StatusOK)

	return sdkErr
}

func (sdk mgSDK) RemoveClientParent(ctx context.Context, id, workspaceID, groupID, token string) errors.SDKError {
	pgr := parentGroupReq{ParentGroupID: groupID}
	data, err := json.Marshal(pgr)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.clientsURL, workspaceID, clientsEndpoint, id, parentEndpoint)
	_, _, sdkErr := sdk.processRequest(ctx, http.MethodDelete, url, token, data, nil, http.StatusNoContent)

	return sdkErr
}

func (sdk mgSDK) DeleteClient(ctx context.Context, id, workspaceID, token string) errors.SDKError {
	if id == "" {
		return errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.clientsURL, workspaceID, clientsEndpoint, id)
	_, _, sdkErr := sdk.processRequest(ctx, http.MethodDelete, url, token, nil, nil, http.StatusNoContent)
	return sdkErr
}

func (sdk mgSDK) CreateClientRole(ctx context.Context, id, workspaceID string, rq RoleReq, token string) (Role, errors.SDKError) {
	return sdk.createRole(ctx, sdk.clientsURL, clientsEndpoint, id, workspaceID, rq, token)
}

func (sdk mgSDK) ClientRoles(ctx context.Context, id, workspaceID string, pm PageMetadata, token string) (RolesPage, errors.SDKError) {
	return sdk.listRoles(ctx, sdk.clientsURL, clientsEndpoint, id, workspaceID, pm, token)
}

func (sdk mgSDK) ClientRole(ctx context.Context, id, roleID, workspaceID, token string) (Role, errors.SDKError) {
	return sdk.viewRole(ctx, sdk.clientsURL, clientsEndpoint, id, roleID, workspaceID, token)
}

func (sdk mgSDK) UpdateClientRole(ctx context.Context, id, roleID, newName, workspaceID string, token string) (Role, errors.SDKError) {
	return sdk.updateRole(ctx, sdk.clientsURL, clientsEndpoint, id, roleID, newName, workspaceID, token)
}

func (sdk mgSDK) DeleteClientRole(ctx context.Context, id, roleID, workspaceID, token string) errors.SDKError {
	return sdk.deleteRole(ctx, sdk.clientsURL, clientsEndpoint, id, roleID, workspaceID, token)
}

func (sdk mgSDK) AddClientRoleActions(ctx context.Context, id, roleID, workspaceID string, actions []string, token string) ([]string, errors.SDKError) {
	return sdk.addRoleActions(ctx, sdk.clientsURL, clientsEndpoint, id, roleID, workspaceID, actions, token)
}

func (sdk mgSDK) ClientRoleActions(ctx context.Context, id, roleID, workspaceID string, token string) ([]string, errors.SDKError) {
	return sdk.listRoleActions(ctx, sdk.clientsURL, clientsEndpoint, id, roleID, workspaceID, token)
}

func (sdk mgSDK) RemoveClientRoleActions(ctx context.Context, id, roleID, workspaceID string, actions []string, token string) errors.SDKError {
	return sdk.removeRoleActions(ctx, sdk.clientsURL, clientsEndpoint, id, roleID, workspaceID, actions, token)
}

func (sdk mgSDK) RemoveAllClientRoleActions(ctx context.Context, id, roleID, workspaceID, token string) errors.SDKError {
	return sdk.removeAllRoleActions(ctx, sdk.clientsURL, clientsEndpoint, id, roleID, workspaceID, token)
}

func (sdk mgSDK) AddClientRoleMembers(ctx context.Context, id, roleID, workspaceID string, members []string, token string) ([]string, errors.SDKError) {
	return sdk.addRoleMembers(ctx, sdk.clientsURL, clientsEndpoint, id, roleID, workspaceID, members, token)
}

func (sdk mgSDK) ClientRoleMembers(ctx context.Context, id, roleID, workspaceID string, pm PageMetadata, token string) (RoleMembersPage, errors.SDKError) {
	return sdk.listRoleMembers(ctx, sdk.clientsURL, clientsEndpoint, id, roleID, workspaceID, pm, token)
}

func (sdk mgSDK) RemoveClientRoleMembers(ctx context.Context, id, roleID, workspaceID string, members []string, token string) errors.SDKError {
	return sdk.removeRoleMembers(ctx, sdk.clientsURL, clientsEndpoint, id, roleID, workspaceID, members, token)
}

func (sdk mgSDK) RemoveAllClientRoleMembers(ctx context.Context, id, roleID, workspaceID, token string) errors.SDKError {
	return sdk.removeAllRoleMembers(ctx, sdk.clientsURL, clientsEndpoint, id, roleID, workspaceID, token)
}

func (sdk mgSDK) AvailableClientRoleActions(ctx context.Context, workspaceID, token string) ([]string, errors.SDKError) {
	return sdk.listAvailableRoleActions(ctx, sdk.clientsURL, clientsEndpoint, workspaceID, token)
}

func (sdk mgSDK) ListClientMembers(ctx context.Context, clientID, workspaceID string, pm PageMetadata, token string) (EntityMembersPage, errors.SDKError) {
	return sdk.listEntityMembers(ctx, sdk.clientsURL, workspaceID, clientsEndpoint, clientID, token, pm)
}
