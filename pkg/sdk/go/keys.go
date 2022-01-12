package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/mainflux/mainflux/pkg/errors"
)

const keysEndpoint = "keys"

func (sdk mfSDK) Issue(token string, k Key) (issueKeyRes, error) {
	data, err := json.Marshal(k)
	if err != nil {
		return issueKeyRes{}, err
	}

	url := fmt.Sprintf("%s/%s", sdk.authURL, keysEndpoint)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return issueKeyRes{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return issueKeyRes{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return issueKeyRes{}, err
	}

	if resp.StatusCode != http.StatusCreated {
		return issueKeyRes{}, errors.Wrap(ErrFailedCreation, errors.New(resp.Status))
	}

	var key issueKeyRes
	if err := json.Unmarshal(body, &key); err != nil {
		return issueKeyRes{}, err
	}

	return key, nil
}

func (sdk mfSDK) Revoke(id, token string) error {
	url := fmt.Sprintf("%s/%s/%s", sdk.authURL, keysEndpoint, id)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return errors.Wrap(ErrFailedRemoval, errors.New(resp.Status))
	}

	return nil
}

func (sdk mfSDK) RetrieveKey(id, token string) (retrieveKeyRes, error) {
	url := fmt.Sprintf("%s/%s/%s", sdk.authURL, keysEndpoint, id)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return retrieveKeyRes{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return retrieveKeyRes{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return retrieveKeyRes{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return retrieveKeyRes{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var key retrieveKeyRes
	if err := json.Unmarshal(body, &key); err != nil {
		return retrieveKeyRes{}, err
	}

	return key, nil
}
