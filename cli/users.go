// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/absmach/magistrala/pkg/atom"
	"github.com/spf13/cobra"
)

// atomKindHuman is the literal Atom wire kind for CreateEntity/ListEntities.
// pkg/atom's own atomKindHuman constant is unexported, so it is duplicated
// here the same way cli/devices.go duplicates atomKindDevice.
const atomKindHuman = "human"

// Operation unique to users. The shared ones (create, get, update, delete,
// enable, disable, all) come from utils.go's block.
const userPassword = "password"

const (
	usageUserCreate   = "cli users create <JSON_user> [workspace_id]"
	usageUserGet      = "cli users <user_id|all> get [workspace_id]"
	usageUserUpdate   = "cli users <user_id> update <JSON_string>"
	usageUserDelete   = "cli users <user_id> delete"
	usageUserEnable   = "cli users <user_id> enable"
	usageUserDisable  = "cli users <user_id> disable"
	usageUserPassword = "cli users <user_id> password set <new_password>"
)

// NewUsersCmd wraps pkg/atom's generic entity API for entities of kind
// "human" — see cli/devices.go for the same shape applied to devices. A
// human is created without a password (createEntity has no password field);
// "password set" is the follow-up that makes an admin-created account
// loginable, wrapping atom.Client.CreatePassword.
//
// Self-service sign-up and login/logout/session-refresh are not covered
// here: they are unauthenticated or session-scoped operations, not entity
// management, and stay GraphQL-only per the API reference.
func NewUsersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "users <user_id|all|create> [operation] [args...]",
		Short: "Users management",
		Long: `Format:
  users create <JSON_user> [workspace_id]
  users <user_id|all> <operation> [args...]

Operations (require user_id/all): get, update, delete, enable, disable, password

A user is an Atom entity of kind "human". workspace_id is optional on create
and "all get" — omit it to create/list a user with no workspace membership.

createEntity has no password field, so a newly created user cannot log in
until a password is set:
  users <user_id> password set <new_password>

Self-service sign-up, login, logout and session refresh are GraphQL-only —
see the API reference — since they are not entity management.

Examples:
  users create '{"name":"Jane Doe"}' <workspace_id>
  users all get <workspace_id>
  users <user_id> get
  users <user_id> update '{"name":"Jane Doe"}'
  users <user_id> delete
  users <user_id> enable
  users <user_id> disable
  users <user_id> password set <new_password>`,

		Run: func(cmd *cobra.Command, args []string) {
			if !requireAtomClient(cmd) {
				return
			}
			if len(args) == 0 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			if args[0] == create {
				handleUserCreate(cmd, args[1:])
				return
			}

			if len(args) < 2 {
				logUsageCmd(*cmd, "users <user_id|all> <get|update|delete|enable|disable|password> [args...]")
				return
			}

			userID := args[0]
			operation := args[1]
			opArgs := args[2:]

			switch operation {
			case get:
				handleUserGet(cmd, userID, opArgs)
			case update:
				handleUserUpdate(cmd, userID, opArgs)
			case delete:
				handleUserDelete(cmd, userID, opArgs)
			case enable:
				handleUserEnable(cmd, userID, opArgs)
			case disable:
				handleUserDisable(cmd, userID, opArgs)
			case userPassword:
				handleUserPassword(cmd, userID, opArgs)
			default:
				logErrorCmd(*cmd, fmt.Errorf("unknown operation: %s", operation))
			}
		},
	}

	return cmd
}

func handleUserCreate(cmd *cobra.Command, args []string) {
	if len(args) < 1 || len(args) > 2 {
		logUsageCmd(*cmd, usageUserCreate)
		return
	}

	var user atom.Entity
	if err := json.Unmarshal([]byte(args[0]), &user); err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	user.Kind = atomKindHuman
	if len(args) == 2 {
		user.TenantID = args[1]
	}

	created, err := atomClient.CreateEntity(cmd.Context(), user)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, created)
}

// handleUserGet's "all" branch lists users, optionally scoped to a
// workspace — unlike devices, a tenant-less listing is a legitimate query
// here since a self-registered user may not belong to any workspace yet.
func handleUserGet(cmd *cobra.Command, userID string, args []string) {
	if userID == all {
		if len(args) > 1 {
			logUsageCmd(*cmd, usageUserGet)
			return
		}

		tenantID := ""
		if len(args) == 1 {
			tenantID = args[0]
		}

		q := atom.Query{
			Kind:     atomKindHuman,
			TenantID: tenantID,
			Q:        Name,
			Limit:    Limit,
			Offset:   Offset,
		}
		l, err := atomClient.ListEntities(cmd.Context(), q)
		if err != nil {
			logErrorCmd(*cmd, err)
			return
		}

		logJSONCmd(*cmd, l)
		return
	}

	if len(args) != 0 {
		logUsageCmd(*cmd, usageUserGet)
		return
	}

	u, err := atomClient.GetEntity(cmd.Context(), userID)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, u)
}

func handleUserUpdate(cmd *cobra.Command, userID string, args []string) {
	if len(args) != 1 {
		logUsageCmd(*cmd, usageUserUpdate)
		return
	}

	var user atom.Entity
	if err := json.Unmarshal([]byte(args[0]), &user); err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	updated, err := atomClient.UpdateEntity(cmd.Context(), userID, user)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, updated)
}

func handleUserDelete(cmd *cobra.Command, userID string, args []string) {
	if len(args) != 0 {
		logUsageCmd(*cmd, usageUserDelete)
		return
	}

	if err := atomClient.DeleteEntity(cmd.Context(), userID); err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logOKCmd(*cmd)
}

func handleUserEnable(cmd *cobra.Command, userID string, args []string) {
	if len(args) != 0 {
		logUsageCmd(*cmd, usageUserEnable)
		return
	}

	u, err := atomClient.UpdateEntity(cmd.Context(), userID, atom.Entity{Status: atomEntityStatusActive})
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, u)
}

func handleUserDisable(cmd *cobra.Command, userID string, args []string) {
	if len(args) != 0 {
		logUsageCmd(*cmd, usageUserDisable)
		return
	}

	u, err := atomClient.UpdateEntity(cmd.Context(), userID, atom.Entity{Status: atomEntityStatusInactive})
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, u)
}

// handleUserPassword only implements "set": createPassword is also how an
// existing password gets replaced (see the API reference's Set/Change
// Password section) — there is no separate update mutation to wrap.
func handleUserPassword(cmd *cobra.Command, userID string, args []string) {
	if len(args) != 2 || args[0] != "set" {
		logUsageCmd(*cmd, usageUserPassword)
		return
	}

	if err := atomClient.CreatePassword(cmd.Context(), userID, args[1]); err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logOKCmd(*cmd)
}
