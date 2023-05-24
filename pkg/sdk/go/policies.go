package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mainflux/mainflux/pkg/errors"
)

const policiesEndpoint = "policies"

type Policy struct {
	Object   string   `json:"object,omitempty"`
	Subject  []string `json:"subjects,omitempty"`
	Policies []string `json:"policies,omitempty"`
}

func (sdk mfSDK) CreatePolicy(policy Policy, token string) errors.SDKError {
	data, err := json.Marshal(policy)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s", sdk.authURL, policiesEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, string(CTJSON), data, http.StatusCreated)
	return sdkerr
}

func (sdk mfSDK) DeletePolicy(policy Policy, token string) errors.SDKError {
	data, err := json.Marshal(policy)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s", sdk.authURL, policiesEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodDelete, url, token, string(CTJSON), data, http.StatusNoContent)
	return sdkerr
}
