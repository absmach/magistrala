// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

type identityRes struct {
	id    string
	email string
}

type issueRes struct {
	value string
}

type authorizeRes struct {
	authorized bool
}

type addPolicyRes struct {
	authorized bool
}

type deletePolicyRes struct {
	deleted bool
}

type membersRes struct {
	total     uint64
	offset    uint64
	limit     uint64
	groupType string
	members   []string
}
type emptyRes struct {
	err error
}
