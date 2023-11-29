// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

type identityRes struct {
	id       string
	userID   string
	domainID string
}

type issueRes struct {
	accessToken  string
	refreshToken string
	accessType   string
}

type authorizeRes struct {
	id         string
	authorized bool
}

type addPolicyRes struct {
	authorized bool
}
type addPoliciesRes struct {
	authorized bool
}

type deletePolicyRes struct {
	deleted bool
}

type deletePoliciesRes struct {
	deleted bool
}

type listObjectsRes struct {
	policies      []string
	nextPageToken string
}

type countObjectsRes struct {
	count int
}

type listSubjectsRes struct {
	policies      []string
	nextPageToken string
}

type countSubjectsRes struct {
	count int
}

type listPermissionsRes struct {
	Domain          string
	SubjectType     string
	Subject         string
	SubjectRelation string
	ObjectType      string
	Object          string
	Permissions     []string
}
