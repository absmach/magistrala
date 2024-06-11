// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/absmach/magistrala"
)

const (
	defaultConstraintPath = "./constraints_config.toml"
	filePermission        = 0o644
)

type Limits struct {
	Create int64 `toml:"create"`
	Update int64 `toml:"update"`
}
type Constraints struct {
	Limits Limits `toml:"limits"`
}
type tomlConfig struct {
	Services map[string]Constraints
}

// Attempts to read constraints from the default path, if the file does not exist,
// it will be created with the default constraints.
func New(serviceName string) (magistrala.Constraints, error) {
	return read(defaultConstraintPath, serviceName)
}
func read(file, serviceName string) (magistrala.Constraints, error) {
	var tc tomlConfig
	if _, err := toml.DecodeFile(file, &tc.Services); err != nil {
		return nil, fmt.Errorf("error reading config file: %s", err)
	}
	svcConstraint, exists := tc.Services[serviceName]
	if !exists {
		return nil, fmt.Errorf("section [%s] not found", serviceName)
	}
	return svcConstraint, nil
}

func (c Constraints) CheckLimits(operation magistrala.Operation, currentValue uint64) error {
	
	return nil
}
