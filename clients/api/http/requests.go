package http

import (
	"github.com/asaskevich/govalidator"
	"github.com/mainflux/mainflux/clients"
)

const maxLimitSize = 100

type apiReq interface {
	validate() error
}

type identityReq struct {
	key string
}

func (req identityReq) validate() error {
	if req.key == "" {
		return clients.ErrUnauthorizedAccess
	}

	return nil
}

type addClientReq struct {
	key    string
	client clients.Client
}

func (req addClientReq) validate() error {
	if req.key == "" {
		return clients.ErrUnauthorizedAccess
	}

	return req.client.Validate()
}

type updateClientReq struct {
	key    string
	id     string
	client clients.Client
}

func (req updateClientReq) validate() error {
	if req.key == "" {
		return clients.ErrUnauthorizedAccess
	}

	if !govalidator.IsUUID(req.id) {
		return clients.ErrNotFound
	}

	return req.client.Validate()
}

type createChannelReq struct {
	key     string
	channel clients.Channel
}

func (req createChannelReq) validate() error {
	if req.key == "" {
		return clients.ErrUnauthorizedAccess
	}

	return nil
}

type updateChannelReq struct {
	key     string
	id      string
	channel clients.Channel
}

func (req updateChannelReq) validate() error {
	if req.key == "" {
		return clients.ErrUnauthorizedAccess
	}

	if !govalidator.IsUUID(req.id) {
		return clients.ErrNotFound
	}

	return nil
}

type viewResourceReq struct {
	key string
	id  string
}

func (req viewResourceReq) validate() error {
	if req.key == "" {
		return clients.ErrUnauthorizedAccess
	}

	if !govalidator.IsUUID(req.id) {
		return clients.ErrNotFound
	}

	return nil
}

type listResourcesReq struct {
	key    string
	offset int
	limit  int
}

func (req *listResourcesReq) validate() error {
	if req.key == "" {
		return clients.ErrUnauthorizedAccess
	}

	if req.offset >= 0 && req.limit > 0 && req.limit <= maxLimitSize {
		return nil
	}

	return clients.ErrMalformedEntity
}

type connectionReq struct {
	key      string
	chanID   string
	clientID string
}

func (req connectionReq) validate() error {
	if req.key == "" {
		return clients.ErrUnauthorizedAccess
	}

	if !govalidator.IsUUID(req.chanID) || !govalidator.IsUUID(req.clientID) {
		return clients.ErrNotFound
	}

	return nil
}
