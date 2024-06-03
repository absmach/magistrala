// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package constraints

import (
	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/constraints/config"
)

var _ magistrala.ConstraintsProvider = (*constraintProvider)(nil)

type constraintProvider struct{}

func New() magistrala.ConstraintsProvider {
	return &constraintProvider{}
}

func (c *constraintProvider) Constraints() (magistrala.Constraints, error) {
	consts, err := config.ParseConstraints()
	if err != nil {
		return magistrala.Constraints{}, err
	}
	return consts, nil
}
