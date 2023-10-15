package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mainflux/mainflux/pkg/errors"
)

const (
	policyEndpoint    = "policies"
	authorizeEndpoint = "authorize"
	accessEndpoint    = "access"
)

// Policy represents an argument struct for making a policy related function calls.
type Policy struct {
	OwnerID   string    `json:"owner_id"`
	Subject   string    `json:"subject"`
	Object    string    `json:"object"`
	Actions   []string  `json:"actions"`
	External  bool      `json:"external,omitempty"` // This is specificially used in things service. If set to true, it means the subject is userID otherwise it is thingID.
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AccessRequest struct {
	Subject    string `json:"subject,omitempty"`
	Object     string `json:"object,omitempty"`
	Action     string `json:"action,omitempty"`
	EntityType string `json:"entity_type,omitempty"`
}

func (sdk mfSDK) AuthorizeUser(accessReq AccessRequest, token string) (bool, errors.SDKError) {
	data, err := json.Marshal(accessReq)
	if err != nil {
		return false, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s", sdk.usersURL, authorizeEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return false, sdkerr
	}

	return true, nil
}

func (sdk mfSDK) CreateUserPolicy(p Policy, token string) errors.SDKError {
	data, err := json.Marshal(p)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s", sdk.usersURL, policyEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusCreated)
	if sdkerr != nil {
		return sdkerr
	}

	return nil
}

func (sdk mfSDK) UpdateUserPolicy(p Policy, token string) errors.SDKError {
	data, err := json.Marshal(p)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s", sdk.usersURL, policyEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPut, url, token, data, nil, http.StatusNoContent)
	if sdkerr != nil {
		return sdkerr
	}

	return nil
}

func (sdk mfSDK) ListUserPolicies(pm PageMetadata, token string) (PolicyPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.usersURL, policyEndpoint, pm)
	if err != nil {
		return PolicyPage{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return PolicyPage{}, sdkerr
	}

	var pp PolicyPage
	if err := json.Unmarshal(body, &pp); err != nil {
		return PolicyPage{}, errors.NewSDKError(err)
	}

	return pp, nil
}

func (sdk mfSDK) DeleteUserPolicy(p Policy, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.usersURL, policyEndpoint, p.Subject, p.Object)

	_, _, sdkerr := sdk.processRequest(http.MethodDelete, url, token, nil, nil, http.StatusNoContent)

	return sdkerr
}

func (sdk mfSDK) CreateThingPolicy(p Policy, token string) errors.SDKError {
	data, err := json.Marshal(p)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s", sdk.thingsURL, policyEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusCreated)
	if sdkerr != nil {
		return sdkerr
	}

	return nil
}

func (sdk mfSDK) UpdateThingPolicy(p Policy, token string) errors.SDKError {
	data, err := json.Marshal(p)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s", sdk.thingsURL, policyEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPut, url, token, data, nil, http.StatusNoContent)
	if sdkerr != nil {
		return sdkerr
	}

	return nil
}

func (sdk mfSDK) ListThingPolicies(pm PageMetadata, token string) (PolicyPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.thingsURL, policyEndpoint, pm)
	if err != nil {
		return PolicyPage{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return PolicyPage{}, sdkerr
	}

	var pp PolicyPage
	if err := json.Unmarshal(body, &pp); err != nil {
		return PolicyPage{}, errors.NewSDKError(err)
	}

	return pp, nil
}

func (sdk mfSDK) DeleteThingPolicy(p Policy, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.thingsURL, policyEndpoint, p.Subject, p.Object)

	_, _, sdkerr := sdk.processRequest(http.MethodDelete, url, token, nil, nil, http.StatusNoContent)

	return sdkerr
}

func (sdk mfSDK) Assign(actions []string, userID, groupID, token string) errors.SDKError {
	policy := Policy{
		Subject: userID,
		Object:  groupID,
		Actions: actions,
	}
	return sdk.CreateUserPolicy(policy, token)
}

func (sdk mfSDK) Unassign(userID, groupID, token string) errors.SDKError {
	policy := Policy{
		Subject: userID,
		Object:  groupID,
	}

	return sdk.DeleteUserPolicy(policy, token)
}

func (sdk mfSDK) Connect(conn Connection, token string) errors.SDKError {
	data, err := json.Marshal(conn)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s", sdk.thingsURL, connectEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusOK)

	return sdkerr
}

func (sdk mfSDK) Disconnect(connIDs Connection, token string) errors.SDKError {
	data, err := json.Marshal(connIDs)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s", sdk.thingsURL, disconnectEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusNoContent)

	return sdkerr
}

func (sdk mfSDK) ConnectThing(thingID, channelID, token string) errors.SDKError {
	policy := Policy{
		Subject: thingID,
		Object:  channelID,
	}

	return sdk.CreateThingPolicy(policy, token)
}

func (sdk mfSDK) DisconnectThing(thingID, channelID, token string) errors.SDKError {
	policy := Policy{
		Subject: thingID,
		Object:  channelID,
	}

	return sdk.DeleteThingPolicy(policy, token)
}

func (sdk mfSDK) AuthorizeThing(accessReq AccessRequest, token string) (bool, string, errors.SDKError) {
	data, err := json.Marshal(accessReq)
	if err != nil {
		return false, "", errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s", sdk.thingsURL, channelsEndpoint, accessReq.Object, accessEndpoint)

	_, body, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return false, "", sdkerr
	}
	resp := canAccessRes{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return false, "", errors.NewSDKError(err)
	}

	return resp.Authorized, resp.ThingID, nil
}
