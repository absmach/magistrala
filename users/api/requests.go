// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/users"
)

const maxLimitSize = 100

type createClientReq struct {
	client mgclients.Client
}

func (req createUserReq) validate() error {
	if len(req.user.Name) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}
	if len(req.user.UserName) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}
	if len(req.user.FirstName) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}
	if len(req.user.LastName) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}
	if req.user.Credentials.Identity == "" {
		return apiutil.ErrMissingIdentity
	}
	if req.user.Credentials.Secret == "" {
		return apiutil.ErrMissingPass
	}
	if !passRegex.MatchString(req.user.Credentials.Secret) {
		return apiutil.ErrPasswordFormat
	}

	return req.user.Validate()
}

type viewClientReq struct {
	id string
}

func (req viewClientReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type listClientsReq struct {
	status   mgclients.Status
	offset   uint64
	limit    uint64
	name     string
	tag      string
	identity string
	metadata users.Metadata
	order    string
	dir      string
	id       string
}

func (req listClientsReq) validate() error {
	if req.limit > maxLimitSize || req.limit < 1 {
		return apiutil.ErrLimitSize
	}
	if req.dir != "" && (req.dir != api.AscDir && req.dir != api.DescDir) {
		return apiutil.ErrInvalidDirection
	}

	return nil
}

type searchClientsReq struct {
	Offset uint64
	Limit  uint64
	Name   string
	Id     string
	Order  string
	Dir    string
}

func (req searchClientsReq) validate() error {
	if req.Name == "" && req.Id == "" {
		return apiutil.ErrEmptySearchQuery
	}

	return nil
}

type listMembersByObjectReq struct {
	mgclients.Page
	objectKind string
	objectID   string
}

func (req listMembersByObjectReq) validate() error {
	if req.objectID == "" {
		return apiutil.ErrMissingID
	}
	if req.objectKind == "" {
		return apiutil.ErrMissingMemberKind
	}

	return nil
}

type updateClientReq struct {
	id       string
	Name     string         `json:"name,omitempty"`
	Metadata users.Metadata `json:"metadata,omitempty"`
}

func (req updateClientReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type updateClientTagsReq struct {
	id   string
	Tags []string `json:"tags,omitempty"`
}

func (req updateClientTagsReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type updateClientRoleReq struct {
	id   string
	role mgclients.Role
	Role string `json:"role,omitempty"`
}

func (req updateClientRoleReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type updateClientIdentityReq struct {
	id       string
	Identity string `json:"identity,omitempty"`
}

func (req updateClientIdentityReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type updateClientSecretReq struct {
	OldSecret string `json:"old_secret,omitempty"`
	NewSecret string `json:"new_secret,omitempty"`
}

func (req updateClientSecretReq) validate() error {
	if req.OldSecret == "" || req.NewSecret == "" {
		return apiutil.ErrMissingPass
	}
	if !passRegex.MatchString(req.NewSecret) {
		return apiutil.ErrPasswordFormat
	}

	return nil
}

type changeClientStatusReq struct {
	id string
}

func (req changeClientStatusReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	if req.FullName == "" {
		return apiutil.ErrMissingFullName
	}

	return nil
}

type changeClientStatusReq struct {
	id string
}

func (req changeClientStatusReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type loginUserReq struct {
	Identity string `json:"identity,omitempty"`
	Secret   string `json:"secret,omitempty"`
	DomainID string `json:"domain_id,omitempty"`
}

func (req loginUserReq) validate() error {
	if req.Identity == "" {
		return apiutil.ErrMissingIdentity
	}
	if req.Secret == "" {
		return apiutil.ErrMissingPass
	}

	return nil
}

type tokenReq struct {
	RefreshToken string `json:"refresh_token,omitempty"`
	DomainID     string `json:"domain_id,omitempty"`
}

func (req tokenReq) validate() error {
	if req.RefreshToken == "" {
		return apiutil.ErrBearerToken
	}

	return nil
}

type passwResetReq struct {
	Email string `json:"email"`
	Host  string `json:"host"`
}

func (req passwResetReq) validate() error {
	if req.Email == "" {
		return apiutil.ErrMissingEmail
	}
	if req.Host == "" {
		return apiutil.ErrMissingHost
	}

	return nil
}

type resetTokenReq struct {
	Token    string `json:"token"`
	Password string `json:"password"`
	ConfPass string `json:"confirm_password"`
}

func (req resetTokenReq) validate() error {
	if req.Password == "" {
		return apiutil.ErrMissingPass
	}
	if req.ConfPass == "" {
		return apiutil.ErrMissingConfPass
	}
	if req.Token == "" {
		return apiutil.ErrBearerToken
	}
	if req.Password != req.ConfPass {
		return apiutil.ErrInvalidResetPass
	}
	if !passRegex.MatchString(req.ConfPass) {
		return apiutil.ErrPasswordFormat
	}

	return nil
}

type assignUsersReq struct {
	groupID  string
	Relation string   `json:"relation"`
	UserIDs  []string `json:"user_ids"`
}

func (req assignUsersReq) validate() error {
	if req.Relation == "" {
		return apiutil.ErrMissingRelation
	}

	if req.groupID == "" {
		return apiutil.ErrMissingID
	}

	if len(req.UserIDs) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}

type unassignUsersReq struct {
	groupID  string
	Relation string   `json:"relation"`
	UserIDs  []string `json:"user_ids"`
}

func (req unassignUsersReq) validate() error {
	if req.groupID == "" {
		return apiutil.ErrMissingID
	}

	if len(req.UserIDs) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}

type assignGroupsReq struct {
	groupID  string
	GroupIDs []string `json:"group_ids"`
}

func (req assignGroupsReq) validate() error {
	if req.groupID == "" {
		return apiutil.ErrMissingID
	}

	if len(req.GroupIDs) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}

type unassignGroupsReq struct {
	groupID  string
	GroupIDs []string `json:"group_ids"`
}

func (req unassignGroupsReq) validate() error {
	if req.groupID == "" {
		return apiutil.ErrMissingID
	}

	if len(req.GroupIDs) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}
