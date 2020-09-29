package api

type provisionReq struct {
	token       string
	Name        string `json:"name"`
	ExternalID  string `json:"external_id"`
	ExternalKey string `json:"external_key"`
}

func (req provisionReq) validate() error {
	if req.ExternalID == "" || req.ExternalKey == "" {
		return errMalformedEntity
	}
	return nil
}

type mappingReq struct {
	token string
}

func (req mappingReq) validate() error {
	if req.token == "" {
		return errUnauthorized
	}
	return nil
}
