// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/url"

	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/users"
)

const maxLimitSize = 100

type createUserReq struct {
	users.User
}

func (req createUserReq) validate() error {
	if len(req.User.FirstName) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}
	if len(req.User.LastName) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}
	if req.User.FirstName == "" {
		return apiutil.ErrMissingFirstName
	}
	if req.User.LastName == "" {
		return apiutil.ErrMissingLastName
	}
	if req.User.Credentials.Username == "" {
		return apiutil.ErrMissingUsername
	}
	if err := api.ValidateUserName(req.User.Credentials.Username); err != nil {
		return err
	}
	// Username must not be a valid email format due to username/email login.
	if err := api.ValidateEmail(req.User.Credentials.Username); err == nil {
		return apiutil.ErrInvalidUsername
	}

	if req.User.Email == "" {
		return apiutil.ErrMissingEmail
	}
	// Email must be in a valid format.
	if err := api.ValidateEmail(req.User.Email); err != nil {
		return err
	}
	if req.User.Credentials.Secret == "" {
		return apiutil.ErrMissingPass
	}
	if !passRegex.MatchString(req.User.Credentials.Secret) {
		return apiutil.ErrPasswordFormat
	}
	if req.User.Status == users.AllStatus {
		return svcerr.ErrInvalidStatus
	}
	if req.User.ProfilePicture != "" {
		if _, err := url.Parse(req.User.ProfilePicture); err != nil {
			return apiutil.ErrInvalidProfilePictureURL
		}
	}

	return req.User.Validate()
}

type sendVerificationReq struct{}

type verifyEmailReq struct {
	token string
}

func (req verifyEmailReq) validate() error {
	if req.token == "" {
		return apiutil.ErrInvalidVerification
	}

	return nil
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

type listUsersReq struct {
	status    users.Status
	offset    uint64
	limit     uint64
	onlyTotal bool
	userName  string
	tags      users.TagsQuery
	firstName string
	lastName  string
	email     string
	metadata  users.Metadata
	order     string
	dir       string
	id        string
}

func (req listUsersReq) validate() error {
	if req.limit > maxLimitSize || req.limit < 1 {
		return apiutil.ErrLimitSize
	}

	switch req.order {
	case "", api.CreatedAtOrder, api.UpdatedAtOrder, api.FirstNameKey, api.LastNameKey, api.UsernameKey, api.EmailKey:
	default:
		return apiutil.ErrInvalidOrder
	}

	if req.dir != "" && (req.dir != api.AscDir && req.dir != api.DescDir) {
		return apiutil.ErrInvalidDirection
	}

	return nil
}

type searchUsersReq struct {
	Offset    uint64
	Limit     uint64
	Username  string
	FirstName string
	LastName  string
	Id        string
	Order     string
	Dir       string
}

func (req searchUsersReq) validate() error {
	if req.Username == "" && req.Id == "" && req.FirstName == "" && req.LastName == "" {
		return apiutil.ErrEmptySearchQuery
	}

	return nil
}

type updateUserReq struct {
	id              string
	FirstName       *string         `json:"first_name,omitempty"`
	LastName        *string         `json:"last_name,omitempty"`
	Metadata        *users.Metadata `json:"metadata,omitempty"`
	PrivateMetadata *users.Metadata `json:"private_metadata,omitempty"`
}

func (req updateUserReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type updateUserTagsReq struct {
	id   string
	Tags *[]string `json:"tags,omitempty"`
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

type updateEmailReq struct {
	id    string
	Email string `json:"email,omitempty"`
}

func (req updateEmailReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	if err := api.ValidateEmail(req.Email); err != nil {
		return err
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

type updateUsernameReq struct {
	id       string
	Username string `json:"username,omitempty"`
}

func (req updateUsernameReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	if len(req.Username) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}
	if req.Username == "" {
		return apiutil.ErrMissingUsername
	}

	return nil
}

type updateProfilePictureReq struct {
	id             string
	ProfilePicture *string `json:"profile_picture,omitempty"`
}

func (req updateProfilePictureReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	if req.ProfilePicture != nil {
		if _, err := url.Parse(*req.ProfilePicture); err != nil {
			return apiutil.ErrInvalidProfilePictureURL
		}
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
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"`
	Description string `json:"description,omitempty"`
}

func (req loginUserReq) validate() error {
	if req.Username == "" {
		return apiutil.ErrMissingUsernameEmail
	}
	if req.Password == "" {
		return apiutil.ErrMissingPass
	}

	return nil
}

type tokenReq struct {
	RefreshToken string `json:"refresh_token,omitempty"`
}

func (req tokenReq) validate() error {
	if req.RefreshToken == "" {
		return apiutil.ErrBearerToken
	}

	return nil
}

type revokeTokenReq struct {
	TokenID string `json:"token_id,omitempty"`
}

func (req revokeTokenReq) validate() error {
	if req.TokenID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type passResetReq struct {
	Email string `json:"email"`
}

func (req passResetReq) validate() error {
	if req.Email == "" {
		return apiutil.ErrMissingEmail
	}

	return nil
}

type resetTokenReq struct {
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
	if req.Password != req.ConfPass {
		return apiutil.ErrInvalidResetPass
	}
	if !passRegex.MatchString(req.ConfPass) {
		return apiutil.ErrPasswordFormat
	}

	return nil
}
