// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"github.com/caarlos0/env/v7"
)

type Options struct {
	// Environment keys and values that will be accessible for the service
	Environment map[string]string

	// TagName specifies another tagname to use rather than the default env
	TagName string

	// RequiredIfNoDef automatically sets all env as required if they do not declare 'envDefault'
	RequiredIfNoDef bool

	// OnSet allows to run a function when a value is set
	OnSet env.OnSetFn

	// Prefix define a prefix for each key
	Prefix string
}

func Parse(v interface{}, opts ...Options) error {
	altOpts := []env.Options{}

	for _, opt := range opts {
		altOpts = append(altOpts, env.Options{
			Environment:     opt.Environment,
			TagName:         opt.TagName,
			RequiredIfNoDef: opt.RequiredIfNoDef,
			OnSet:           opt.OnSet,
			Prefix:          opt.Prefix,
		})
	}

	return env.Parse(v, altOpts...)
}
