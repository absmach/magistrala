// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/absmach/magistrala/clients"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
)

type createClientReq struct {
	client clients.Client
}

func (req createClientReq) validate() error {
	if len(req.client.Name) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}
	if req.client.ID != "" {
		return api.ValidateUUID(req.client.ID)
	}

	return nil
}

type createClientsReq struct {
	Clients []clients.Client
}

func (req createClientsReq) validate() error {
	if len(req.Clients) == 0 {
		return apiutil.ErrEmptyList
	}
	for _, c := range req.Clients {
		if c.ID != "" {
			if err := api.ValidateUUID(c.ID); err != nil {
				return err
			}
		}
		if len(c.Name) > api.MaxNameSize {
			return apiutil.ErrNameSize
		}
	}

	return nil
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

type viewClientPermsReq struct {
	id string
}

func (req viewClientPermsReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type listClientsReq struct {
	status     clients.Status
	offset     uint64
	limit      uint64
	name       string
	tag        string
	permission string
	visibility string
	userID     string
	listPerms  bool
	metadata   clients.Metadata
	id         string
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
	clients.Page
	groupID string
}

func (req listMembersReq) validate() error {
	if req.groupID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type updateClientReq struct {
	id       string
	Name     string                 `json:"name,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Tags     []string               `json:"tags,omitempty"`
}

func (req updateClientReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	if len(req.Name) > api.MaxNameSize {
		return apiutil.ErrNameSize
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

type updateClientCredentialsReq struct {
	id     string
	Secret string `json:"secret,omitempty"`
}

func (req updateClientCredentialsReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	if req.Secret == "" {
		return apiutil.ErrMissingSecret
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

type setClientParentGroupReq struct {
	id            string
	ParentGroupID string `json:"parent_group_id"`
}

func (req setClientParentGroupReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	if req.ParentGroupID == "" {
		return apiutil.ErrMissingParentGroupID
	}
	return nil
}

type removeClientParentGroupReq struct {
	id string
}

func (req removeClientParentGroupReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type deleteClientReq struct {
	id string
}

func (req deleteClientReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}
