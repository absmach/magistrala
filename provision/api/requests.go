package api

import "github.com/mainflux/mainflux/internal/apiutil"

type provisionReq struct {
	token       string
	Name        string `json:"name"`
	ExternalID  string `json:"external_id"`
	ExternalKey string `json:"external_key"`
}

func (req provisionReq) validate() error {
	if req.ExternalID == "" {
		return apiutil.ErrMissingID
	}

	if req.ExternalKey == "" {
		return apiutil.ErrBearerKey
	}

	return nil
}

type mappingReq struct {
	token string
}

func (req mappingReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	return nil
}
