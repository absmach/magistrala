// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import "github.com/spf13/cobra"

// Command names, also used by the GraphQL responses that carry the same name.
const (
	cmdDomains  = "domains"
	cmdClients  = "clients"
	cmdChannels = "channels"
	cmdGroups   = "groups"
)

// GraphQL variable and input object keys.
const (
	varInput       = "input"
	varID          = "id"
	varTenantID    = "tenantId"
	varKind        = "kind"
	varLimit       = "limit"
	varOffset      = "offset"
	varName        = "name"
	varAlias       = "alias"
	varAttributes  = "attributes"
	varDescription = "description"
	varGroupType   = "groupType"
	varOwnerID     = "ownerId"
)

// Top-level response fields selected out of the GraphQL data object.
const (
	respLogin          = "login"
	respTenant         = "tenant"
	respTenants        = "tenants"
	respCreateTenant   = "createTenant"
	respEntity         = "entity"
	respEntities       = "entities"
	respCreateEntity   = "createEntity"
	respResource       = "resource"
	respResources      = "resources"
	respCreateResource = "createResource"
	respGroup          = "group"
	respCreateGroup    = "createGroup"
	respAuthzCheck     = "authzCheck"
)

const (
	useList           = "list"
	useCreateInDomain = "create <domain_id> <name>"
	useListInDomain   = "list <domain_id>"

	defaultEntityKind   = "device"
	defaultResourceKind = "channel"
)

const (
	tenantFields   = "id name alias status tags attributes createdAt updatedAt"
	entityFields   = "id kind profileId profileVersionId name alias externalId tenantId objectGroupIds status attributes createdAt updatedAt"
	resourceFields = "id kind name alias tenantId ownerId objectGroupIds attributes createdAt updatedAt"
	groupFields    = "id name tenantId groupType description parentId status attributes createdAt updatedAt"
)

const (
	loginMutation = `mutation Login($input: LoginInput!) {
  login(input: $input) {
    token
    entityId
    sessionId
    expiresAt
    emailVerified
    verificationRequired
  }
}`

	getDomainQuery = `query GetDomain($id: ID!) {
  tenant(id: $id) { ` + tenantFields + ` }
}`

	listDomainsQuery = `query ListDomains($limit: Int, $offset: Int) {
  tenants(limit: $limit, offset: $offset) {
    total
    items { ` + tenantFields + ` }
  }
}`

	createDomainMutation = `mutation CreateDomain($input: CreateTenantInput!) {
  createTenant(input: $input) { ` + tenantFields + ` }
}`

	getClientQuery = `query GetClient($id: ID!) {
  entity(id: $id) { ` + entityFields + ` }
}`

	listClientsQuery = `query ListClients($tenantId: ID, $kind: EntityKind, $limit: Int, $offset: Int) {
  entities(tenantId: $tenantId, kind: $kind, limit: $limit, offset: $offset) {
    total
    items { ` + entityFields + ` }
  }
}`

	createClientMutation = `mutation CreateClient($input: CreateEntityInput!) {
  createEntity(input: $input) { ` + entityFields + ` }
}`

	getChannelQuery = `query GetChannel($id: ID!) {
  resource(id: $id) { ` + resourceFields + ` }
}`

	listChannelsQuery = `query ListChannels($tenantId: ID, $kind: String, $limit: Int, $offset: Int) {
  resources(tenantId: $tenantId, kind: $kind, limit: $limit, offset: $offset) {
    total
    items { ` + resourceFields + ` }
  }
}`

	createChannelMutation = `mutation CreateChannel($input: CreateResourceInput!) {
  createResource(input: $input) { ` + resourceFields + ` }
}`

	getGroupQuery = `query GetGroup($id: ID!) {
  group(id: $id) { ` + groupFields + ` }
}`

	listGroupsQuery = `query ListGroups($tenantId: ID, $limit: Int, $offset: Int) {
  groups(tenantId: $tenantId, limit: $limit, offset: $offset) {
    total
    items { ` + groupFields + ` }
  }
}`

	createGroupMutation = `mutation CreateGroup($input: CreateGroupInput!) {
  createGroup(input: $input) { ` + groupFields + ` }
}`

	authzCheckMutation = `mutation AuthzCheck($input: AuthzCheckInput!) {
  authzCheck(input: $input) {
    allowed
    reason
    details
  }
}`
)

// gqlCmdConfig describes one Atom GraphQL-backed CLI command.
type gqlCmdConfig struct {
	use   string
	short string
	args  cobra.PositionalArgs
	query string
	// field is the top-level GraphQL response field to print.
	field string
	// anonymous commands run without a bearer token; every other command
	// requires one.
	anonymous bool
	// vars builds the GraphQL variables from the positional arguments. It runs
	// after flag parsing, so it may read flag-backed values.
	vars func(args []string) (map[string]any, error)
	// flags registers command specific flags.
	flags func(cmd *cobra.Command)
}

func newGQLCmd(opts *rootOptions, cfg gqlCmdConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   cfg.use,
		Short: cfg.short,
		Args:  cfg.args,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Arguments and flags already parsed, so any failure past this
			// point is a runtime one and printing usage would only be noise.
			cmd.SilenceUsage = true

			client := opts.client()
			if !cfg.anonymous {
				var err error
				if client, err = opts.authedClient(); err != nil {
					return err
				}
			}

			variables, err := cfg.vars(args)
			if err != nil {
				return err
			}

			value, err := client.do(cmd.Context(), cfg.query, variables, cfg.field)
			if err != nil {
				return err
			}

			return writeOutput(cmd, value)
		},
	}
	if cfg.flags != nil {
		cfg.flags(cmd)
	}

	return cmd
}

// newGetCmd builds the shared "get one object by ID" command.
func newGetCmd(opts *rootOptions, use, short, query, field string) *cobra.Command {
	return newGQLCmd(opts, gqlCmdConfig{
		use:   use,
		short: short,
		args:  cobra.ExactArgs(1),
		query: query,
		field: field,
		vars: func(args []string) (map[string]any, error) {
			return map[string]any{varID: args[0]}, nil
		},
	})
}

func newLoginCmd(opts *rootOptions) *cobra.Command {
	var tenantID, tenantAlias, kind string

	return newGQLCmd(opts, gqlCmdConfig{
		use:       "login <identifier> <secret>",
		short:     "Authenticate against Atom",
		args:      cobra.ExactArgs(2),
		query:     loginMutation,
		field:     respLogin,
		anonymous: true,
		vars: func(args []string) (map[string]any, error) {
			input := gqlInput{"identifier": args[0], "secret": args[1]}
			input.setString(varKind, kind)
			input.setString(varTenantID, tenantID)
			input.setString("tenantAlias", tenantAlias)

			return map[string]any{varInput: input}, nil
		},
		flags: func(cmd *cobra.Command) {
			cmd.Flags().StringVar(&tenantID, "tenant-id", "", "tenant ID for tenant-scoped credentials")
			cmd.Flags().StringVar(&tenantAlias, "tenant-alias", "", "tenant alias for tenant-scoped credentials")
			cmd.Flags().StringVar(&kind, varKind, "password", "credential kind")
		},
	})
}

func newDomainsCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: cmdDomains, Short: "Manage Magistrala domains through Atom tenants"}
	cmd.AddCommand(
		newDomainCreateCmd(opts),
		newDomainListCmd(opts),
		newGetCmd(opts, "get <domain_id>", "Get a domain", getDomainQuery, respTenant),
	)

	return cmd
}

func newDomainCreateCmd(opts *rootOptions) *cobra.Command {
	var alias, attrs string

	return newGQLCmd(opts, gqlCmdConfig{
		use:   "create <name>",
		short: "Create a domain",
		args:  cobra.ExactArgs(1),
		query: createDomainMutation,
		field: respCreateTenant,
		vars: func(args []string) (map[string]any, error) {
			attributes, err := parseJSONFlag(attrs)
			if err != nil {
				return nil, err
			}
			input := gqlInput{varName: args[0]}
			input.setString(varAlias, alias)
			input.setObject(varAttributes, attributes)

			return map[string]any{varInput: input}, nil
		},
		flags: func(cmd *cobra.Command) {
			cmd.Flags().StringVar(&alias, varAlias, "", "domain alias")
			cmd.Flags().StringVar(&attrs, varAttributes, "", "JSON attributes")
		},
	})
}

func newDomainListCmd(opts *rootOptions) *cobra.Command {
	return newGQLCmd(opts, gqlCmdConfig{
		use:   useList,
		short: "List domains",
		args:  cobra.NoArgs,
		query: listDomainsQuery,
		field: respTenants,
		vars: func(_ []string) (map[string]any, error) {
			return listVariables(), nil
		},
	})
}

func newGraphQLClientsCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: cmdClients, Short: "Manage Magistrala clients through Atom entities"}
	cmd.AddCommand(
		newClientCreateCmd(opts),
		newClientListCmd(opts),
		newGetCmd(opts, "get <client_id>", "Get a client", getClientQuery, respEntity),
	)

	return cmd
}

func newClientCreateCmd(opts *rootOptions) *cobra.Command {
	var alias, kind, attrs string

	return newGQLCmd(opts, gqlCmdConfig{
		use:   useCreateInDomain,
		short: "Create a client",
		args:  cobra.ExactArgs(2),
		query: createClientMutation,
		field: respCreateEntity,
		vars: func(args []string) (map[string]any, error) {
			attributes, err := parseJSONFlag(attrs)
			if err != nil {
				return nil, err
			}
			input := gqlInput{varTenantID: args[0], varName: args[1]}
			input.setString(varKind, kind)
			input.setString(varAlias, alias)
			input.setObject(varAttributes, attributes)

			return map[string]any{varInput: input}, nil
		},
		flags: func(cmd *cobra.Command) {
			cmd.Flags().StringVar(&alias, varAlias, "", "client alias")
			cmd.Flags().StringVar(&kind, varKind, defaultEntityKind, "Atom EntityKind value")
			cmd.Flags().StringVar(&attrs, varAttributes, "", "JSON attributes")
		},
	})
}

func newClientListCmd(opts *rootOptions) *cobra.Command {
	var kind string

	return newGQLCmd(opts, gqlCmdConfig{
		use:   useListInDomain,
		short: "List clients",
		args:  cobra.ExactArgs(1),
		query: listClientsQuery,
		field: respEntities,
		vars: func(args []string) (map[string]any, error) {
			variables := listVariables()
			variables[varTenantID] = args[0]
			// An empty kind is not a valid EntityKind, so it means "no filter".
			if kind != "" {
				variables[varKind] = kind
			}

			return variables, nil
		},
		flags: func(cmd *cobra.Command) {
			cmd.Flags().StringVar(&kind, varKind, defaultEntityKind, "Atom EntityKind filter, empty lists every kind")
		},
	})
}

func newGraphQLChannelsCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: cmdChannels, Short: "Manage Magistrala channels through Atom resources"}
	cmd.AddCommand(
		newChannelCreateCmd(opts),
		newChannelListCmd(opts),
		newGetCmd(opts, "get <channel_id>", "Get a channel", getChannelQuery, respResource),
	)

	return cmd
}

func newChannelCreateCmd(opts *rootOptions) *cobra.Command {
	var alias, kind, ownerID, attrs string

	return newGQLCmd(opts, gqlCmdConfig{
		use:   useCreateInDomain,
		short: "Create a channel",
		args:  cobra.ExactArgs(2),
		query: createChannelMutation,
		field: respCreateResource,
		vars: func(args []string) (map[string]any, error) {
			attributes, err := parseJSONFlag(attrs)
			if err != nil {
				return nil, err
			}
			// kind is non-null in CreateResourceInput, so it is always sent.
			input := gqlInput{varTenantID: args[0], varName: args[1], varKind: kind}
			input.setString(varAlias, alias)
			input.setString(varOwnerID, ownerID)
			input.setObject(varAttributes, attributes)

			return map[string]any{varInput: input}, nil
		},
		flags: func(cmd *cobra.Command) {
			cmd.Flags().StringVar(&alias, varAlias, "", "channel alias")
			cmd.Flags().StringVar(&kind, varKind, defaultResourceKind, "Atom resource kind")
			cmd.Flags().StringVar(&ownerID, "owner-id", "", "owner entity ID")
			cmd.Flags().StringVar(&attrs, varAttributes, "", "JSON attributes")
		},
	})
}

func newChannelListCmd(opts *rootOptions) *cobra.Command {
	var kind string

	return newGQLCmd(opts, gqlCmdConfig{
		use:   useListInDomain,
		short: "List channels",
		args:  cobra.ExactArgs(1),
		query: listChannelsQuery,
		field: respResources,
		vars: func(args []string) (map[string]any, error) {
			variables := listVariables()
			variables[varTenantID] = args[0]
			// Filtering on an empty kind would match nothing.
			if kind != "" {
				variables[varKind] = kind
			}

			return variables, nil
		},
		flags: func(cmd *cobra.Command) {
			cmd.Flags().StringVar(&kind, varKind, defaultResourceKind, "Atom resource kind filter, empty lists every kind")
		},
	})
}

func newGraphQLGroupsCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: cmdGroups, Short: "Manage Magistrala groups through Atom groups"}
	cmd.AddCommand(
		newGroupCreateCmd(opts),
		newGroupListCmd(opts),
		newGetCmd(opts, "get <group_id>", "Get a group", getGroupQuery, respGroup),
	)

	return cmd
}

func newGroupCreateCmd(opts *rootOptions) *cobra.Command {
	var groupType, description, attrs string

	return newGQLCmd(opts, gqlCmdConfig{
		use:   useCreateInDomain,
		short: "Create a group",
		args:  cobra.ExactArgs(2),
		query: createGroupMutation,
		field: respCreateGroup,
		vars: func(args []string) (map[string]any, error) {
			attributes, err := parseJSONFlag(attrs)
			if err != nil {
				return nil, err
			}
			input := gqlInput{varTenantID: args[0], varName: args[1]}
			input.setString(varGroupType, groupType)
			input.setString(varDescription, description)
			input.setObject(varAttributes, attributes)

			return map[string]any{varInput: input}, nil
		},
		flags: func(cmd *cobra.Command) {
			cmd.Flags().StringVar(&groupType, "type", "", "group type")
			cmd.Flags().StringVar(&description, varDescription, "", "group description")
			cmd.Flags().StringVar(&attrs, varAttributes, "", "JSON attributes")
		},
	})
}

func newGroupListCmd(opts *rootOptions) *cobra.Command {
	return newGQLCmd(opts, gqlCmdConfig{
		use:   useListInDomain,
		short: "List groups",
		args:  cobra.ExactArgs(1),
		query: listGroupsQuery,
		field: cmdGroups,
		vars: func(args []string) (map[string]any, error) {
			variables := listVariables()
			variables[varTenantID] = args[0]

			return variables, nil
		},
	})
}

func newAuthzCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "authz", Short: "Run Atom authorization checks"}
	cmd.AddCommand(newAuthzCheckCmd(opts))

	return cmd
}

func newAuthzCheckCmd(opts *rootOptions) *cobra.Command {
	var objectKind, objectID, resourceID, contextJSON string

	return newGQLCmd(opts, gqlCmdConfig{
		use:   "check <subject_id> <action>",
		short: "Check whether a subject can perform an action",
		args:  cobra.ExactArgs(2),
		query: authzCheckMutation,
		field: respAuthzCheck,
		vars: func(args []string) (map[string]any, error) {
			contextValue, err := parseJSONFlag(contextJSON)
			if err != nil {
				return nil, err
			}
			input := gqlInput{"subjectId": args[0], "action": args[1]}
			input.setString("resourceId", resourceID)
			input.setString("objectKind", objectKind)
			input.setString("objectId", objectID)
			input.setObject("context", contextValue)

			return map[string]any{varInput: input}, nil
		},
		flags: func(cmd *cobra.Command) {
			cmd.Flags().StringVar(&objectKind, "object-kind", "", "object kind")
			cmd.Flags().StringVar(&objectID, "object-id", "", "object ID")
			cmd.Flags().StringVar(&resourceID, "resource-id", "", "resource ID")
			cmd.Flags().StringVar(&contextJSON, "context", "", "JSON context")
		},
	})
}
