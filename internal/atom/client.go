// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	baseURL       string
	token         string
	adminUsername string
	adminSecret   string
	userAgent     string
	httpClient    *http.Client
}

func NewClient(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	return &Client{
		baseURL:       strings.TrimRight(cfg.URL, "/"),
		token:         cfg.Token,
		adminUsername: cfg.AdminUsername,
		adminSecret:   cfg.AdminSecret,
		userAgent:     cfg.UserAgent,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) UpsertTenant(ctx context.Context, tenant Tenant) error {
	if tenant.ID == "" {
		_, err := c.CreateTenant(ctx, tenant)
		return err
	}
	if _, err := c.CreateTenant(ctx, tenant); err == nil || !IsConflict(err) {
		return err
	}
	_, err := c.UpdateTenant(ctx, tenant.ID, tenant)
	return err
}

func (c *Client) CreateTenant(ctx context.Context, tenant Tenant) (Tenant, error) {
	var out struct {
		CreateTenant Tenant `json:"createTenant"`
	}
	err := c.graphQL(ctx, `mutation CreateTenant($input: CreateTenantInput!) {
		createTenant(input: $input) { id name route: alias status tags attributes created_by: createdBy updated_by: updatedBy created_at: createdAt updated_at: updatedAt }
	}`, map[string]any{"input": tenantCreateInput(tenant)}, &out)
	return out.CreateTenant, err
}

func (c *Client) GetTenant(ctx context.Context, id string) (Tenant, error) {
	var out struct {
		Tenant Tenant `json:"tenant"`
	}
	err := c.graphQL(ctx, `query Tenant($id: ID!) {
		tenant(id: $id) { id name route: alias status tags attributes created_by: createdBy updated_by: updatedBy created_at: createdAt updated_at: updatedAt }
	}`, map[string]any{"id": id}, &out)
	return out.Tenant, err
}

func (c *Client) UpdateTenant(ctx context.Context, id string, tenant Tenant) (Tenant, error) {
	var out struct {
		UpdateTenant Tenant `json:"updateTenant"`
	}
	err := c.graphQL(ctx, `mutation UpdateTenant($id: ID!, $input: UpdateTenantInput!) {
		updateTenant(id: $id, input: $input) { id name route: alias status tags attributes created_by: createdBy updated_by: updatedBy created_at: createdAt updated_at: updatedAt }
	}`, map[string]any{"id": id, "input": tenantUpdateInput(tenant)}, &out)
	return out.UpdateTenant, err
}

func (c *Client) ChangeTenantStatus(ctx context.Context, id, action string) (Tenant, error) {
	field := map[string]string{
		"enable":  "enableTenant",
		"disable": "disableTenant",
		"freeze":  "freezeTenant",
	}[action]
	if field == "" {
		return Tenant{}, Error{StatusCode: http.StatusBadRequest, Message: "unsupported tenant status action: " + action}
	}
	var out map[string]Tenant
	err := c.graphQL(ctx, fmt.Sprintf(`mutation ChangeTenantStatus($id: ID!) {
		%s(id: $id) { id name route: alias status tags attributes created_by: createdBy updated_by: updatedBy created_at: createdAt updated_at: updatedAt }
	}`, field), map[string]any{"id": id}, &out)
	if err != nil {
		return Tenant{}, err
	}
	return out[field], nil
}

func (c *Client) UpsertEntity(ctx context.Context, entity Entity) error {
	if entity.ID == "" {
		_, err := c.CreateEntity(ctx, entity)
		return err
	}
	if _, err := c.CreateEntity(ctx, entity); err == nil || !IsConflict(err) {
		return err
	}
	_, err := c.UpdateEntity(ctx, entity.ID, entity)
	return err
}

func (c *Client) CreateEntity(ctx context.Context, entity Entity) (Entity, error) {
	var out struct {
		CreateEntity Entity `json:"createEntity"`
	}
	err := c.graphQL(ctx, `mutation CreateEntity($input: CreateEntityInput!) {
		createEntity(input: $input) { id kind name tenant_id: tenantId status attributes created_at: createdAt updated_at: updatedAt }
	}`, map[string]any{"input": entityCreateInput(entity)}, &out)
	return out.CreateEntity, err
}

func (c *Client) GetEntity(ctx context.Context, id string) (Entity, error) {
	var out struct {
		Entity Entity `json:"entity"`
	}
	err := c.graphQL(ctx, `query Entity($id: ID!) {
		entity(id: $id) { id kind name tenant_id: tenantId status attributes created_at: createdAt updated_at: updatedAt }
	}`, map[string]any{"id": id}, &out)
	return out.Entity, err
}

func (c *Client) UpdateEntity(ctx context.Context, id string, entity Entity) (Entity, error) {
	var out struct {
		UpdateEntity Entity `json:"updateEntity"`
	}
	err := c.graphQL(ctx, `mutation UpdateEntity($id: ID!, $input: UpdateEntityInput!) {
		updateEntity(id: $id, input: $input) { id kind name tenant_id: tenantId status attributes created_at: createdAt updated_at: updatedAt }
	}`, map[string]any{"id": id, "input": entityUpdateInput(entity)}, &out)
	return out.UpdateEntity, err
}

func (c *Client) UpsertGroup(ctx context.Context, group Group) error {
	if group.ID == "" {
		_, err := c.CreateGroup(ctx, group)
		return err
	}
	if _, err := c.CreateGroup(ctx, group); err == nil || !IsConflict(err) {
		return err
	}
	_, err := c.UpdateGroup(ctx, group.ID, group)
	return err
}

func (c *Client) CreateGroup(ctx context.Context, group Group) (Group, error) {
	var out struct {
		CreateGroup Group `json:"createGroup"`
	}
	err := c.graphQL(ctx, `mutation CreateGroup($input: CreateGroupInput!) {
		createGroup(input: $input) { id name tenant_id: tenantId description parent_id: parentId status attributes created_at: createdAt updated_at: updatedAt }
	}`, map[string]any{"input": groupCreateInput(group)}, &out)
	return out.CreateGroup, err
}

func (c *Client) GetGroup(ctx context.Context, id string) (Group, error) {
	var out struct {
		Group Group `json:"group"`
	}
	err := c.graphQL(ctx, `query Group($id: ID!) {
		group(id: $id) { id name tenant_id: tenantId description parent_id: parentId status attributes created_at: createdAt updated_at: updatedAt }
	}`, map[string]any{"id": id}, &out)
	return out.Group, err
}

func (c *Client) UpdateGroup(ctx context.Context, id string, group Group) (Group, error) {
	var out struct {
		UpdateGroup Group `json:"updateGroup"`
	}
	err := c.graphQL(ctx, `mutation UpdateGroup($id: ID!, $input: UpdateGroupInput!) {
		updateGroup(id: $id, input: $input) { id name tenant_id: tenantId description parent_id: parentId status attributes created_at: createdAt updated_at: updatedAt }
	}`, map[string]any{"id": id, "input": groupUpdateInput(group)}, &out)
	return out.UpdateGroup, err
}

func (c *Client) UpsertResource(ctx context.Context, resource Resource) error {
	if resource.ID == "" {
		_, err := c.CreateResource(ctx, resource)
		return err
	}
	if _, err := c.CreateResource(ctx, resource); err == nil || !IsConflict(err) {
		return err
	}
	_, err := c.UpdateResource(ctx, resource.ID, resource)
	return err
}

func (c *Client) CreateResource(ctx context.Context, resource Resource) (Resource, error) {
	var out struct {
		CreateResource Resource `json:"createResource"`
	}
	err := c.graphQL(ctx, `mutation CreateResource($input: CreateResourceInput!) {
		createResource(input: $input) { id kind name tenant_id: tenantId owner_id: ownerId attributes created_at: createdAt updated_at: updatedAt }
	}`, map[string]any{"input": resourceCreateInput(resource)}, &out)
	return out.CreateResource, err
}

func (c *Client) GetResource(ctx context.Context, id string) (Resource, error) {
	var out struct {
		Resource Resource `json:"resource"`
	}
	err := c.graphQL(ctx, `query Resource($id: ID!) {
		resource(id: $id) { id kind name tenant_id: tenantId owner_id: ownerId attributes created_at: createdAt updated_at: updatedAt }
	}`, map[string]any{"id": id}, &out)
	return out.Resource, err
}

func (c *Client) UpdateResource(ctx context.Context, id string, resource Resource) (Resource, error) {
	var out struct {
		UpdateResource Resource `json:"updateResource"`
	}
	err := c.graphQL(ctx, `mutation UpdateResource($id: ID!, $input: UpdateResourceInput!) {
		updateResource(id: $id, input: $input) { id kind name tenant_id: tenantId owner_id: ownerId attributes created_at: createdAt updated_at: updatedAt }
	}`, map[string]any{"id": id, "input": resourceUpdateInput(resource)}, &out)
	return out.UpdateResource, err
}

func (c *Client) DeleteTenant(ctx context.Context, id string) error {
	return c.graphQL(ctx, `mutation DeleteTenant($id: ID!) { deleteTenant(id: $id) }`, map[string]any{"id": id}, nil)
}

func (c *Client) ListTenants(ctx context.Context, q Query) (TenantList, error) {
	var out struct {
		Tenants TenantList `json:"tenants"`
	}
	err := c.graphQL(ctx, `query Tenants($q: String, $name: String, $alias: String, $status: TenantStatus, $limit: Int, $offset: Int) {
		tenants(q: $q, name: $name, alias: $alias, status: $status, limit: $limit, offset: $offset) {
			total
			items { id name route: alias status tags attributes created_by: createdBy updated_by: updatedBy created_at: createdAt updated_at: updatedAt }
		}
	}`, queryVariables(q), &out)
	return out.Tenants, err
}

func (c *Client) CheckAuthz(ctx context.Context, req AuthzRequest) (AuthzResponse, error) {
	var out struct {
		AuthzCheck AuthzResponse `json:"authzCheck"`
	}
	err := c.graphQL(ctx, `mutation AuthzCheck($input: AuthzCheckInput!) {
		authzCheck(input: $input) { allowed reason }
	}`, map[string]any{"input": authzInput(req)}, &out)
	return out.AuthzCheck, err
}

func (c *Client) CheckAuthzWithToken(ctx context.Context, token string, req AuthzRequest) (AuthzResponse, error) {
	var out struct {
		AuthzCheck AuthzResponse `json:"authzCheck"`
	}
	err := c.graphQLWithToken(ctx, `mutation AuthzCheck($input: AuthzCheckInput!) {
		authzCheck(input: $input) { allowed reason }
	}`, map[string]any{"input": authzInput(req)}, &out, token)
	return out.AuthzCheck, err
}

func (c *Client) ListCapabilities(ctx context.Context) (CapabilityList, error) {
	var out struct {
		Actions CapabilityList `json:"actions"`
	}
	err := c.graphQL(ctx, `query Actions($limit: Int!) {
		actions(limit: $limit) { total items { id name description } }
	}`, map[string]any{"limit": 100}, &out)
	return out.Actions, err
}

func (c *Client) CapabilityID(ctx context.Context, name string) (string, error) {
	list, err := c.ListCapabilities(ctx)
	if err != nil {
		return "", err
	}
	for _, capability := range list.Items {
		if capability.Name == name {
			return capability.ID, nil
		}
	}
	return "", Error{StatusCode: http.StatusNotFound, Message: "capability " + name + " not found"}
}

func (c *Client) CreateCapability(ctx context.Context, name, description string) (Capability, error) {
	var out struct {
		CreateAction Capability `json:"createAction"`
	}
	input := map[string]any{"name": name}
	setIfNotEmpty(input, "description", description)
	err := c.graphQL(ctx, `mutation CreateAction($input: CreateActionInput!) {
		createAction(input: $input) { id name description }
	}`, map[string]any{"input": input}, &out)
	return out.CreateAction, err
}

func (c *Client) AddCapabilityApplicability(ctx context.Context, actionID, objectKind, objectType string) (CapabilityApplicability, error) {
	var out struct {
		AddActionApplicability CapabilityApplicability `json:"addActionApplicability"`
	}
	input := map[string]any{
		"actionId":   actionID,
		"objectKind": objectKind,
	}
	setIfNotEmpty(input, "objectType", objectType)
	err := c.graphQL(ctx, `mutation AddActionApplicability($input: AddActionApplicabilityInput!) {
		addActionApplicability(input: $input) {
			action_id: actionId
			action_name: actionName
			description
			object_kind: objectKind
			object_type: objectType
		}
	}`, map[string]any{"input": input}, &out)
	return out.AddActionApplicability, err
}

func (c *Client) ListActionAssignmentRules(ctx context.Context, spec ActionAssignmentRuleSpec) (ActionAssignmentRuleList, error) {
	var out struct {
		ActionAssignmentRules ActionAssignmentRuleList `json:"actionAssignmentRules"`
	}
	vars := map[string]any{"limit": 100, "offset": 0}
	setIfNotEmpty(vars, "tenantId", spec.TenantID)
	setIfNotEmpty(vars, "entityKind", spec.EntityKind)
	setIfNotEmpty(vars, "actionName", spec.ActionName)
	setIfNotEmpty(vars, "objectKind", spec.ObjectKind)
	setIfNotEmpty(vars, "objectType", spec.ObjectType)
	setIfNotEmpty(vars, "decision", spec.Decision)
	err := c.graphQL(ctx, `query ActionAssignmentRules(
		$tenantId: ID,
		$entityKind: EntityKind,
		$actionName: String,
		$objectKind: String,
		$objectType: String,
		$decision: ActionAssignmentRuleDecision,
		$limit: Int!,
		$offset: Int!
	) {
		actionAssignmentRules(
			tenantId: $tenantId,
			entityKind: $entityKind,
			actionName: $actionName,
			objectKind: $objectKind,
			objectType: $objectType,
			decision: $decision,
			limit: $limit,
			offset: $offset
		) {
			total
			items {
				id
				tenant_id: tenantId
				entity_kind: entityKind
				action_name: actionName
				object_kind: objectKind
				object_type: objectType
				decision
				is_absolute: isAbsolute
				created_at: createdAt
			}
		}
	}`, vars, &out)
	return out.ActionAssignmentRules, err
}

func (c *Client) CreateActionAssignmentRule(ctx context.Context, spec ActionAssignmentRuleSpec) (ActionAssignmentRule, error) {
	var out struct {
		CreateActionAssignmentRule ActionAssignmentRule `json:"createActionAssignmentRule"`
	}
	input := map[string]any{
		"entityKind": spec.EntityKind,
		"actionName": spec.ActionName,
		"objectKind": spec.ObjectKind,
		"decision":   spec.Decision,
		"isAbsolute": spec.IsAbsolute,
	}
	setIfNotEmpty(input, "tenantId", spec.TenantID)
	setIfNotEmpty(input, "objectType", spec.ObjectType)
	err := c.graphQL(ctx, `mutation CreateActionAssignmentRule($input: CreateActionAssignmentRuleInput!) {
		createActionAssignmentRule(input: $input) {
			id
			tenant_id: tenantId
			entity_kind: entityKind
			action_name: actionName
			object_kind: objectKind
			object_type: objectType
			decision
			is_absolute: isAbsolute
			created_at: createdAt
		}
	}`, map[string]any{"input": input}, &out)
	return out.CreateActionAssignmentRule, err
}

func (c *Client) CreatePermissionBlock(ctx context.Context, block CreatePermissionBlock) (PermissionBlock, error) {
	var out struct {
		CreatePermissionBlock PermissionBlock `json:"createPermissionBlock"`
	}
	err := c.graphQL(ctx, `mutation CreatePermissionBlock($input: CreatePermissionBlockInput!) {
		createPermissionBlock(input: $input) {
			id tenant_id: tenantId scope_mode: scopeMode object_kind: objectKind object_type: objectType object_id: objectId group_id: groupId effect conditions
			actions { id name description }
		}
	}`, map[string]any{"input": permissionBlockInput(block)}, &out)
	return out.CreatePermissionBlock, err
}

func (c *Client) CreateDirectPolicy(ctx context.Context, policy CreateDirectPolicy) (DirectPolicy, error) {
	var out struct {
		CreateDirectPolicy DirectPolicy `json:"createDirectPolicy"`
	}
	err := c.graphQL(ctx, `mutation CreateDirectPolicy($input: CreateDirectPolicyInput!) {
		createDirectPolicy(input: $input) {
			id tenant_id: tenantId subject_kind: subjectKind subject_id: subjectId permission_block_id: permissionBlockId created_at: createdAt
			permission_block: permissionBlock {
				id tenant_id: tenantId scope_mode: scopeMode object_kind: objectKind object_type: objectType object_id: objectId group_id: groupId effect conditions
				actions { id name description }
			}
		}
	}`, map[string]any{"input": directPolicyInput(policy)}, &out)
	return out.CreateDirectPolicy, err
}

func (c *Client) ListDirectPolicies(ctx context.Context, q DirectPolicyQuery) (DirectPolicyList, error) {
	var out struct {
		DirectPolicies DirectPolicyList `json:"directPolicies"`
	}
	err := c.graphQL(ctx, `query DirectPolicies($tenantId: ID, $subjectKind: SubjectKind, $subjectId: ID, $limit: Int, $offset: Int) {
		directPolicies(tenantId: $tenantId, subjectKind: $subjectKind, subjectId: $subjectId, limit: $limit, offset: $offset) {
			total
			items {
				id tenant_id: tenantId subject_kind: subjectKind subject_id: subjectId permission_block_id: permissionBlockId created_at: createdAt
				permission_block: permissionBlock {
					id tenant_id: tenantId scope_mode: scopeMode object_kind: objectKind object_type: objectType object_id: objectId group_id: groupId effect conditions
					actions { id name description }
				}
			}
		}
	}`, directPolicyQueryVariables(q), &out)
	return out.DirectPolicies, err
}

func (c *Client) DeleteDirectPolicy(ctx context.Context, id string) error {
	return c.graphQL(ctx, `mutation DeleteDirectPolicy($id: ID!) { deleteDirectPolicy(id: $id) }`, map[string]any{"id": id}, nil)
}

func (c *Client) AuthorizedObjectIDs(ctx context.Context, q AuthorizedObjectIDsQuery) (AuthorizedObjectIDs, error) {
	var out struct {
		AuthorizedObjectIDs AuthorizedObjectIDs `json:"authorizedObjectIds"`
	}
	err := c.graphQL(ctx, `query AuthorizedObjectIDs($input: AuthorizedObjectIdsInput!) {
		authorizedObjectIds(input: $input) {
			ids
			total
		}
	}`, authorizedObjectIDVariables(q), &out)
	return out.AuthorizedObjectIDs, err
}

func (c *Client) LoginPassword(ctx context.Context, identifier, secret string) (LoginResponse, error) {
	var out LoginResponse
	err := c.doWithToken(ctx, http.MethodPost, "/auth/login", LoginRequest{
		Identifier: identifier,
		Secret:     secret,
		Kind:       "password",
	}, &out, "")
	return out, err
}

func (c *Client) Introspect(ctx context.Context, token string) (IntrospectionResponse, error) {
	var out IntrospectionResponse
	err := c.doWithToken(ctx, http.MethodGet, "/auth/introspect", nil, &out, token)
	return out, err
}

func (c *Client) DeleteEntity(ctx context.Context, id string) error {
	return c.graphQL(ctx, `mutation DeleteEntity($id: ID!) { deleteEntity(id: $id) }`, map[string]any{"id": id}, nil)
}

func (c *Client) CreatePassword(ctx context.Context, entityID, password string) error {
	return c.graphQL(ctx, `mutation CreatePassword($entityId: ID!, $password: String!) {
		createPassword(entityId: $entityId, password: $password)
	}`, map[string]any{"entityId": entityID, "password": password}, nil)
}

func (c *Client) CreateAPIKey(ctx context.Context, entityID, description string) (APIKeyResponse, error) {
	var out struct {
		CreateAPIKey APIKeyResponse `json:"createApiKey"`
	}
	err := c.graphQL(ctx, `mutation CreateAPIKey($entityId: ID!, $input: CreateApiKeyInput!) {
		createApiKey(entityId: $entityId, input: $input) {
			credentialId
			key
			expiresAt
		}
	}`, map[string]any{
		"entityId": entityID,
		"input": map[string]any{
			"description": description,
		},
	}, &out)
	return out.CreateAPIKey, err
}

func (c *Client) CreateSharedKey(ctx context.Context, entityID, key, description string) (SharedKeyResponse, error) {
	var out struct {
		CreateSharedKey SharedKeyResponse `json:"createSharedKey"`
	}
	input := map[string]any{}
	setIfNotEmpty(input, "key", key)
	setIfNotEmpty(input, "description", description)
	err := c.graphQL(ctx, `mutation CreateSharedKey($entityId: ID!, $input: CreateSharedKeyInput!) {
		createSharedKey(entityId: $entityId, input: $input) {
			credentialId
			key
			expiresAt
		}
	}`, map[string]any{
		"entityId": entityID,
		"input":    input,
	}, &out)
	return out.CreateSharedKey, err
}

func (c *Client) RevealSharedKey(ctx context.Context, entityID, credentialID string) (SharedKeyResponse, error) {
	var out struct {
		RevealSharedKey SharedKeyResponse `json:"revealSharedKey"`
	}
	err := c.graphQL(ctx, `mutation RevealSharedKey($entityId: ID!, $credentialId: ID!) {
		revealSharedKey(entityId: $entityId, credentialId: $credentialId) {
			credentialId
			key
			expiresAt
		}
	}`, map[string]any{
		"entityId":     entityID,
		"credentialId": credentialID,
	}, &out)
	return out.RevealSharedKey, err
}

func (c *Client) ListCredentials(ctx context.Context, entityID string) (CredentialList, error) {
	var out struct {
		Credentials CredentialList `json:"credentials"`
	}
	err := c.graphQL(ctx, `query Credentials($entityId: ID!) {
		credentials(entityId: $entityId) {
			total
			items {
				id
				entity_id: entityId
				kind
				identifier
				status
				expires_at: expiresAt
				created_at: createdAt
			}
		}
	}`, map[string]any{"entityId": entityID}, &out)
	return out.Credentials, err
}

func (c *Client) RevokeCredential(ctx context.Context, entityID, credentialID string) error {
	return c.graphQL(ctx, `mutation RevokeCredential($entityId: ID!, $credentialId: ID!) {
		revokeCredential(entityId: $entityId, credentialId: $credentialId)
	}`, map[string]any{"entityId": entityID, "credentialId": credentialID}, nil)
}

func (c *Client) ListEntities(ctx context.Context, q Query) (EntityList, error) {
	var out struct {
		Entities EntityList `json:"entities"`
	}
	err := c.graphQL(ctx, `query Entities($q: String, $kind: EntityKind, $tenantId: ID, $status: EntityStatus, $limit: Int, $offset: Int) {
		entities(q: $q, kind: $kind, tenantId: $tenantId, status: $status, limit: $limit, offset: $offset) {
			total
			items { id kind name tenant_id: tenantId status attributes created_at: createdAt updated_at: updatedAt }
		}
	}`, objectQueryVariables(q), &out)
	return out.Entities, err
}

func (c *Client) DeleteGroup(ctx context.Context, id string) error {
	return c.graphQL(ctx, `mutation DeleteGroup($id: ID!) { deleteGroup(id: $id) }`, map[string]any{"id": id}, nil)
}

func (c *Client) ListGroups(ctx context.Context, q Query) (GroupList, error) {
	var out struct {
		Groups GroupList `json:"groups"`
	}
	err := c.graphQL(ctx, `query Groups($q: String, $tenantId: ID, $status: EntityStatus, $limit: Int, $offset: Int) {
		groups(q: $q, tenantId: $tenantId, status: $status, limit: $limit, offset: $offset) {
			total
			items { id name tenant_id: tenantId description parent_id: parentId status attributes created_at: createdAt updated_at: updatedAt }
		}
	}`, objectQueryVariables(q), &out)
	return out.Groups, err
}

func (c *Client) DeleteResource(ctx context.Context, id string) error {
	return c.graphQL(ctx, `mutation DeleteResource($id: ID!) { deleteResource(id: $id) }`, map[string]any{"id": id}, nil)
}

func (c *Client) ListResources(ctx context.Context, q Query) (ResourceList, error) {
	var out struct {
		Resources ResourceList `json:"resources"`
	}
	vars := objectQueryVariables(q)
	if q.Name != "" && q.Q == "" {
		vars["q"] = q.Name
	}
	err := c.graphQL(ctx, `query Resources($q: String, $kind: String, $tenantId: ID, $limit: Int, $offset: Int) {
		resources(q: $q, kind: $kind, tenantId: $tenantId, limit: $limit, offset: $offset) {
			total
			items { id kind name tenant_id: tenantId owner_id: ownerId attributes created_at: createdAt updated_at: updatedAt }
		}
	}`, vars, &out)
	return out.Resources, err
}

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphQLErrorItem struct {
	Message string `json:"message"`
}

type graphQLResponse struct {
	Data   json.RawMessage    `json:"data"`
	Errors []graphQLErrorItem `json:"errors,omitempty"`
}

func (c *Client) graphQL(ctx context.Context, query string, variables map[string]any, out any) error {
	var response graphQLResponse
	if err := c.do(ctx, http.MethodPost, "/graphql", graphQLRequest{Query: query, Variables: variables}, &response); err != nil {
		return err
	}
	return decodeGraphQLResponse(response, out)
}

func (c *Client) graphQLWithToken(ctx context.Context, query string, variables map[string]any, out any, token string) error {
	var response graphQLResponse
	if err := c.doWithToken(ctx, http.MethodPost, "/graphql", graphQLRequest{Query: query, Variables: variables}, &response, token); err != nil {
		return err
	}
	return decodeGraphQLResponse(response, out)
}

func decodeGraphQLResponse(response graphQLResponse, out any) error {
	if len(response.Errors) > 0 {
		return graphQLErr(response.Errors)
	}
	if out == nil {
		return nil
	}
	if len(response.Data) == 0 {
		return Error{StatusCode: http.StatusInternalServerError, Message: "atom GraphQL response did not include data"}
	}
	return json.Unmarshal(response.Data, out)
}

func graphQLErr(errors []graphQLErrorItem) error {
	messages := make([]string, 0, len(errors))
	for _, err := range errors {
		if err.Message != "" {
			messages = append(messages, err.Message)
		}
	}
	message := strings.Join(messages, "; ")
	lower := strings.ToLower(message)
	status := http.StatusBadRequest
	switch {
	case strings.Contains(lower, "duplicate") || strings.Contains(lower, "already exists") || strings.Contains(lower, "unique"):
		status = http.StatusConflict
	case strings.Contains(lower, "not found"):
		status = http.StatusNotFound
	case strings.Contains(lower, "unauthenticated") || strings.Contains(lower, "authentication"):
		status = http.StatusUnauthorized
	case strings.Contains(lower, "forbidden") || strings.Contains(lower, "authorization") || strings.Contains(lower, "access denied"):
		status = http.StatusForbidden
	}
	return Error{StatusCode: status, Message: message}
}

func tenantCreateInput(tenant Tenant) map[string]any {
	input := map[string]any{"name": tenant.Name}
	setIfNotEmpty(input, "id", tenant.ID)
	setIfNotEmpty(input, "alias", tenant.Route)
	if tenant.Tags != nil {
		input["tags"] = tenant.Tags
	}
	if tenant.Attributes != nil {
		input["attributes"] = tenant.Attributes
	}
	return input
}

func tenantUpdateInput(tenant Tenant) map[string]any {
	input := map[string]any{}
	setIfNotEmpty(input, "name", tenant.Name)
	setIfNotEmpty(input, "alias", tenant.Route)
	if tenant.Tags != nil {
		input["tags"] = tenant.Tags
	}
	if tenant.Attributes != nil {
		input["attributes"] = tenant.Attributes
	}
	return input
}

func entityCreateInput(entity Entity) map[string]any {
	input := map[string]any{"name": entity.Name}
	setIfNotEmpty(input, "id", entity.ID)
	setIfNotEmpty(input, "kind", entity.Kind)
	setIfNotEmpty(input, "tenantId", entity.TenantID)
	if entity.Attributes != nil {
		input["attributes"] = entity.Attributes
	} else {
		input["attributes"] = map[string]any{}
	}
	return input
}

func entityUpdateInput(entity Entity) map[string]any {
	input := map[string]any{}
	setIfNotEmpty(input, "name", entity.Name)
	setIfNotEmpty(input, "status", entity.Status)
	if entity.Attributes != nil {
		input["attributes"] = entity.Attributes
	}
	return input
}

func groupCreateInput(group Group) map[string]any {
	input := map[string]any{"name": group.Name}
	setIfNotEmpty(input, "id", group.ID)
	setIfNotEmpty(input, "tenantId", group.TenantID)
	setIfNotEmpty(input, "description", group.Description)
	if group.Attributes != nil {
		input["attributes"] = group.Attributes
	}
	return input
}

func groupUpdateInput(group Group) map[string]any {
	input := map[string]any{}
	setIfNotEmpty(input, "name", group.Name)
	setIfNotEmpty(input, "description", group.Description)
	setIfNotEmpty(input, "status", group.Status)
	if group.Attributes != nil {
		input["attributes"] = group.Attributes
	}
	return input
}

func resourceCreateInput(resource Resource) map[string]any {
	input := map[string]any{"kind": resource.Kind}
	setIfNotEmpty(input, "id", resource.ID)
	setIfNotEmpty(input, "name", resource.Name)
	setIfNotEmpty(input, "tenantId", resource.TenantID)
	setIfNotEmpty(input, "ownerId", resource.OwnerID)
	if resource.Attributes != nil {
		input["attributes"] = resource.Attributes
	}
	return input
}

func resourceUpdateInput(resource Resource) map[string]any {
	input := map[string]any{}
	setIfNotEmpty(input, "name", resource.Name)
	if resource.Attributes != nil {
		input["attributes"] = resource.Attributes
	}
	return input
}

func authzInput(req AuthzRequest) map[string]any {
	input := map[string]any{
		"subjectId": req.SubjectID,
		"action":    req.Action,
	}
	setIfNotEmpty(input, "resourceId", req.ResourceID)
	setIfNotEmpty(input, "objectKind", req.ObjectKind)
	setIfNotEmpty(input, "objectId", req.ObjectID)
	if req.Context != nil {
		input["context"] = req.Context
	}
	return input
}

func permissionBlockInput(block CreatePermissionBlock) map[string]any {
	input := map[string]any{
		"scopeMode": block.ScopeMode,
		"actionIds": block.ActionIDs,
	}
	setIfNotEmpty(input, "tenantId", block.TenantID)
	setIfNotEmpty(input, "objectKind", block.ObjectKind)
	setIfNotEmpty(input, "objectType", block.ObjectType)
	setIfNotEmpty(input, "objectId", block.ObjectID)
	setIfNotEmpty(input, "groupId", block.GroupID)
	setIfNotEmpty(input, "effect", block.Effect)
	if block.Conditions != nil {
		input["conditions"] = block.Conditions
	}
	return input
}

func directPolicyInput(policy CreateDirectPolicy) map[string]any {
	input := map[string]any{
		"subjectKind":       policy.SubjectKind,
		"subjectId":         policy.SubjectID,
		"permissionBlockId": policy.PermissionBlockID,
	}
	setIfNotEmpty(input, "tenantId", policy.TenantID)
	return input
}

func directPolicyQueryVariables(q DirectPolicyQuery) map[string]any {
	vars := map[string]any{}
	setIfNotEmpty(vars, "tenantId", q.TenantID)
	setIfNotEmpty(vars, "subjectKind", q.SubjectKind)
	setIfNotEmpty(vars, "subjectId", q.SubjectID)
	if q.Limit > 0 {
		vars["limit"] = int(q.Limit)
	}
	if q.Offset > 0 {
		vars["offset"] = int(q.Offset)
	}
	return vars
}

func authorizedObjectIDVariables(q AuthorizedObjectIDsQuery) map[string]any {
	input := map[string]any{
		"subjectId":  q.SubjectID,
		"action":     q.Action,
		"objectKind": q.ObjectKind,
	}
	setIfNotEmpty(input, "objectType", q.ObjectType)
	setIfNotEmpty(input, "tenantId", q.TenantID)
	setIfNotEmpty(input, "q", q.Q)
	if q.Limit > 0 {
		input["limit"] = int(q.Limit)
	}
	if q.Offset > 0 {
		input["offset"] = int(q.Offset)
	}
	return map[string]any{"input": input}
}

func queryVariables(q Query) map[string]any {
	vars := map[string]any{}
	setIfNotEmpty(vars, "q", q.Q)
	setIfNotEmpty(vars, "name", q.Name)
	setIfNotEmpty(vars, "alias", q.Route)
	setIfNotEmpty(vars, "kind", q.Kind)
	setIfNotEmpty(vars, "tenantId", q.TenantID)
	setIfNotEmpty(vars, "status", q.Status)
	if q.Limit > 0 {
		vars["limit"] = int(q.Limit)
	}
	if q.Offset > 0 {
		vars["offset"] = int(q.Offset)
	}
	return vars
}

func objectQueryVariables(q Query) map[string]any {
	vars := map[string]any{}
	setIfNotEmpty(vars, "q", q.Q)
	setIfNotEmpty(vars, "kind", q.Kind)
	setIfNotEmpty(vars, "tenantId", q.TenantID)
	setIfNotEmpty(vars, "status", q.Status)
	if q.Limit > 0 {
		vars["limit"] = int(q.Limit)
	}
	if q.Offset > 0 {
		vars["offset"] = int(q.Offset)
	}
	return vars
}

func setIfNotEmpty(values map[string]any, key, value string) {
	if value != "" {
		values[key] = value
	}
}

func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	token := c.token
	if token == "" && c.adminSecret != "" {
		adminToken, err := c.loginAdmin(ctx)
		if err != nil {
			return err
		}
		token = adminToken
	}
	return c.doWithToken(ctx, method, path, in, out, token)
}

func (c *Client) loginAdmin(ctx context.Context) (string, error) {
	username := c.adminUsername
	if username == "" {
		username = defaultAdminUsername
	}
	resp, err := c.LoginPassword(ctx, username, c.adminSecret)
	if err != nil {
		return "", err
	}
	return resp.Token, nil
}

func (c *Client) doWithToken(ctx context.Context, method, path string, in, out any, token string) error {
	if c.baseURL == "" {
		return Error{StatusCode: 0, Message: "atom URL is empty"}
	}

	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return Error{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(msg))}
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

type Error struct {
	StatusCode int
	Message    string
}

func (e Error) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("atom request failed with status %d", e.StatusCode)
	}
	return fmt.Sprintf("atom request failed with status %d: %s", e.StatusCode, e.Message)
}

func IsConflict(err error) bool {
	ae, ok := err.(Error)
	return ok && ae.StatusCode == http.StatusConflict
}

func IsNotFound(err error) bool {
	ae, ok := err.(Error)
	return ok && ae.StatusCode == http.StatusNotFound
}
