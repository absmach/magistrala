package api

import (
	provsdk "github.com/mainflux/mainflux/provision/sdk"
)

type addThingReq struct {
	ExternalID  string `json:"externalid"`
	ExternalKey string `json:"externalkey"`
}

func (req addThingReq) validate() error {
	if req.ExternalID == "" || req.ExternalKey == "" {
		return provsdk.ErrMalformedEntity
	}

	return nil
}
