// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
	"fmt"
)

var magistralaActionDescriptions = map[string]string{
	atomActionRead:      "Read / view an object",
	atomActionWrite:     "Create or update an object",
	atomActionDelete:    "Delete an object",
	atomActionManage:    "Full administrative control",
	atomActionPublish:   "Publish messages to a channel",
	atomActionSubscribe: "Subscribe to channel messages",
	atomActionExecute:   "Execute a command or action",
	atomActionList:      "List objects",
}

var magistralaActionApplicability = []CapabilityApplicabilitySpec{
	{ActionName: atomActionWrite, ObjectKind: atomObjectKindTenant},

	{ActionName: atomActionRead, ObjectKind: atomObjectKindGroup},
	{ActionName: atomActionWrite, ObjectKind: atomObjectKindGroup},
	{ActionName: atomActionDelete, ObjectKind: atomObjectKindGroup},
	{ActionName: atomActionManage, ObjectKind: atomObjectKindGroup},
	{ActionName: atomActionList, ObjectKind: atomObjectKindGroup},

	{ActionName: atomActionRead, ObjectKind: atomObjectKindResource, ObjectType: atomObjectTypeResourceChannel},
	{ActionName: atomActionWrite, ObjectKind: atomObjectKindResource, ObjectType: atomObjectTypeResourceChannel},
	{ActionName: atomActionDelete, ObjectKind: atomObjectKindResource, ObjectType: atomObjectTypeResourceChannel},
	{ActionName: atomActionManage, ObjectKind: atomObjectKindResource, ObjectType: atomObjectTypeResourceChannel},
	{ActionName: atomActionPublish, ObjectKind: atomObjectKindResource, ObjectType: atomObjectTypeResourceChannel},
	{ActionName: atomActionSubscribe, ObjectKind: atomObjectKindResource, ObjectType: atomObjectTypeResourceChannel},

	{ActionName: atomActionRead, ObjectKind: atomObjectKindResource, ObjectType: atomObjectTypeResourceRule},
	{ActionName: atomActionWrite, ObjectKind: atomObjectKindResource, ObjectType: atomObjectTypeResourceRule},
	{ActionName: atomActionDelete, ObjectKind: atomObjectKindResource, ObjectType: atomObjectTypeResourceRule},
	{ActionName: atomActionManage, ObjectKind: atomObjectKindResource, ObjectType: atomObjectTypeResourceRule},
	{ActionName: atomActionExecute, ObjectKind: atomObjectKindResource, ObjectType: atomObjectTypeResourceRule},
	{ActionName: atomActionList, ObjectKind: atomObjectKindResource, ObjectType: atomObjectTypeResourceRule},

	{ActionName: atomActionRead, ObjectKind: atomObjectKindResource, ObjectType: atomObjectTypeResourceReport},
	{ActionName: atomActionWrite, ObjectKind: atomObjectKindResource, ObjectType: atomObjectTypeResourceReport},
	{ActionName: atomActionDelete, ObjectKind: atomObjectKindResource, ObjectType: atomObjectTypeResourceReport},
	{ActionName: atomActionManage, ObjectKind: atomObjectKindResource, ObjectType: atomObjectTypeResourceReport},
	{ActionName: atomActionExecute, ObjectKind: atomObjectKindResource, ObjectType: atomObjectTypeResourceReport},
	{ActionName: atomActionList, ObjectKind: atomObjectKindResource, ObjectType: atomObjectTypeResourceReport},

	{ActionName: atomActionRead, ObjectKind: atomObjectKindResource, ObjectType: atomObjectTypeResourceAlarm},
	{ActionName: atomActionWrite, ObjectKind: atomObjectKindResource, ObjectType: atomObjectTypeResourceAlarm},
	{ActionName: atomActionDelete, ObjectKind: atomObjectKindResource, ObjectType: atomObjectTypeResourceAlarm},
	{ActionName: atomActionManage, ObjectKind: atomObjectKindResource, ObjectType: atomObjectTypeResourceAlarm},
	{ActionName: atomActionList, ObjectKind: atomObjectKindResource, ObjectType: atomObjectTypeResourceAlarm},
}

var magistralaActionAssignmentRules = []ActionAssignmentRuleSpec{
	{
		EntityKind: atomKindDevice,
		ActionName: atomActionPublish,
		ObjectKind: atomObjectKindResource,
		ObjectType: atomObjectTypeResourceChannel,
		Decision:   atomDecisionAllow,
	},
	{
		EntityKind: atomKindDevice,
		ActionName: atomActionSubscribe,
		ObjectKind: atomObjectKindResource,
		ObjectType: atomObjectTypeResourceChannel,
		Decision:   atomDecisionAllow,
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
