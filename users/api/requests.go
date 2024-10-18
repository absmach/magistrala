// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/users"
)

const maxLimitSize = 100

type createUserReq struct {
	user users.User
}

func (req createUserReq) validate() error {
	if len(req.user.FirstName) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}
	if len(req.user.LastName) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}
	if req.user.Credentials.UserName == "" {
		return apiutil.ErrMissingUserName
	}
	if req.user.Identity == "" {
		return apiutil.ErrMissingIdentity
	}
	if req.user.Credentials.Secret == "" {
		return apiutil.ErrMissingPass
	}
	if !passRegex.MatchString(req.user.Credentials.Secret) {
		return apiutil.ErrPasswordFormat
	}
	if req.user.Status == users.AllStatus {
		return svcerr.ErrInvalidStatus
	}

	return req.user.Validate()
}

type viewUserReq struct {
	id string
}

func (req viewUserReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type viewUserByUserNameReq struct {
	userName string
}

func (req viewUserByUserNameReq) validate() error {
	if req.userName == "" {
		return apiutil.ErrMissingUserName
	}

	return nil
}

type listUsersReq struct {
	status    users.Status
	offset    uint64
	limit     uint64
	userName  string
	tag       string
	firstName string
	lastName  string
	identity  string
	metadata  users.Metadata
	order     string
	dir       string
	id        string
}

func (req listUsersReq) validate() error {
	if req.limit > maxLimitSize || req.limit < 1 {
		return apiutil.ErrLimitSize
	}
	if req.dir != "" && (req.dir != api.AscDir && req.dir != api.DescDir) {
		return apiutil.ErrInvalidDirection
	}

	return nil
}

type searchUsersReq struct {
	Offset    uint64
	Limit     uint64
	UserName  string
	FirstName string
	LastName  string
	Id        string
	Order     string
	Dir       string
}

func (req searchUsersReq) validate() error {
	if req.UserName == "" && req.Id == "" {
		return apiutil.ErrEmptySearchQuery
	}

	return nil
}

type listMembersByObjectReq struct {
	users.Page
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

type updateUserReq struct {
	id       string
	UserName string         `json:"user_name,omitempty"`
	Metadata users.Metadata `json:"metadata,omitempty"`
}

func (req updateUserReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type updateUserTagsReq struct {
	id   string
	Tags []string `json:"tags,omitempty"`
}

func (req updateUserTagsReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type updateUserRoleReq struct {
	id   string
	role users.Role
	Role string `json:"role,omitempty"`
}

func (req updateUserRoleReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type updateUserIdentityReq struct {
	id       string
	Identity string `json:"identity,omitempty"`
}

func (req updateUserIdentityReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type updateUserSecretReq struct {
	OldSecret string `json:"old_secret,omitempty"`
	NewSecret string `json:"new_secret,omitempty"`
}

func (req updateUserSecretReq) validate() error {
	if req.OldSecret == "" || req.NewSecret == "" {
		return apiutil.ErrMissingPass
	}
	if !passRegex.MatchString(req.NewSecret) {
		return apiutil.ErrPasswordFormat
	}

	return nil
}

type updateUserNamesReq struct {
	User users.User
}

func (req updateUserNamesReq) validate() error {
	if req.User.ID == "" {
		return apiutil.ErrMissingID
	}
	if len(req.User.Credentials.UserName) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}
	if len(req.User.FirstName) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}
	if len(req.User.LastName) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}

	if req.User.FirstName == "" && req.User.LastName == "" {
		return apiutil.ErrMissingFullName
	}

	return nil
}

type updateProfilePictureReq struct {
	id             string
	ProfilePicture string `json:"profile_picture,omitempty"`
}

func (req updateProfilePictureReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type changeUserStatusReq struct {
	id string
}

func (req changeUserStatusReq) validate() error {
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
