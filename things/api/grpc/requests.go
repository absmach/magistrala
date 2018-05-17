package grpc

import (
	"github.com/asaskevich/govalidator"
	"github.com/mainflux/mainflux/things"
)

type accessReq struct {
	thingKey string
	chanID   string
}

func (req accessReq) validate() error {
	if !govalidator.IsUUID(req.chanID) || req.thingKey == "" {
		return things.ErrMalformedEntity
	}
	return nil
}

type identifyReq struct {
	key string
}
