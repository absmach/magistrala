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
	ID              string                    `json:"id,omitempty"`
	Name            string                    `json:"name,omitempty"`
	Tags            []string                  `json:"tags,omitempty"`
	DomainID        string                    `json:"domain_id,omitempty"`
	ParentGroup     string                    `json:"parent_group_id,omitempty"`
	Credentials     ClientCredentials         `json:"credentials"`
	Metadata        map[string]any            `json:"metadata,omitempty"`
	PrivateMetadata map[string]any            `json:"private_metadata,omitempty"`
	CreatedAt       time.Time                 `json:"created_at,omitempty"`
	UpdatedAt       time.Time                 `json:"updated_at,omitempty"`
	UpdatedBy       string                    `json:"updated_by,omitempty"`
	Status          string                    `json:"status,omitempty"`
	Permissions     []string                  `json:"permissions,omitempty"`
	Roles           []roles.MemberRoleActions `json:"roles,omitempty"`
}

type ClientCredentials struct {
	Identity string `json:"identity,omitempty"`
	Secret   string `json:"secret,omitempty"`
}

func (sdk mgSDK) CreateClient(ctx context.Context, client Client, domainID, token string) (Client, errors.SDKError) {
	data, err := json.Marshal(client)
	if err != nil {
		return Client{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.clientsURL, domainID, clientsEndpoint)

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

func (sdk mgSDK) CreateClients(ctx context.Context, clients []Client, domainID, token string) ([]Client, errors.SDKError) {
	data, err := json.Marshal(clients)
	if err != nil {
		return []Client{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s", sdk.clientsURL, domainID, clientsEndpoint, "bulk")

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

func (sdk mgSDK) Clients(ctx context.Context, pm PageMetadata, domainID, token string) (ClientsPage, errors.SDKError) {
	endpoint := fmt.Sprintf("%s/%s", domainID, clientsEndpoint)
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

func (sdk mgSDK) Client(ctx context.Context, id, domainID, token string) (Client, errors.SDKError) {
	if id == "" {
		return Client{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.clientsURL, domainID, clientsEndpoint, id)

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

func (sdk mgSDK) UpdateClient(ctx context.Context, t Client, domainID, token string) (Client, errors.SDKError) {
	if t.ID == "" {
		return Client{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.clientsURL, domainID, clientsEndpoint, t.ID)

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

func (sdk mgSDK) UpdateClientTags(ctx context.Context, t Client, domainID, token string) (Client, errors.SDKError) {
	data, err := json.Marshal(t)
	if err != nil {
		return Client{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/tags", sdk.clientsURL, domainID, clientsEndpoint, t.ID)

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

func (sdk mgSDK) UpdateClientSecret(ctx context.Context, id, secret, domainID, token string) (Client, errors.SDKError) {
	ucsr := updateClientSecretReq{Secret: secret}

	data, err := json.Marshal(ucsr)
	if err != nil {
		return Client{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/secret", sdk.clientsURL, domainID, clientsEndpoint, id)

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

func (sdk mgSDK) EnableClient(ctx context.Context, id, domainID, token string) (Client, errors.SDKError) {
	return sdk.changeClientStatus(ctx, id, enableEndpoint, domainID, token)
}

func (sdk mgSDK) DisableClient(ctx context.Context, id, domainID, token string) (Client, errors.SDKError) {
	return sdk.changeClientStatus(ctx, id, disableEndpoint, domainID, token)
}

func (sdk mgSDK) changeClientStatus(ctx context.Context, id, status, domainID, token string) (Client, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.clientsURL, domainID, clientsEndpoint, id, status)

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

func (sdk mgSDK) SetClientParent(ctx context.Context, id, domainID, groupID, token string) errors.SDKError {
	scpg := parentGroupReq{ParentGroupID: groupID}
	data, err := json.Marshal(scpg)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.clientsURL, domainID, clientsEndpoint, id, parentEndpoint)
	_, _, sdkErr := sdk.processRequest(ctx, http.MethodPost, url, token, data, nil, http.StatusOK)

	return sdkErr
}

func (sdk mgSDK) RemoveClientParent(ctx context.Context, id, domainID, groupID, token string) errors.SDKError {
	pgr := parentGroupReq{ParentGroupID: groupID}
	data, err := json.Marshal(pgr)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.clientsURL, domainID, clientsEndpoint, id, parentEndpoint)
	_, _, sdkErr := sdk.processRequest(ctx, http.MethodDelete, url, token, data, nil, http.StatusNoContent)

	return sdkErr
}

func (sdk mgSDK) DeleteClient(ctx context.Context, id, domainID, token string) errors.SDKError {
	if id == "" {
		return errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.clientsURL, domainID, clientsEndpoint, id)
	_, _, sdkErr := sdk.processRequest(ctx, http.MethodDelete, url, token, nil, nil, http.StatusNoContent)
	return sdkErr
}

func (sdk mgSDK) CreateClientRole(ctx context.Context, id, domainID string, rq RoleReq, token string) (Role, errors.SDKError) {
	return sdk.createRole(ctx, sdk.clientsURL, clientsEndpoint, id, domainID, rq, token)
}

func (sdk mgSDK) ClientRoles(ctx context.Context, id, domainID string, pm PageMetadata, token string) (RolesPage, errors.SDKError) {
	return sdk.listRoles(ctx, sdk.clientsURL, clientsEndpoint, id, domainID, pm, token)
}

func (sdk mgSDK) ClientRole(ctx context.Context, id, roleID, domainID, token string) (Role, errors.SDKError) {
	return sdk.viewRole(ctx, sdk.clientsURL, clientsEndpoint, id, roleID, domainID, token)
}

func (sdk mgSDK) UpdateClientRole(ctx context.Context, id, roleID, newName, domainID string, token string) (Role, errors.SDKError) {
	return sdk.updateRole(ctx, sdk.clientsURL, clientsEndpoint, id, roleID, newName, domainID, token)
}

func (sdk mgSDK) DeleteClientRole(ctx context.Context, id, roleID, domainID, token string) errors.SDKError {
	return sdk.deleteRole(ctx, sdk.clientsURL, clientsEndpoint, id, roleID, domainID, token)
}

func (sdk mgSDK) AddClientRoleActions(ctx context.Context, id, roleID, domainID string, actions []string, token string) ([]string, errors.SDKError) {
	return sdk.addRoleActions(ctx, sdk.clientsURL, clientsEndpoint, id, roleID, domainID, actions, token)
}

func (sdk mgSDK) ClientRoleActions(ctx context.Context, id, roleID, domainID string, token string) ([]string, errors.SDKError) {
	return sdk.listRoleActions(ctx, sdk.clientsURL, clientsEndpoint, id, roleID, domainID, token)
}

func (sdk mgSDK) RemoveClientRoleActions(ctx context.Context, id, roleID, domainID string, actions []string, token string) errors.SDKError {
	return sdk.removeRoleActions(ctx, sdk.clientsURL, clientsEndpoint, id, roleID, domainID, actions, token)
}

func (sdk mgSDK) RemoveAllClientRoleActions(ctx context.Context, id, roleID, domainID, token string) errors.SDKError {
	return sdk.removeAllRoleActions(ctx, sdk.clientsURL, clientsEndpoint, id, roleID, domainID, token)
}

func (sdk mgSDK) AddClientRoleMembers(ctx context.Context, id, roleID, domainID string, members []string, token string) ([]string, errors.SDKError) {
	return sdk.addRoleMembers(ctx, sdk.clientsURL, clientsEndpoint, id, roleID, domainID, members, token)
}

func (sdk mgSDK) ClientRoleMembers(ctx context.Context, id, roleID, domainID string, pm PageMetadata, token string) (RoleMembersPage, errors.SDKError) {
	return sdk.listRoleMembers(ctx, sdk.clientsURL, clientsEndpoint, id, roleID, domainID, pm, token)
}

func (sdk mgSDK) RemoveClientRoleMembers(ctx context.Context, id, roleID, domainID string, members []string, token string) errors.SDKError {
	return sdk.removeRoleMembers(ctx, sdk.clientsURL, clientsEndpoint, id, roleID, domainID, members, token)
}

func (sdk mgSDK) RemoveAllClientRoleMembers(ctx context.Context, id, roleID, domainID, token string) errors.SDKError {
	return sdk.removeAllRoleMembers(ctx, sdk.clientsURL, clientsEndpoint, id, roleID, domainID, token)
}

func (sdk mgSDK) AvailableClientRoleActions(ctx context.Context, domainID, token string) ([]string, errors.SDKError) {
	return sdk.listAvailableRoleActions(ctx, sdk.clientsURL, clientsEndpoint, domainID, token)
}

func (sdk mgSDK) ListClientMembers(ctx context.Context, clientID, domainID string, pm PageMetadata, token string) (EntityMembersPage, errors.SDKError) {
	return sdk.listEntityMembers(ctx, sdk.clientsURL, domainID, clientsEndpoint, clientID, token, pm)
}
