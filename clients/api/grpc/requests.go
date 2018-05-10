package grpc

import (
	"github.com/asaskevich/govalidator"
	"github.com/mainflux/mainflux/clients"
)

type accessReq struct {
	clientKey string
	chanID    string
}

func (req accessReq) validate() error {
	if !govalidator.IsUUID(req.chanID) || req.clientKey == "" {
		return clients.ErrMalformedEntity
	}
	return nil
}
