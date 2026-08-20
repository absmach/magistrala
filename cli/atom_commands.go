// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

const (
	tenantFields   = "id name alias status tags attributes createdAt updatedAt"
	entityFields   = "id kind profileId profileVersionId name alias tenantId objectGroupIds status attributes createdAt updatedAt"
	resourceFields = "id kind name alias tenantId ownerId objectGroupIds attributes createdAt updatedAt"
	groupFields    = "id name tenantId groupType description parentId status attributes createdAt updatedAt"
)

func newLoginCmd(opts *rootOptions) *cobra.Command {
	var tenantID, tenantAlias, kind string
	cmd := &cobra.Command{
		Use:   "login <identifier> <secret>",
		Short: "Authenticate against Atom",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := map[string]any{
				"identifier":  args[0],
				"secret":      args[1],
				"kind":        kind,
				"tenantId":    optionalString(tenantID),
				"tenantAlias": optionalString(tenantAlias),
			}
			var out struct {
				Login map[string]any `json:"login"`
			}
			err := opts.client().do(context.Background(), `
mutation Login($input: LoginInput!) {
  login(input: $input) {
    token
    entityId
    sessionId
    expiresAt
    emailVerified
    verificationRequired
  }
}`, map[string]any{"input": input}, &out)
			if err != nil {
				return err
			}
			return writeOutput(cmd, out.Login)
		},
	}
	cmd.Flags().StringVar(&tenantID, "tenant-id", "", "tenant ID for tenant-scoped credentials")
	cmd.Flags().StringVar(&tenantAlias, "tenant-alias", "", "tenant alias for tenant-scoped credentials")
	cmd.Flags().StringVar(&kind, "kind", "password", "credential kind")
	return cmd
}

func newDomainsCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "domains", Short: "Manage Magistrala domains through Atom tenants"}
	cmd.AddCommand(newDomainCreateCmd(opts), newDomainListCmd(opts), newDomainGetCmd(opts))
	return cmd
}

func newDomainCreateCmd(opts *rootOptions) *cobra.Command {
	var alias, attrs string
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.authedClient()
			if err != nil {
				return err
			}
			attributes, err := parseJSONFlag(attrs)
			if err != nil {
				return err
			}
			var out struct {
				CreateTenant map[string]any `json:"createTenant"`
			}
			err = client.do(context.Background(), `
mutation CreateDomain($input: CreateTenantInput!) {
  createTenant(input: $input) { `+tenantFields+` }
}`, map[string]any{"input": map[string]any{
				"name":       args[0],
				"alias":      optionalString(alias),
				"attributes": optionalObject(attributes),
			}}, &out)
			if err != nil {
				return err
			}
			return writeOutput(cmd, out.CreateTenant)
		},
	}
	cmd.Flags().StringVar(&alias, "alias", "", "domain alias")
	cmd.Flags().StringVar(&attrs, "attributes", "", "JSON attributes")
	return cmd
}

func newDomainListCmd(opts *rootOptions) *cobra.Command {
	var limit, offset int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List domains",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.authedClient()
			if err != nil {
				return err
			}
			var out struct {
				Tenants map[string]any `json:"tenants"`
			}
			err = client.do(context.Background(), `
query ListDomains($limit: Int, $offset: Int) {
  tenants(limit: $limit, offset: $offset) {
    total
    items { `+tenantFields+` }
  }
}`, map[string]any{"limit": limit, "offset": offset}, &out)
			if err != nil {
				return err
			}
			return writeOutput(cmd, out.Tenants)
		},
	}
	commonListFlags(cmd, &limit, &offset)
	return cmd
}

func newDomainGetCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <domain_id>",
		Short: "Get a domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.authedClient()
			if err != nil {
				return err
			}
			var out struct {
				Tenant map[string]any `json:"tenant"`
			}
			err = client.do(context.Background(), `
query GetDomain($id: ID!) {
  tenant(id: $id) { `+tenantFields+` }
}`, map[string]any{"id": args[0]}, &out)
			if err != nil {
				return err
			}
			return writeOutput(cmd, out.Tenant)
		},
	}
	return cmd
}

func newGraphQLClientsCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "clients", Short: "Manage Magistrala clients through Atom entities"}
	cmd.AddCommand(newClientCreateCmd(opts), newClientListCmd(opts), newClientGetCmd(opts))
	return cmd
}

func newClientCreateCmd(opts *rootOptions) *cobra.Command {
	var alias, kind, attrs string
	cmd := &cobra.Command{
		Use:   "create <domain_id> <name>",
		Short: "Create a client",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.authedClient()
			if err != nil {
				return err
			}
			attributes, err := parseJSONFlag(attrs)
			if err != nil {
				return err
			}
			var out struct {
				CreateEntity map[string]any `json:"createEntity"`
			}
			err = client.do(context.Background(), `
mutation CreateClient($input: CreateEntityInput!) {
  createEntity(input: $input) { `+entityFields+` }
}`, map[string]any{"input": map[string]any{
				"tenantId":   args[0],
				"name":       args[1],
				"kind":       kind,
				"alias":      optionalString(alias),
				"attributes": optionalObject(attributes),
			}}, &out)
			if err != nil {
				return err
			}
			return writeOutput(cmd, out.CreateEntity)
		},
	}
	cmd.Flags().StringVar(&alias, "alias", "", "client alias")
	cmd.Flags().StringVar(&kind, "kind", "device", "Atom EntityKind value")
	cmd.Flags().StringVar(&attrs, "attributes", "", "JSON attributes")
	return cmd
}

func newClientListCmd(opts *rootOptions) *cobra.Command {
	var limit, offset int
	var kind string
	cmd := &cobra.Command{
		Use:   "list <domain_id>",
		Short: "List clients",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.authedClient()
			if err != nil {
				return err
			}
			var out struct {
				Entities map[string]any `json:"entities"`
			}
			err = client.do(context.Background(), `
query ListClients($tenantId: ID!, $kind: EntityKind, $limit: Int, $offset: Int) {
  entities(tenantId: $tenantId, kind: $kind, limit: $limit, offset: $offset) {
    total
    items { `+entityFields+` }
  }
}`, map[string]any{"tenantId": args[0], "kind": kind, "limit": limit, "offset": offset}, &out)
			if err != nil {
				return err
			}
			return writeOutput(cmd, out.Entities)
		},
	}
	cmd.Flags().StringVar(&kind, "kind", "device", "Atom EntityKind value")
	commonListFlags(cmd, &limit, &offset)
	return cmd
}

func newClientGetCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <client_id>",
		Short: "Get a client",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.authedClient()
			if err != nil {
				return err
			}
			var out struct {
				Entity map[string]any `json:"entity"`
			}
			err = client.do(context.Background(), `
query GetClient($id: ID!) {
  entity(id: $id) { `+entityFields+` }
}`, map[string]any{"id": args[0]}, &out)
			if err != nil {
				return err
			}
			return writeOutput(cmd, out.Entity)
		},
	}
	return cmd
}

func newGraphQLChannelsCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "channels", Short: "Manage Magistrala channels through Atom resources"}
	cmd.AddCommand(newChannelCreateCmd(opts), newChannelListCmd(opts), newChannelGetCmd(opts))
	return cmd
}

func newChannelCreateCmd(opts *rootOptions) *cobra.Command {
	var alias, kind, ownerID, attrs string
	cmd := &cobra.Command{
		Use:   "create <domain_id> <name>",
		Short: "Create a channel",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.authedClient()
			if err != nil {
				return err
			}
			attributes, err := parseJSONFlag(attrs)
			if err != nil {
				return err
			}
			var out struct {
				CreateResource map[string]any `json:"createResource"`
			}
			err = client.do(context.Background(), `
mutation CreateChannel($input: CreateResourceInput!) {
  createResource(input: $input) { `+resourceFields+` }
}`, map[string]any{"input": map[string]any{
				"tenantId":   args[0],
				"name":       args[1],
				"kind":       kind,
				"alias":      optionalString(alias),
				"ownerId":    optionalString(ownerID),
				"attributes": optionalObject(attributes),
			}}, &out)
			if err != nil {
				return err
			}
			return writeOutput(cmd, out.CreateResource)
		},
	}
	cmd.Flags().StringVar(&alias, "alias", "", "channel alias")
	cmd.Flags().StringVar(&kind, "kind", "channel", "Atom resource kind")
	cmd.Flags().StringVar(&ownerID, "owner-id", "", "owner entity ID")
	cmd.Flags().StringVar(&attrs, "attributes", "", "JSON attributes")
	return cmd
}

func newChannelListCmd(opts *rootOptions) *cobra.Command {
	var limit, offset int
	var kind string
	cmd := &cobra.Command{
		Use:   "list <domain_id>",
		Short: "List channels",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.authedClient()
			if err != nil {
				return err
			}
			var out struct {
				Resources map[string]any `json:"resources"`
			}
			err = client.do(context.Background(), `
query ListChannels($tenantId: ID!, $kind: String, $limit: Int, $offset: Int) {
  resources(tenantId: $tenantId, kind: $kind, limit: $limit, offset: $offset) {
    total
    items { `+resourceFields+` }
  }
}`, map[string]any{"tenantId": args[0], "kind": kind, "limit": limit, "offset": offset}, &out)
			if err != nil {
				return err
			}
			return writeOutput(cmd, out.Resources)
		},
	}
	cmd.Flags().StringVar(&kind, "kind", "channel", "Atom resource kind")
	commonListFlags(cmd, &limit, &offset)
	return cmd
}

func newChannelGetCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <channel_id>",
		Short: "Get a channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.authedClient()
			if err != nil {
				return err
			}
			var out struct {
				Resource map[string]any `json:"resource"`
			}
			err = client.do(context.Background(), `
query GetChannel($id: ID!) {
  resource(id: $id) { `+resourceFields+` }
}`, map[string]any{"id": args[0]}, &out)
			if err != nil {
				return err
			}
			return writeOutput(cmd, out.Resource)
		},
	}
	return cmd
}

func newGraphQLGroupsCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "groups", Short: "Manage Magistrala groups through Atom groups"}
	cmd.AddCommand(newGroupCreateCmd(opts), newGroupListCmd(opts), newGroupGetCmd(opts))
	return cmd
}

func newGroupCreateCmd(opts *rootOptions) *cobra.Command {
	var groupType, description, attrs string
	cmd := &cobra.Command{
		Use:   "create <domain_id> <name>",
		Short: "Create a group",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.authedClient()
			if err != nil {
				return err
			}
			attributes, err := parseJSONFlag(attrs)
			if err != nil {
				return err
			}
			mutationName := "createObjectGroup"
			switch groupType {
			case "object":
			case "principal":
				mutationName = "createPrincipalGroup"
			default:
				return fmt.Errorf("unsupported group type %q: expected object or principal", groupType)
			}
			var out map[string]map[string]any
			err = client.do(context.Background(), `
mutation CreateGroup($input: CreateGroupInput!) {
  `+mutationName+`(input: $input) { `+groupFields+` }
}`, map[string]any{"input": map[string]any{
				"tenantId":    args[0],
				"name":        args[1],
				"description": optionalString(description),
				"attributes":  optionalObject(attributes),
			}}, &out)
			if err != nil {
				return err
			}
			return writeOutput(cmd, out[mutationName])
		},
	}
	cmd.Flags().StringVar(&groupType, "type", "object", "group type: object or principal")
	cmd.Flags().StringVar(&description, "description", "", "group description")
	cmd.Flags().StringVar(&attrs, "attributes", "", "JSON attributes")
	return cmd
}

func newGroupListCmd(opts *rootOptions) *cobra.Command {
	var limit, offset int
	cmd := &cobra.Command{
		Use:   "list <domain_id>",
		Short: "List groups",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.authedClient()
			if err != nil {
				return err
			}
			var out struct {
				Groups map[string]any `json:"groups"`
			}
			err = client.do(context.Background(), `
query ListGroups($tenantId: ID!, $limit: Int, $offset: Int) {
  groups(tenantId: $tenantId, limit: $limit, offset: $offset) {
    total
    items { `+groupFields+` }
  }
}`, map[string]any{"tenantId": args[0], "limit": limit, "offset": offset}, &out)
			if err != nil {
				return err
			}
			return writeOutput(cmd, out.Groups)
		},
	}
	commonListFlags(cmd, &limit, &offset)
	return cmd
}

func newGroupGetCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <group_id>",
		Short: "Get a group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.authedClient()
			if err != nil {
				return err
			}
			var out struct {
				Group map[string]any `json:"group"`
			}
			err = client.do(context.Background(), `
query GetGroup($id: ID!) {
  group(id: $id) { `+groupFields+` }
}`, map[string]any{"id": args[0]}, &out)
			if err != nil {
				return err
			}
			return writeOutput(cmd, out.Group)
		},
	}
	return cmd
}

func newAuthzCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "authz", Short: "Run Atom authorization checks"}
	cmd.AddCommand(newAuthzCheckCmd(opts))
	return cmd
}

func newAuthzCheckCmd(opts *rootOptions) *cobra.Command {
	var objectKind, objectID, resourceID, contextJSON string
	cmd := &cobra.Command{
		Use:   "check <subject_id> <action>",
		Short: "Check whether a subject can perform an action",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.authedClient()
			if err != nil {
				return err
			}
			contextValue, err := parseJSONFlag(contextJSON)
			if err != nil {
				return err
			}
			var out struct {
				AuthzCheck map[string]any `json:"authzCheck"`
			}
			err = client.do(context.Background(), `
mutation AuthzCheck($input: AuthzCheckInput!) {
  authzCheck(input: $input) {
    allowed
    reason
    details
  }
}`, map[string]any{"input": map[string]any{
				"subjectId":  args[0],
				"action":     args[1],
				"resourceId": optionalString(resourceID),
				"objectKind": optionalString(objectKind),
				"objectId":   optionalString(objectID),
				"context":    optionalObject(contextValue),
			}}, &out)
			if err != nil {
				return err
			}
			return writeOutput(cmd, out.AuthzCheck)
		},
	}
	cmd.Flags().StringVar(&objectKind, "object-kind", "", "object kind")
	cmd.Flags().StringVar(&objectID, "object-id", "", "object ID")
	cmd.Flags().StringVar(&resourceID, "resource-id", "", "resource ID")
	cmd.Flags().StringVar(&contextJSON, "context", "", "JSON context")
	return cmd
}
