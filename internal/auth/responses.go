// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

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
