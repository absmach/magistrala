package api

import "github.com/mainflux/mainflux/pkg/errors"

type provisionReq struct {
	token       string
	Name        string `json:"name"`
	ExternalID  string `json:"external_id"`
	ExternalKey string `json:"external_key"`
}

func (req provisionReq) validate() error {
	if req.ExternalID == "" || req.ExternalKey == "" {
		return errors.ErrMalformedEntity
	}
	return nil
}

type mappingReq struct {
	token string
}

func (req mappingReq) validate() error {
	if req.token == "" {
		return errors.ErrAuthentication
	}
	return nil
}
