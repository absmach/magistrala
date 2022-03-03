// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/gofrs/uuid"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/things"
)

const (
	maxLimitSize = 100
	maxNameSize  = 1024
	nameOrder    = "name"
	idOrder      = "id"
	ascDir       = "asc"
	descDir      = "desc"
	readPolicy   = "read"
	writePolicy  = "write"
	deletePolicy = "delete"
)

type createThingReq struct {
	token    string
	Name     string                 `json:"name,omitempty"`
	Key      string                 `json:"key,omitempty"`
	ID       string                 `json:"id,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func validateUUID(extID string) (err error) {
	id, err := uuid.FromString(extID)
	if id.String() != extID || err != nil {
		return apiutil.ErrInvalidIDFormat
	}

	return nil
}

func (req createThingReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	// Do the validation only if request contains ID
	if req.ID != "" {
		return validateUUID(req.ID)
	}

	return nil
}

type createThingsReq struct {
	token  string
	Things []createThingReq
}

func (req createThingsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.Things) <= 0 {
		return apiutil.ErrEmptyList
	}

	for _, thing := range req.Things {
		if thing.ID != "" {
			if err := validateUUID(thing.ID); err != nil {
				return err
			}
		}

		if len(thing.Name) > maxNameSize {
			return apiutil.ErrNameSize
		}
	}

	return nil
}

type shareThingReq struct {
	token    string
	thingID  string
	UserIDs  []string `json:"user_ids"`
	Policies []string `json:"policies"`
}

func (req shareThingReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.thingID == "" || len(req.UserIDs) == 0 {
		return apiutil.ErrMissingID
	}

	if len(req.Policies) == 0 {
		return apiutil.ErrEmptyList
	}

	for _, p := range req.Policies {
		if p != readPolicy && p != writePolicy && p != deletePolicy {
			return apiutil.ErrMalformedPolicy
		}
	}
	return nil
}

type updateThingReq struct {
	token    string
	id       string
	Name     string                 `json:"name,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (req updateThingReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	if len(req.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	return nil
}

type updateKeyReq struct {
	token string
	id    string
	Key   string `json:"key"`
}

func (req updateKeyReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	if req.Key == "" {
		return apiutil.ErrBearerKey
	}

	return nil
}

type createChannelReq struct {
	token    string
	Name     string                 `json:"name,omitempty"`
	ID       string                 `json:"id,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (req createChannelReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	// Do the validation only if request contains ID
	if req.ID != "" {
		return validateUUID(req.ID)
	}

	return nil
}

type createChannelsReq struct {
	token    string
	Channels []createChannelReq
}

func (req createChannelsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.Channels) <= 0 {
		return apiutil.ErrEmptyList
	}

	for _, channel := range req.Channels {
		if channel.ID != "" {
			if err := validateUUID(channel.ID); err != nil {
				return err
			}
		}

		if len(channel.Name) > maxNameSize {
			return apiutil.ErrNameSize
		}
	}

	return nil
}

type updateChannelReq struct {
	token    string
	id       string
	Name     string                 `json:"name,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (req updateChannelReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	if len(req.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	return nil
}

type viewResourceReq struct {
	token string
	id    string
}

func (req viewResourceReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type listResourcesReq struct {
	token        string
	pageMetadata things.PageMetadata
}

func (req *listResourcesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.pageMetadata.Limit > maxLimitSize || req.pageMetadata.Limit < 1 {
		return apiutil.ErrLimitSize
	}

	if len(req.pageMetadata.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	if req.pageMetadata.Order != "" &&
		req.pageMetadata.Order != nameOrder && req.pageMetadata.Order != idOrder {
		return apiutil.ErrInvalidOrder
	}

	if req.pageMetadata.Dir != "" &&
		req.pageMetadata.Dir != ascDir && req.pageMetadata.Dir != descDir {
		return apiutil.ErrInvalidDirection
	}

	return nil
}

type listByConnectionReq struct {
	token        string
	id           string
	pageMetadata things.PageMetadata
}

func (req listByConnectionReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	if req.pageMetadata.Limit > maxLimitSize || req.pageMetadata.Limit < 1 {
		return apiutil.ErrLimitSize
	}

	if req.pageMetadata.Order != "" &&
		req.pageMetadata.Order != nameOrder && req.pageMetadata.Order != idOrder {
		return apiutil.ErrInvalidOrder
	}

	if req.pageMetadata.Dir != "" &&
		req.pageMetadata.Dir != ascDir && req.pageMetadata.Dir != descDir {
		return apiutil.ErrInvalidDirection
	}

	return nil
}

type connectThingReq struct {
	token   string
	chanID  string
	thingID string
}

func (req connectThingReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.chanID == "" || req.thingID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type connectReq struct {
	token      string
	ChannelIDs []string `json:"channel_ids,omitempty"`
	ThingIDs   []string `json:"thing_ids,omitempty"`
}

func (req connectReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.ChannelIDs) == 0 || len(req.ThingIDs) == 0 {
		return apiutil.ErrEmptyList
	}

	for _, chID := range req.ChannelIDs {
		if chID == "" {
			return apiutil.ErrMissingID
		}
	}
	for _, thingID := range req.ThingIDs {
		if thingID == "" {
			return apiutil.ErrMissingID
		}
	}

	return nil
}

type listThingsGroupReq struct {
	token        string
	groupID      string
	pageMetadata things.PageMetadata
}

func (req listThingsGroupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingID
	}

	if req.pageMetadata.Limit > maxLimitSize || req.pageMetadata.Limit < 1 {
		return apiutil.ErrLimitSize
	}

	if len(req.pageMetadata.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	if req.pageMetadata.Order != "" &&
		req.pageMetadata.Order != nameOrder && req.pageMetadata.Order != idOrder {
		return apiutil.ErrInvalidOrder
	}

	if req.pageMetadata.Dir != "" &&
		req.pageMetadata.Dir != ascDir && req.pageMetadata.Dir != descDir {
		return apiutil.ErrInvalidDirection
	}

	return nil

}
