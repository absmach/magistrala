package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/mainflux/mainflux/pkg/errors"
)

const certsEndpoint = "certs"

// Cert represents certs data.
type Cert struct {
	CACert     string `json:"issuing_ca,omitempty"`
	ClientKey  string `json:"client_key,omitempty"`
	ClientCert string `json:"client_cert,omitempty"`
}

func (sdk mfSDK) IssueCert(thingID string, keyBits int, keyType, ttl, token string) (Cert, error) {
	var c Cert
	r := certReq{
		ThingID: thingID,
		KeyBits: keyBits,
		KeyType: keyType,
		TTL:     ttl,
	}
	d, err := json.Marshal(r)
	if err != nil {
		return Cert{}, err
	}

	url := fmt.Sprintf("%s/%s", sdk.certsURL, certsEndpoint)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(d))
	if err != nil {
		return Cert{}, err
	}

	res, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return Cert{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		return Cert{}, errors.Wrap(ErrFailedCreation, errors.New(res.Status))
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return Cert{}, err
	}

	if err := json.Unmarshal(body, &c); err != nil {
		return Cert{}, err
	}
	return c, nil
}

func (sdk mfSDK) RevokeCert(thingID, token string) error {
	url := fmt.Sprintf("%s/%s/%s", sdk.certsURL, certsEndpoint, thingID)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	res, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return errors.Wrap(ErrFailedRemoval, errors.New(res.Status))
	}
	return nil
}

type certReq struct {
	ThingID    string `json:"thing_id"`
	KeyBits    int    `json:"key_bits"`
	KeyType    string `json:"key_type"`
	Encryption string `json:"encryption"`
	TTL        string `json:"ttl"`
}
