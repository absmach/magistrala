// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
	"fmt"
)

var magistralaActionDescriptions = map[string]string{
	"read":      "Read / view an object",
	"write":     "Create or update an object",
	"delete":    "Delete an object",
	"manage":    "Full administrative control",
	"publish":   "Publish messages to a channel",
	"subscribe": "Subscribe to channel messages",
	"execute":   "Execute a command or action",
}

var magistralaActionApplicability = []CapabilityApplicabilitySpec{
	{ActionName: "read", ObjectKind: "resource", ObjectType: "resource:channel"},
	{ActionName: "write", ObjectKind: "resource", ObjectType: "resource:channel"},
	{ActionName: "delete", ObjectKind: "resource", ObjectType: "resource:channel"},
	{ActionName: "manage", ObjectKind: "resource", ObjectType: "resource:channel"},
	{ActionName: "publish", ObjectKind: "resource", ObjectType: "resource:channel"},
	{ActionName: "subscribe", ObjectKind: "resource", ObjectType: "resource:channel"},

	{ActionName: "read", ObjectKind: "resource", ObjectType: "resource:rule"},
	{ActionName: "write", ObjectKind: "resource", ObjectType: "resource:rule"},
	{ActionName: "delete", ObjectKind: "resource", ObjectType: "resource:rule"},
	{ActionName: "manage", ObjectKind: "resource", ObjectType: "resource:rule"},
	{ActionName: "execute", ObjectKind: "resource", ObjectType: "resource:rule"},

	{ActionName: "read", ObjectKind: "resource", ObjectType: "resource:report"},
	{ActionName: "write", ObjectKind: "resource", ObjectType: "resource:report"},
	{ActionName: "delete", ObjectKind: "resource", ObjectType: "resource:report"},
	{ActionName: "manage", ObjectKind: "resource", ObjectType: "resource:report"},
	{ActionName: "execute", ObjectKind: "resource", ObjectType: "resource:report"},

	{ActionName: "read", ObjectKind: "resource", ObjectType: "resource:alarm"},
	{ActionName: "write", ObjectKind: "resource", ObjectType: "resource:alarm"},
	{ActionName: "delete", ObjectKind: "resource", ObjectType: "resource:alarm"},
	{ActionName: "manage", ObjectKind: "resource", ObjectType: "resource:alarm"},
}

var magistralaActionAssignmentRules = []ActionAssignmentRuleSpec{
	{
		EntityKind: "device",
		ActionName: "publish",
		ObjectKind: "resource",
		ObjectType: "resource:channel",
		Decision:   "allow",
	},
	{
		EntityKind: "device",
		ActionName: "subscribe",
		ObjectKind: "resource",
		ObjectType: "resource:channel",
		Decision:   "allow",
	},
}

// BootstrapMagistralaActions installs Magistrala-specific action applicability in Atom.
// It is safe to call repeatedly during startup.
func BootstrapMagistralaActions(ctx context.Context, client *Client) error {
	if client == nil {
		return fmt.Errorf("atom client is nil")
	}
	capabilities, err := client.ListCapabilities(ctx)
	if err != nil {
		return fmt.Errorf("list atom actions: %w", err)
	}
	byName := map[string]Capability{}
	for _, capability := range capabilities.Items {
		byName[capability.Name] = capability
	}

	for _, spec := range magistralaActionApplicability {
		capability, ok := byName[spec.ActionName]
		if !ok {
			description := spec.Description
			if description == "" {
				description = magistralaActionDescriptions[spec.ActionName]
			}
			capability, err = client.CreateCapability(ctx, spec.ActionName, description)
			if err != nil {
				if !IsConflict(err) {
					return fmt.Errorf("create atom action %q: %w", spec.ActionName, err)
				}
				id, lookupErr := client.CapabilityID(ctx, spec.ActionName)
				if lookupErr != nil {
					return fmt.Errorf("lookup existing atom action %q after conflict: %w", spec.ActionName, lookupErr)
				}
				capability = Capability{ID: id, Name: spec.ActionName, Description: description}
			}
			byName[spec.ActionName] = capability
		}
		if _, err := client.AddCapabilityApplicability(ctx, capability.ID, spec.ObjectKind, spec.ObjectType); err != nil {
			return fmt.Errorf("add atom applicability %s -> %s:%s: %w", spec.ActionName, spec.ObjectKind, spec.ObjectType, err)
		}
	}

	for _, spec := range magistralaActionAssignmentRules {
		if err := ensureActionAssignmentRule(ctx, client, spec); err != nil {
			return fmt.Errorf("ensure atom assignment guardrail %s %s %s:%s: %w", spec.EntityKind, spec.ActionName, spec.ObjectKind, spec.ObjectType, err)
		}
	}
	return nil
}

func ensureActionAssignmentRule(ctx context.Context, client *Client, spec ActionAssignmentRuleSpec) error {
	rules, err := client.ListActionAssignmentRules(ctx, spec)
	if err != nil {
		return err
	}
	if actionAssignmentRuleExists(rules.Items, spec) {
		return nil
	}
	if _, err := client.CreateActionAssignmentRule(ctx, spec); err != nil {
		if !IsConflict(err) {
			return err
		}
		rules, lookupErr := client.ListActionAssignmentRules(ctx, spec)
		if lookupErr != nil {
			return fmt.Errorf("lookup existing rule after conflict: %w", lookupErr)
		}
		if actionAssignmentRuleExists(rules.Items, spec) {
			return nil
		}
		return err
	}
	return nil
}

func actionAssignmentRuleExists(rules []ActionAssignmentRule, spec ActionAssignmentRuleSpec) bool {
	for _, rule := range rules {
		if rule.TenantID == spec.TenantID &&
			rule.EntityKind == spec.EntityKind &&
			rule.ActionName == spec.ActionName &&
			rule.ObjectKind == spec.ObjectKind &&
			rule.ObjectType == spec.ObjectType &&
			rule.Decision == spec.Decision &&
			rule.IsAbsolute == spec.IsAbsolute {
			return true
		}
	}
	return false
}
