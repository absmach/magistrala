// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package permissions

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type PermissionConfig struct {
	Entities map[string]EntityPermissions `yaml:",inline"`
}

type EntityPermissions struct {
	Operations      []map[string]string `yaml:"operations"`
	RolesOperations []map[string]string `yaml:"roles_operations"`
}

func ParsePermissionsFile(filePath string) (*PermissionConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read permissions file: %w", err)
	}

	var config PermissionConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse permissions file: %w", err)
	}

	return &config, nil
}

type PATEntityPermissions struct {
	Operations      []map[string]string `yaml:"operations"`
	RolesOperations []map[string]string `yaml:"roles_operations"`
}

type PATPermissionConfig struct {
	Entities map[string]PATEntityPermissions `yaml:",inline"`
}

func ParsePATPermissionsFile(filePath string) (*PATPermissionConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read PAT permissions file: %w", err)
	}

	var config PATPermissionConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse PAT permissions file: %w", err)
	}

	return &config, nil
}

// GetPATEntityOperations returns a registry mapping each PAT entity type to the
// set of valid PAT operation names. Role management operations are prefixed with
// "role_" (matching the RoleOperationPrefix convention used by the auth package).
func (pc *PATPermissionConfig) GetPATEntityOperations() map[string]map[string]bool {
	result := make(map[string]map[string]bool, len(pc.Entities))
	for entity, perms := range pc.Entities {
		ops := make(map[string]bool)
		for _, opMap := range perms.Operations {
			for name := range opMap {
				ops[name] = true
			}
		}
		for _, opMap := range perms.RolesOperations {
			for name := range opMap {
				ops["role_"+name] = true
			}
		}
		result[entity] = ops
	}
	return result
}

func (pc *PermissionConfig) GetEntityPermissions(entityType string) (map[string]Permission, map[string]Permission, error) {
	entityPerms, ok := pc.Entities[entityType]
	if !ok {
		return nil, nil, fmt.Errorf("entity type %s not found in permissions file", entityType)
	}

	operations := make(map[string]Permission)
	for _, op := range entityPerms.Operations {
		for name, perm := range op {
			if perm != "" {
				operations[name] = Permission(perm)
			}
		}
	}

	rolesOperations := make(map[string]Permission)
	for _, op := range entityPerms.RolesOperations {
		for name, perm := range op {
			if perm != "" {
				rolesOperations[name] = Permission(perm)
			}
		}
	}

	return operations, rolesOperations, nil
}
