// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/internal/apiutil"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
)

type createClientReq struct {
	client mgclients.Client
	token  string
}

func (req createClientReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if len(req.client.Name) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}
	if req.client.ID != "" {
		return api.ValidateUUID(req.client.ID)
	}

	return nil
}

type createClientsReq struct {
	token   string
	Clients []mgclients.Client
}

func (req createClientsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if len(req.Clients) == 0 {
		return apiutil.ErrEmptyList
	}
	for _, client := range req.Clients {
		if client.ID != "" {
			if err := api.ValidateUUID(client.ID); err != nil {
				return err
			}
		}
		if len(client.Name) > api.MaxNameSize {
			return apiutil.ErrNameSize
		}
	}

	return nil
}

type viewClientReq struct {
	token string
	id    string
}

func (req viewClientReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type viewClientPermsReq struct {
	token string
	id    string
}

func (req viewClientPermsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type listClientsReq struct {
	token      string
	status     mgclients.Status
	offset     uint64
	limit      uint64
	name       string
	tag        string
	owner      string
	permission string
	visibility string
	userID     string
	listPerms  bool
	metadata   mgclients.Metadata
}

func (req listClientsReq) validate() error {
	if req.limit > api.MaxLimitSize || req.limit < 1 {
		return apiutil.ErrLimitSize
	}
	if req.visibility != "" &&
		req.visibility != api.AllVisibility &&
		req.visibility != api.MyVisibility &&
		req.visibility != api.SharedVisibility {
		return apiutil.ErrInvalidVisibilityType
	}
	if len(req.name) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}

	return nil
}

type listMembersReq struct {
	mgclients.Page
	token   string
	groupID string
}

func (req listMembersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.groupID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type updateClientReq struct {
	token    string
	id       string
	Name     string                 `json:"name,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Tags     []string               `json:"tags,omitempty"`
}

func (req updateClientReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	if len(req.Name) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}

	return nil
}

type updateClientTagsReq struct {
	id    string
	token string
	Tags  []string `json:"tags,omitempty"`
}

func (req updateClientTagsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type updateClientCredentialsReq struct {
	token  string
	id     string
	Secret string `json:"secret,omitempty"`
}

func (req updateClientCredentialsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	if req.Secret == "" {
		return apiutil.ErrBearerKey
	}

	return nil
}

type changeClientStatusReq struct {
	token string
	id    string
}

func (req changeClientStatusReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type assignUsersRequest struct {
	token    string
	groupID  string
	Relation string   `json:"relation"`
	UserIDs  []string `json:"user_ids"`
}

func (req assignUsersRequest) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

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

type unassignUsersRequest struct {
	token    string
	groupID  string
	Relation string   `json:"relation"`
	UserIDs  []string `json:"user_ids"`
}

func (req unassignUsersRequest) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

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

type assignUserGroupsRequest struct {
	token        string
	groupID      string
	UserGroupIDs []string `json:"group_ids"`
}

func (req assignUserGroupsRequest) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingID
	}

	if len(req.UserGroupIDs) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}

type unassignUserGroupsRequest struct {
	token        string
	groupID      string
	UserGroupIDs []string `json:"group_ids"`
}

func (req unassignUserGroupsRequest) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingID
	}

	if len(req.UserGroupIDs) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}

type connectChannelThingRequest struct {
	token     string
	ThingID   string `json:"thing_id,omitempty"`
	ChannelID string `json:"channel_id,omitempty"`
}

func (req *connectChannelThingRequest) validate() error {
	if req.ThingID == "" || req.ChannelID == "" {
		return errors.ErrCreateEntity
	}
	return nil
}

type disconnectChannelThingRequest struct {
	token     string
	ThingID   string `json:"thing_id,omitempty"`
	ChannelID string `json:"channel_id,omitempty"`
}

func (req *disconnectChannelThingRequest) validate() error {
	if req.ThingID == "" || req.ChannelID == "" {
		return errors.ErrCreateEntity
	}
	return nil
}

type thingShareRequest struct {
	token    string
	thingID  string
	Relation string   `json:"relation,omitempty"`
	UserIDs  []string `json:"user_ids,omitempty"`
}

func (req *thingShareRequest) validate() error {
	if req.thingID == "" {
		return errors.ErrMalformedEntity
	}
	if req.Relation == "" || len(req.UserIDs) <= 0 {
		return errors.ErrCreateEntity
	}
	return nil
}

type thingUnshareRequest struct {
	token    string
	thingID  string
	Relation string   `json:"relation,omitempty"`
	UserIDs  []string `json:"user_ids,omitempty"`
}

func (req *thingUnshareRequest) validate() error {
	if req.thingID == "" {
		return errors.ErrMalformedEntity
	}
	if req.Relation == "" || len(req.UserIDs) <= 0 {
		return errors.ErrCreateEntity
	}
	return nil
}

type deleteClientReq struct {
	token string
	id    string
}

func (req deleteClientReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	return nil
}
