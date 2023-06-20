// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/mainflux/mainflux/internal/api"
	"github.com/mainflux/mainflux/internal/apiutil"
	mfclients "github.com/mainflux/mainflux/pkg/clients"
)

type createClientReq struct {
	client mfclients.Client
	token  string
}

func (req createClientReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if len(req.client.Name) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}
	// Do the validation only if request contains ID
	if req.client.ID != "" {
		return api.ValidateUUID(req.client.ID)
	}

	return nil
}

type createClientsReq struct {
	token   string
	Clients []mfclients.Client
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
	return nil
}

type listClientsReq struct {
	token      string
	status     mfclients.Status
	offset     uint64
	limit      uint64
	name       string
	tag        string
	owner      string
	sharedBy   string
	visibility string
	metadata   mfclients.Metadata
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
	if req.limit > api.MaxLimitSize || req.limit < 1 {
		return apiutil.ErrLimitSize
	}

	if len(req.name) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}

	return nil
}

type listMembersReq struct {
	mfclients.Page
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

type updateClientOwnerReq struct {
	id    string
	token string
	Owner string `json:"owner,omitempty"`
}

func (req updateClientOwnerReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	if req.Owner == "" {
		return apiutil.ErrMissingOwner
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
